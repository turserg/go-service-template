package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrEventNotFound = errors.New("event not found")
)

type Repository interface {
	ListEvents(ctx context.Context, input ListEventsInput) (ListEventsOutput, error)
	GetEvent(ctx context.Context, input GetEventInput) (Event, error)
	GetSeatAvailability(ctx context.Context, input GetSeatAvailabilityInput) ([]SeatAvailability, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type ListEventsInput struct {
	PageSize  uint32
	PageToken string
}

type ListEventsOutput struct {
	Events        []Event
	NextPageToken string
}

type GetEventInput struct {
	EventID string
}

type GetSeatAvailabilityInput struct {
	EventID string
	SeatIDs []string
}

type Event struct {
	ID             string
	VenueID        string
	Title          string
	StartsAt       time.Time
	EndsAt         time.Time
	Currency       string
	PriceFromMinor int64
}

type SeatAvailability struct {
	SeatID     string
	Section    string
	Row        string
	Number     string
	Status     string
	PriceMinor int64
	Currency   string
}

func (s *Service) ListEvents(ctx context.Context, input ListEventsInput) (ListEventsOutput, error) {
	if input.PageSize == 0 {
		input.PageSize = 20
	}
	return s.repo.ListEvents(ctx, input)
}

func (s *Service) GetEvent(ctx context.Context, input GetEventInput) (Event, error) {
	if input.EventID == "" {
		return Event{}, fmt.Errorf("%w: event_id is required", ErrInvalidInput)
	}
	return s.repo.GetEvent(ctx, input)
}

func (s *Service) GetSeatAvailability(ctx context.Context, input GetSeatAvailabilityInput) ([]SeatAvailability, error) {
	if input.EventID == "" {
		return nil, fmt.Errorf("%w: event_id is required", ErrInvalidInput)
	}
	return s.repo.GetSeatAvailability(ctx, input)
}
