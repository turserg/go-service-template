package grpctransport

import (
	"context"
	"errors"
	"log/slog"

	bookingv1 "github.com/turserg/go-service-template/gen/go/booking/v1"
	catalogv1 "github.com/turserg/go-service-template/gen/go/catalog/v1"
	ticketingv1 "github.com/turserg/go-service-template/gen/go/ticketing/v1"
	bookingusecase "github.com/turserg/go-service-template/internal/usecase/booking"
	catalogusecase "github.com/turserg/go-service-template/internal/usecase/catalog"
	ticketingusecase "github.com/turserg/go-service-template/internal/usecase/ticketing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CatalogService struct {
	catalogv1.UnimplementedCatalogServiceServer
	logger  *slog.Logger
	service *catalogusecase.Service
}

func NewCatalogService(logger *slog.Logger, service *catalogusecase.Service) *CatalogService {
	return &CatalogService{
		logger:  logger,
		service: service,
	}
}

func (s *CatalogService) ListEvents(ctx context.Context, req *catalogv1.ListEventsRequest) (*catalogv1.ListEventsResponse, error) {
	output, err := s.service.ListEvents(ctx, catalogusecase.ListEventsInput{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, mapCatalogError(err)
	}

	events := make([]*catalogv1.Event, 0, len(output.Events))
	for _, event := range output.Events {
		events = append(events, &catalogv1.Event{
			Id:             event.ID,
			VenueId:        event.VenueID,
			Title:          event.Title,
			StartsAt:       timestamppb.New(event.StartsAt),
			EndsAt:         timestamppb.New(event.EndsAt),
			Currency:       event.Currency,
			PriceFromMinor: event.PriceFromMinor,
		})
	}

	return &catalogv1.ListEventsResponse{
		Events:        events,
		NextPageToken: output.NextPageToken,
	}, nil
}

func (s *CatalogService) GetEvent(ctx context.Context, req *catalogv1.GetEventRequest) (*catalogv1.GetEventResponse, error) {
	event, err := s.service.GetEvent(ctx, catalogusecase.GetEventInput{
		EventID: req.GetEventId(),
	})
	if err != nil {
		return nil, mapCatalogError(err)
	}

	return &catalogv1.GetEventResponse{
		Event: &catalogv1.Event{
			Id:             event.ID,
			VenueId:        event.VenueID,
			Title:          event.Title,
			StartsAt:       timestamppb.New(event.StartsAt),
			EndsAt:         timestamppb.New(event.EndsAt),
			Currency:       event.Currency,
			PriceFromMinor: event.PriceFromMinor,
		},
	}, nil
}

func (s *CatalogService) GetSeatAvailability(ctx context.Context, req *catalogv1.GetSeatAvailabilityRequest) (*catalogv1.GetSeatAvailabilityResponse, error) {
	seatsOutput, err := s.service.GetSeatAvailability(ctx, catalogusecase.GetSeatAvailabilityInput{
		EventID: req.GetEventId(),
		SeatIDs: append([]string(nil), req.GetSeatIds()...),
	})
	if err != nil {
		return nil, mapCatalogError(err)
	}

	seats := make([]*catalogv1.SeatAvailability, 0, len(seatsOutput))
	for _, seat := range seatsOutput {
		seats = append(seats, &catalogv1.SeatAvailability{
			SeatId:     seat.SeatID,
			Section:    seat.Section,
			Row:        seat.Row,
			Number:     seat.Number,
			Status:     seat.Status,
			PriceMinor: seat.PriceMinor,
			Currency:   seat.Currency,
		})
	}

	return &catalogv1.GetSeatAvailabilityResponse{Seats: seats}, nil
}

func mapCatalogError(err error) error {
	switch {
	case errors.Is(err, catalogusecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, catalogusecase.ErrEventNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

type BookingService struct {
	bookingv1.UnimplementedBookingServiceServer
	logger  *slog.Logger
	service *bookingusecase.Service
}

func NewBookingService(logger *slog.Logger, service *bookingusecase.Service) *BookingService {
	return &BookingService{
		logger:  logger,
		service: service,
	}
}

func (s *BookingService) ReserveSeats(ctx context.Context, req *bookingv1.ReserveSeatsRequest) (*bookingv1.ReserveSeatsResponse, error) {
	output, err := s.service.ReserveSeats(ctx, bookingusecase.ReserveSeatsInput{
		EventID:        req.GetEventId(),
		UserID:         req.GetUserId(),
		SeatIDs:        append([]string(nil), req.GetSeatIds()...),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapBookingError(err)
	}

	return &bookingv1.ReserveSeatsResponse{
		ReservationId:    output.ReservationID,
		Status:           output.Status,
		ExpiresAt:        timestamppb.New(output.ExpiresAt),
		TotalAmountMinor: output.TotalAmountMinor,
		Currency:         output.Currency,
	}, nil
}

func (s *BookingService) CheckoutOrder(ctx context.Context, req *bookingv1.CheckoutOrderRequest) (*bookingv1.CheckoutOrderResponse, error) {
	output, err := s.service.CheckoutOrder(ctx, bookingusecase.CheckoutOrderInput{
		ReservationID:  req.GetReservationId(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapBookingError(err)
	}

	return &bookingv1.CheckoutOrderResponse{
		OrderId:              output.OrderID,
		Status:               output.Status,
		PaymentTransactionId: output.PaymentTransactionID,
	}, nil
}

func (s *BookingService) CancelOrder(ctx context.Context, req *bookingv1.CancelOrderRequest) (*bookingv1.CancelOrderResponse, error) {
	output, err := s.service.CancelOrder(ctx, bookingusecase.CancelOrderInput{
		OrderID:        req.GetOrderId(),
		UserID:         req.GetUserId(),
		Reason:         req.GetReason(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapBookingError(err)
	}

	return &bookingv1.CancelOrderResponse{
		OrderId:           output.OrderID,
		Status:            output.Status,
		ReleasedSeatCount: output.ReleasedSeatCount,
	}, nil
}

func mapBookingError(err error) error {
	switch {
	case errors.Is(err, bookingusecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, bookingusecase.ErrReservationNotFound), errors.Is(err, bookingusecase.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, bookingusecase.ErrSeatsUnavailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

type TicketService struct {
	ticketingv1.UnimplementedTicketServiceServer
	logger  *slog.Logger
	service *ticketingusecase.Service
}

func NewTicketService(logger *slog.Logger, service *ticketingusecase.Service) *TicketService {
	return &TicketService{
		logger:  logger,
		service: service,
	}
}

func (s *TicketService) IssueTickets(ctx context.Context, req *ticketingv1.IssueTicketsRequest) (*ticketingv1.IssueTicketsResponse, error) {
	output, err := s.service.IssueTickets(ctx, ticketingusecase.IssueTicketsInput{
		OrderID:        req.GetOrderId(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapTicketingError(err)
	}

	tickets := make([]*ticketingv1.Ticket, 0, len(output.Tickets))
	for _, ticket := range output.Tickets {
		tickets = append(tickets, &ticketingv1.Ticket{
			Id:            ticket.ID,
			ReservationId: ticket.ReservationID,
			Status:        ticket.Status,
		})
	}

	return &ticketingv1.IssueTicketsResponse{
		Tickets: tickets,
	}, nil
}

func (s *TicketService) GetTicketStatus(ctx context.Context, req *ticketingv1.GetTicketStatusRequest) (*ticketingv1.GetTicketStatusResponse, error) {
	ticket, err := s.service.GetTicketStatus(ctx, ticketingusecase.GetTicketStatusInput{
		TicketID: req.GetTicketId(),
	})
	if err != nil {
		return nil, mapTicketingError(err)
	}

	return &ticketingv1.GetTicketStatusResponse{
		Ticket: &ticketingv1.Ticket{
			Id:            ticket.ID,
			ReservationId: ticket.ReservationID,
			Status:        ticket.Status,
		},
	}, nil
}

func (s *TicketService) ResendTicket(ctx context.Context, req *ticketingv1.ResendTicketRequest) (*ticketingv1.ResendTicketResponse, error) {
	output, err := s.service.ResendTicket(ctx, ticketingusecase.ResendTicketInput{
		TicketID: req.GetTicketId(),
		Channel:  req.GetChannel(),
	})
	if err != nil {
		return nil, mapTicketingError(err)
	}

	return &ticketingv1.ResendTicketResponse{
		TicketId: output.TicketID,
		Status:   output.Status,
	}, nil
}

func mapTicketingError(err error) error {
	switch {
	case errors.Is(err, ticketingusecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ticketingusecase.ErrTicketNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
