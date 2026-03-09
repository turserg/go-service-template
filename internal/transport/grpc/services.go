package grpctransport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bookingv1 "github.com/turserg/go-service-template/gen/go/booking/v1"
	catalogv1 "github.com/turserg/go-service-template/gen/go/catalog/v1"
	ticketingv1 "github.com/turserg/go-service-template/gen/go/ticketing/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InMemoryState struct {
	mu           sync.RWMutex
	reservations map[string]reservationRecord
	tickets      map[string]ticketingv1.Ticket
}

type reservationRecord struct {
	EventID string
	UserID  string
	SeatIDs []string
	Status  string
}

func NewInMemoryState() *InMemoryState {
	return &InMemoryState{
		reservations: make(map[string]reservationRecord),
		tickets:      make(map[string]ticketingv1.Ticket),
	}
}

type CatalogService struct {
	catalogv1.UnimplementedCatalogServiceServer
	logger *slog.Logger
}

func NewCatalogService(logger *slog.Logger) *CatalogService {
	return &CatalogService{logger: logger}
}

func (s *CatalogService) ListEvents(context.Context, *catalogv1.ListEventsRequest) (*catalogv1.ListEventsResponse, error) {
	now := time.Now().UTC()
	events := []*catalogv1.Event{
		{
			Id:             "evt_rock_001",
			VenueId:        "venue_moscow_01",
			Title:          "Rock Night",
			StartsAt:       timestamppb.New(now.Add(24 * time.Hour)),
			EndsAt:         timestamppb.New(now.Add(27 * time.Hour)),
			Currency:       "USD",
			PriceFromMinor: 5900,
		},
		{
			Id:             "evt_jazz_002",
			VenueId:        "venue_moscow_02",
			Title:          "Jazz Evening",
			StartsAt:       timestamppb.New(now.Add(48 * time.Hour)),
			EndsAt:         timestamppb.New(now.Add(50 * time.Hour)),
			Currency:       "USD",
			PriceFromMinor: 4200,
		},
	}

	return &catalogv1.ListEventsResponse{
		Events: events,
	}, nil
}

func (s *CatalogService) GetEvent(_ context.Context, req *catalogv1.GetEventRequest) (*catalogv1.GetEventResponse, error) {
	if req.GetEventId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	return &catalogv1.GetEventResponse{
		Event: &catalogv1.Event{
			Id:             req.GetEventId(),
			VenueId:        "venue_moscow_01",
			Title:          "Event " + req.GetEventId(),
			StartsAt:       timestamppb.New(time.Now().UTC().Add(24 * time.Hour)),
			EndsAt:         timestamppb.New(time.Now().UTC().Add(27 * time.Hour)),
			Currency:       "USD",
			PriceFromMinor: 5900,
		},
	}, nil
}

func (s *CatalogService) GetSeatAvailability(_ context.Context, req *catalogv1.GetSeatAvailabilityRequest) (*catalogv1.GetSeatAvailabilityResponse, error) {
	if req.GetEventId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	seats := make([]*catalogv1.SeatAvailability, 0, len(req.GetSeatIds()))
	for _, seatID := range req.GetSeatIds() {
		seats = append(seats, &catalogv1.SeatAvailability{
			SeatId:     seatID,
			Section:    "A",
			Row:        "1",
			Number:     seatID,
			Status:     "available",
			PriceMinor: 5900,
			Currency:   "USD",
		})
	}

	return &catalogv1.GetSeatAvailabilityResponse{Seats: seats}, nil
}

type BookingService struct {
	bookingv1.UnimplementedBookingServiceServer
	logger *slog.Logger
	state  *InMemoryState
	seq    uint64
}

func NewBookingService(logger *slog.Logger, state *InMemoryState) *BookingService {
	return &BookingService{
		logger: logger,
		state:  state,
	}
}

func (s *BookingService) ReserveSeats(_ context.Context, req *bookingv1.ReserveSeatsRequest) (*bookingv1.ReserveSeatsResponse, error) {
	if req.GetEventId() == "" || req.GetUserId() == "" || len(req.GetSeatIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id, user_id and seat_ids are required")
	}

	reservationID := fmt.Sprintf("resv_%06d", atomic.AddUint64(&s.seq, 1))
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	s.state.mu.Lock()
	s.state.reservations[reservationID] = reservationRecord{
		EventID: req.GetEventId(),
		UserID:  req.GetUserId(),
		SeatIDs: append([]string(nil), req.GetSeatIds()...),
		Status:  "pending",
	}
	s.state.mu.Unlock()

	return &bookingv1.ReserveSeatsResponse{
		ReservationId:    reservationID,
		Status:           "pending",
		ExpiresAt:        timestamppb.New(expiresAt),
		TotalAmountMinor: int64(len(req.GetSeatIds())) * 5900,
		Currency:         "USD",
	}, nil
}

func (s *BookingService) CheckoutOrder(_ context.Context, req *bookingv1.CheckoutOrderRequest) (*bookingv1.CheckoutOrderResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	s.state.mu.Lock()
	record, ok := s.state.reservations[req.GetReservationId()]
	if !ok {
		s.state.mu.Unlock()
		return nil, status.Error(codes.NotFound, "reservation not found")
	}
	record.Status = "confirmed"
	s.state.reservations[req.GetReservationId()] = record
	s.state.mu.Unlock()

	orderID := "ord_" + req.GetReservationId()
	return &bookingv1.CheckoutOrderResponse{
		OrderId:              orderID,
		Status:               "confirmed",
		PaymentTransactionId: "pay_" + req.GetReservationId(),
	}, nil
}

func (s *BookingService) CancelOrder(_ context.Context, req *bookingv1.CancelOrderRequest) (*bookingv1.CancelOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	reservationID := strings.TrimPrefix(req.GetOrderId(), "ord_")

	s.state.mu.Lock()
	record, ok := s.state.reservations[reservationID]
	if !ok {
		s.state.mu.Unlock()
		return nil, status.Error(codes.NotFound, "reservation not found")
	}
	record.Status = "canceled"
	s.state.reservations[reservationID] = record
	released := uint32(len(record.SeatIDs))
	s.state.mu.Unlock()

	return &bookingv1.CancelOrderResponse{
		OrderId:           req.GetOrderId(),
		Status:            "canceled",
		ReleasedSeatCount: released,
	}, nil
}

type TicketService struct {
	ticketingv1.UnimplementedTicketServiceServer
	logger *slog.Logger
	state  *InMemoryState
	seq    uint64
}

func NewTicketService(logger *slog.Logger, state *InMemoryState) *TicketService {
	return &TicketService{
		logger: logger,
		state:  state,
	}
}

func (s *TicketService) IssueTickets(_ context.Context, req *ticketingv1.IssueTicketsRequest) (*ticketingv1.IssueTicketsResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	reservationID := strings.TrimPrefix(req.GetOrderId(), "ord_")
	if reservationID == req.GetOrderId() {
		return nil, status.Error(codes.InvalidArgument, "order_id format is invalid")
	}

	s.state.mu.RLock()
	record, ok := s.state.reservations[reservationID]
	s.state.mu.RUnlock()
	if !ok {
		return nil, status.Error(codes.NotFound, "reservation not found")
	}

	tickets := make([]*ticketingv1.Ticket, 0, len(record.SeatIDs))
	s.state.mu.Lock()
	for range record.SeatIDs {
		ticketID := fmt.Sprintf("tkt_%06d", atomic.AddUint64(&s.seq, 1))
		ticket := ticketingv1.Ticket{
			Id:            ticketID,
			ReservationId: reservationID,
			Status:        "issued",
		}
		s.state.tickets[ticketID] = ticket
		tickets = append(tickets, &ticketingv1.Ticket{
			Id:            ticket.Id,
			ReservationId: ticket.ReservationId,
			Status:        ticket.Status,
		})
	}
	s.state.mu.Unlock()

	return &ticketingv1.IssueTicketsResponse{Tickets: tickets}, nil
}

func (s *TicketService) GetTicketStatus(_ context.Context, req *ticketingv1.GetTicketStatusRequest) (*ticketingv1.GetTicketStatusResponse, error) {
	if req.GetTicketId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}

	s.state.mu.RLock()
	ticket, ok := s.state.tickets[req.GetTicketId()]
	s.state.mu.RUnlock()
	if !ok {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}

	return &ticketingv1.GetTicketStatusResponse{
		Ticket: &ticketingv1.Ticket{
			Id:            ticket.Id,
			ReservationId: ticket.ReservationId,
			Status:        ticket.Status,
		},
	}, nil
}

func (s *TicketService) ResendTicket(_ context.Context, req *ticketingv1.ResendTicketRequest) (*ticketingv1.ResendTicketResponse, error) {
	if req.GetTicketId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}

	s.state.mu.Lock()
	ticket, ok := s.state.tickets[req.GetTicketId()]
	if !ok {
		s.state.mu.Unlock()
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	ticket.Status = "resent"
	s.state.tickets[req.GetTicketId()] = ticket
	s.state.mu.Unlock()

	return &ticketingv1.ResendTicketResponse{
		TicketId: req.GetTicketId(),
		Status:   "resent",
	}, nil
}
