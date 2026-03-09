package booking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrOrderNotFound       = errors.New("order not found")
	ErrSeatsUnavailable    = errors.New("one or more seats are unavailable")
)

type Repository interface {
	ReserveSeats(ctx context.Context, input ReserveSeatsInput) (ReserveSeatsOutput, error)
	CheckoutOrder(ctx context.Context, input CheckoutOrderInput) (CheckoutOrderOutput, error)
	CancelOrder(ctx context.Context, input CancelOrderInput) (CancelOrderOutput, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type ReserveSeatsInput struct {
	ReservationID  string
	EventID        string
	UserID         string
	SeatIDs        []string
	IdempotencyKey string
	ExpiresAt      time.Time
	Currency       string
}

type ReserveSeatsOutput struct {
	ReservationID    string
	Status           string
	ExpiresAt        time.Time
	TotalAmountMinor int64
	Currency         string
}

type CheckoutOrderInput struct {
	OrderID            string
	ReservationID      string
	PaymentTransaction string
	IdempotencyKey     string
}

type CheckoutOrderOutput struct {
	OrderID              string
	Status               string
	PaymentTransactionID string
}

type CancelOrderInput struct {
	OrderID        string
	UserID         string
	Reason         string
	IdempotencyKey string
}

type CancelOrderOutput struct {
	OrderID           string
	Status            string
	ReleasedSeatCount uint32
}

func (s *Service) ReserveSeats(ctx context.Context, input ReserveSeatsInput) (ReserveSeatsOutput, error) {
	if input.EventID == "" || input.UserID == "" || len(input.SeatIDs) == 0 {
		return ReserveSeatsOutput{}, fmt.Errorf("%w: event_id, user_id and seat_ids are required", ErrInvalidInput)
	}
	if input.ReservationID == "" {
		input.ReservationID = prefixedID("resv")
	}
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = time.Now().UTC().Add(15 * time.Minute)
	}
	if input.Currency == "" {
		input.Currency = "USD"
	}

	return s.repo.ReserveSeats(ctx, input)
}

func (s *Service) CheckoutOrder(ctx context.Context, input CheckoutOrderInput) (CheckoutOrderOutput, error) {
	if input.ReservationID == "" {
		return CheckoutOrderOutput{}, fmt.Errorf("%w: reservation_id is required", ErrInvalidInput)
	}
	if input.OrderID == "" {
		input.OrderID = prefixedID("ord")
	}
	if input.PaymentTransaction == "" {
		input.PaymentTransaction = prefixedID("pay")
	}

	return s.repo.CheckoutOrder(ctx, input)
}

func (s *Service) CancelOrder(ctx context.Context, input CancelOrderInput) (CancelOrderOutput, error) {
	if input.OrderID == "" {
		return CancelOrderOutput{}, fmt.Errorf("%w: order_id is required", ErrInvalidInput)
	}
	return s.repo.CancelOrder(ctx, input)
}

func prefixedID(prefix string) string {
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	return prefix + "_" + id[:12]
}
