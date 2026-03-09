package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	bookingusecase "github.com/turserg/go-service-template/internal/usecase/booking"
)

type BookingRepository struct {
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

func (r *BookingRepository) ReserveSeats(ctx context.Context, input bookingusecase.ReserveSeatsInput) (bookingusecase.ReserveSeatsOutput, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("begin reserve transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	currency := "USD"
	if err = tx.QueryRow(ctx, `SELECT currency FROM events WHERE id = $1`, input.EventID).Scan(&currency); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
		}
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("select event: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT seat_id, price_minor, status
		FROM seat_inventory
		WHERE event_id = $1 AND seat_id = ANY($2)
		FOR UPDATE
	`, input.EventID, input.SeatIDs)
	if err != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("select seats for update: %w", err)
	}
	defer rows.Close()

	locked := make(map[string]struct{}, len(input.SeatIDs))
	var total int64
	for rows.Next() {
		var (
			seatID     string
			priceMinor int64
			status     string
		)
		if scanErr := rows.Scan(&seatID, &priceMinor, &status); scanErr != nil {
			return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("scan seat row: %w", scanErr)
		}
		locked[seatID] = struct{}{}
		if status != "available" {
			return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
		}
		total += priceMinor
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("iterate seat rows: %w", rowsErr)
	}
	if len(locked) != len(input.SeatIDs) {
		return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO reservations (
			id, event_id, user_id, status, expires_at, total_amount_minor, currency, idempotency_key
		) VALUES ($1, $2, $3, 'pending', $4, $5, $6, NULLIF($7, ''))
	`, input.ReservationID, input.EventID, input.UserID, input.ExpiresAt, total, currency, input.IdempotencyKey); err != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("insert reservation: %w", err)
	}

	for _, seatID := range input.SeatIDs {
		if _, err = tx.Exec(ctx, `
			INSERT INTO reservation_seats (reservation_id, event_id, seat_id)
			VALUES ($1, $2, $3)
		`, input.ReservationID, input.EventID, seatID); err != nil {
			return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("insert reservation seat: %w", err)
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE seat_inventory
		SET status = 'reserved', reserved_by = $1
		WHERE event_id = $2 AND seat_id = ANY($3)
	`, input.ReservationID, input.EventID, input.SeatIDs)
	if err != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("update seat inventory reserved: %w", err)
	}
	if int(tag.RowsAffected()) != len(input.SeatIDs) {
		return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
	}

	if err = tx.Commit(ctx); err != nil {
		return bookingusecase.ReserveSeatsOutput{}, fmt.Errorf("commit reserve transaction: %w", err)
	}

	return bookingusecase.ReserveSeatsOutput{
		ReservationID:    input.ReservationID,
		Status:           "pending",
		ExpiresAt:        input.ExpiresAt,
		TotalAmountMinor: total,
		Currency:         currency,
	}, nil
}

func (r *BookingRepository) CheckoutOrder(ctx context.Context, input bookingusecase.CheckoutOrderInput) (bookingusecase.CheckoutOrderOutput, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("begin checkout transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	var (
		reservationStatus string
		userID            string
	)
	if err = tx.QueryRow(ctx, `
		SELECT status, user_id
		FROM reservations
		WHERE id = $1
		FOR UPDATE
	`, input.ReservationID).Scan(&reservationStatus, &userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bookingusecase.CheckoutOrderOutput{}, bookingusecase.ErrReservationNotFound
		}
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("select reservation for checkout: %w", err)
	}

	if reservationStatus == "confirmed" {
		var existingOrderID string
		if err = tx.QueryRow(ctx, `SELECT id FROM orders WHERE reservation_id = $1`, input.ReservationID).Scan(&existingOrderID); err != nil {
			return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("select existing order: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("commit checkout transaction (idempotent): %w", err)
		}
		return bookingusecase.CheckoutOrderOutput{
			OrderID:              existingOrderID,
			Status:               "confirmed",
			PaymentTransactionID: input.PaymentTransaction,
		}, nil
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO orders (id, reservation_id, user_id, status)
		VALUES ($1, $2, $3, 'confirmed')
		ON CONFLICT (reservation_id) DO NOTHING
	`, input.OrderID, input.ReservationID, userID); err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("insert order: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO payment_attempts (order_id, transaction_id, status)
		VALUES ($1, $2, 'captured')
	`, input.OrderID, input.PaymentTransaction); err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("insert payment attempt: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		UPDATE reservations
		SET status = 'confirmed'
		WHERE id = $1
	`, input.ReservationID); err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("update reservation confirmed: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		UPDATE seat_inventory
		SET status = 'sold'
		WHERE reserved_by = $1
	`, input.ReservationID); err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("update seat inventory sold: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return bookingusecase.CheckoutOrderOutput{}, fmt.Errorf("commit checkout transaction: %w", err)
	}

	return bookingusecase.CheckoutOrderOutput{
		OrderID:              input.OrderID,
		Status:               "confirmed",
		PaymentTransactionID: input.PaymentTransaction,
	}, nil
}

func (r *BookingRepository) CancelOrder(ctx context.Context, input bookingusecase.CancelOrderInput) (bookingusecase.CancelOrderOutput, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("begin cancel transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	var (
		reservationID string
		orderStatus   string
	)
	if err = tx.QueryRow(ctx, `
		SELECT reservation_id, status
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, input.OrderID).Scan(&reservationID, &orderStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bookingusecase.CancelOrderOutput{}, bookingusecase.ErrOrderNotFound
		}
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("select order for cancel: %w", err)
	}

	if orderStatus != "canceled" {
		if _, err = tx.Exec(ctx, `UPDATE orders SET status = 'canceled' WHERE id = $1`, input.OrderID); err != nil {
			return bookingusecase.CancelOrderOutput{}, fmt.Errorf("update order canceled: %w", err)
		}
	}

	if _, err = tx.Exec(ctx, `UPDATE reservations SET status = 'canceled' WHERE id = $1`, reservationID); err != nil {
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("update reservation canceled: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE seat_inventory
		SET status = 'available', reserved_by = NULL
		WHERE reserved_by = $1
	`, reservationID)
	if err != nil {
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("update seat inventory available: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO payment_attempts (order_id, transaction_id, status)
		VALUES ($1, COALESCE(NULLIF($2, ''), $1 || '_refund'), 'refunded')
	`, input.OrderID, input.IdempotencyKey); err != nil {
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("insert refund payment attempt: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return bookingusecase.CancelOrderOutput{}, fmt.Errorf("commit cancel transaction: %w", err)
	}

	return bookingusecase.CancelOrderOutput{
		OrderID:           input.OrderID,
		Status:            "canceled",
		ReleasedSeatCount: uint32(tag.RowsAffected()),
	}, nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
