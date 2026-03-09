package memory

import (
	"context"
	"sync"

	bookingusecase "github.com/turserg/go-service-template/internal/usecase/booking"
)

type seatRecord struct {
	priceMinor int64
	status     string
	reservedBy string
}

type reservationRecord struct {
	eventID string
	userID  string
	seatIDs []string
	status  string
}

type orderRecord struct {
	reservationID string
	userID        string
	status        string
}

type BookingRepository struct {
	mu              sync.Mutex
	currencyByEvent map[string]string
	seatsByEvent    map[string]map[string]*seatRecord
	reservations    map[string]reservationRecord
	orders          map[string]orderRecord
}

func NewBookingRepository() *BookingRepository {
	return &BookingRepository{
		currencyByEvent: map[string]string{
			"evt_rock_001": "USD",
			"evt_jazz_002": "USD",
		},
		seatsByEvent: map[string]map[string]*seatRecord{
			"evt_rock_001": buildSeats(50, 5900),
			"evt_jazz_002": buildSeats(40, 4200),
		},
		reservations: make(map[string]reservationRecord),
		orders:       make(map[string]orderRecord),
	}
}

func buildSeats(count int, priceMinor int64) map[string]*seatRecord {
	out := make(map[string]*seatRecord, count)
	for i := 1; i <= count; i++ {
		seatID := "A-" + itoa(i)
		out[seatID] = &seatRecord{
			priceMinor: priceMinor,
			status:     "available",
		}
	}
	return out
}

func (r *BookingRepository) ReserveSeats(_ context.Context, input bookingusecase.ReserveSeatsInput) (bookingusecase.ReserveSeatsOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	eventSeats, ok := r.seatsByEvent[input.EventID]
	if !ok {
		return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
	}

	var total int64
	for _, seatID := range input.SeatIDs {
		seat, exists := eventSeats[seatID]
		if !exists || seat.status != "available" {
			return bookingusecase.ReserveSeatsOutput{}, bookingusecase.ErrSeatsUnavailable
		}
		total += seat.priceMinor
	}

	for _, seatID := range input.SeatIDs {
		seat := eventSeats[seatID]
		seat.status = "reserved"
		seat.reservedBy = input.ReservationID
	}

	r.reservations[input.ReservationID] = reservationRecord{
		eventID: input.EventID,
		userID:  input.UserID,
		seatIDs: append([]string(nil), input.SeatIDs...),
		status:  "pending",
	}

	return bookingusecase.ReserveSeatsOutput{
		ReservationID:    input.ReservationID,
		Status:           "pending",
		ExpiresAt:        input.ExpiresAt,
		TotalAmountMinor: total,
		Currency:         r.currencyByEvent[input.EventID],
	}, nil
}

func (r *BookingRepository) CheckoutOrder(_ context.Context, input bookingusecase.CheckoutOrderInput) (bookingusecase.CheckoutOrderOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reservation, ok := r.reservations[input.ReservationID]
	if !ok {
		return bookingusecase.CheckoutOrderOutput{}, bookingusecase.ErrReservationNotFound
	}

	for orderID, order := range r.orders {
		if order.reservationID == input.ReservationID {
			return bookingusecase.CheckoutOrderOutput{
				OrderID:              orderID,
				Status:               order.status,
				PaymentTransactionID: input.PaymentTransaction,
			}, nil
		}
	}

	reservation.status = "confirmed"
	r.reservations[input.ReservationID] = reservation

	seats := r.seatsByEvent[reservation.eventID]
	for _, seatID := range reservation.seatIDs {
		seat := seats[seatID]
		seat.status = "sold"
	}

	r.orders[input.OrderID] = orderRecord{
		reservationID: input.ReservationID,
		userID:        reservation.userID,
		status:        "confirmed",
	}

	return bookingusecase.CheckoutOrderOutput{
		OrderID:              input.OrderID,
		Status:               "confirmed",
		PaymentTransactionID: input.PaymentTransaction,
	}, nil
}

func (r *BookingRepository) CancelOrder(_ context.Context, input bookingusecase.CancelOrderInput) (bookingusecase.CancelOrderOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[input.OrderID]
	if !ok {
		return bookingusecase.CancelOrderOutput{}, bookingusecase.ErrOrderNotFound
	}
	reservation, ok := r.reservations[order.reservationID]
	if !ok {
		return bookingusecase.CancelOrderOutput{}, bookingusecase.ErrReservationNotFound
	}

	order.status = "canceled"
	r.orders[input.OrderID] = order
	reservation.status = "canceled"
	r.reservations[order.reservationID] = reservation

	var released uint32
	seats := r.seatsByEvent[reservation.eventID]
	for _, seatID := range reservation.seatIDs {
		seat := seats[seatID]
		if seat.status == "reserved" || seat.status == "sold" {
			seat.status = "available"
			seat.reservedBy = ""
			released++
		}
	}

	return bookingusecase.CancelOrderOutput{
		OrderID:           input.OrderID,
		Status:            "canceled",
		ReleasedSeatCount: released,
	}, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [16]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
