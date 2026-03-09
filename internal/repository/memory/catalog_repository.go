package memory

import (
	"context"
	"strconv"
	"time"

	catalogusecase "github.com/turserg/go-service-template/internal/usecase/catalog"
)

type CatalogRepository struct {
	events       []catalogusecase.Event
	seatsByEvent map[string]map[string]catalogusecase.SeatAvailability
}

func NewCatalogRepository() *CatalogRepository {
	now := time.Now().UTC()
	events := []catalogusecase.Event{
		{
			ID:             "evt_rock_001",
			VenueID:        "venue_moscow_01",
			Title:          "Rock Night",
			StartsAt:       now.Add(24 * time.Hour),
			EndsAt:         now.Add(27 * time.Hour),
			Currency:       "USD",
			PriceFromMinor: 5900,
		},
		{
			ID:             "evt_jazz_002",
			VenueID:        "venue_moscow_02",
			Title:          "Jazz Evening",
			StartsAt:       now.Add(48 * time.Hour),
			EndsAt:         now.Add(50 * time.Hour),
			Currency:       "USD",
			PriceFromMinor: 4200,
		},
	}

	return &CatalogRepository{
		events: events,
		seatsByEvent: map[string]map[string]catalogusecase.SeatAvailability{
			"evt_rock_001": buildSeatAvailability(50, 5900),
			"evt_jazz_002": buildSeatAvailability(40, 4200),
		},
	}
}

func buildSeatAvailability(count int, priceMinor int64) map[string]catalogusecase.SeatAvailability {
	out := make(map[string]catalogusecase.SeatAvailability, count)
	for i := 1; i <= count; i++ {
		seatID := "A-" + itoa(i)
		out[seatID] = catalogusecase.SeatAvailability{
			SeatID:     seatID,
			Section:    "A",
			Row:        "1",
			Number:     strconv.Itoa(i),
			Status:     "available",
			PriceMinor: priceMinor,
			Currency:   "USD",
		}
	}
	return out
}

func (r *CatalogRepository) ListEvents(_ context.Context, input catalogusecase.ListEventsInput) (catalogusecase.ListEventsOutput, error) {
	start := 0
	if input.PageToken != "" {
		pageStart, err := strconv.Atoi(input.PageToken)
		if err == nil && pageStart >= 0 && pageStart < len(r.events) {
			start = pageStart
		}
	}

	end := start + int(input.PageSize)
	if end > len(r.events) {
		end = len(r.events)
	}

	events := make([]catalogusecase.Event, 0, end-start)
	for _, event := range r.events[start:end] {
		events = append(events, event)
	}

	nextPageToken := ""
	if end < len(r.events) {
		nextPageToken = strconv.Itoa(end)
	}

	return catalogusecase.ListEventsOutput{
		Events:        events,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *CatalogRepository) GetEvent(_ context.Context, input catalogusecase.GetEventInput) (catalogusecase.Event, error) {
	for _, event := range r.events {
		if event.ID == input.EventID {
			return event, nil
		}
	}
	return catalogusecase.Event{}, catalogusecase.ErrEventNotFound
}

func (r *CatalogRepository) GetSeatAvailability(_ context.Context, input catalogusecase.GetSeatAvailabilityInput) ([]catalogusecase.SeatAvailability, error) {
	eventSeats, ok := r.seatsByEvent[input.EventID]
	if !ok {
		return nil, catalogusecase.ErrEventNotFound
	}

	if len(input.SeatIDs) == 0 {
		seatIDs := make([]string, 0, len(eventSeats))
		for seatID := range eventSeats {
			seatIDs = append(seatIDs, seatID)
		}
		input.SeatIDs = seatIDs
	}

	result := make([]catalogusecase.SeatAvailability, 0, len(input.SeatIDs))
	for _, seatID := range input.SeatIDs {
		seat, exists := eventSeats[seatID]
		if !exists {
			continue
		}
		result = append(result, seat)
	}

	return result, nil
}
