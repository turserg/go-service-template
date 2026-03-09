package ticketing

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrTicketNotFound = errors.New("ticket not found")
)

type Repository interface {
	IssueTickets(ctx context.Context, input IssueTicketsInput) (IssueTicketsOutput, error)
	GetTicketStatus(ctx context.Context, input GetTicketStatusInput) (Ticket, error)
	ResendTicket(ctx context.Context, input ResendTicketInput) (ResendTicketOutput, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type IssueTicketsInput struct {
	OrderID        string
	IdempotencyKey string
}

type IssueTicketsOutput struct {
	Tickets []Ticket
}

type GetTicketStatusInput struct {
	TicketID string
}

type ResendTicketInput struct {
	TicketID string
	Channel  string
}

type ResendTicketOutput struct {
	TicketID string
	Status   string
}

type Ticket struct {
	ID            string
	ReservationID string
	Status        string
}

func (s *Service) IssueTickets(ctx context.Context, input IssueTicketsInput) (IssueTicketsOutput, error) {
	if input.OrderID == "" {
		return IssueTicketsOutput{}, fmt.Errorf("%w: order_id is required", ErrInvalidInput)
	}
	return s.repo.IssueTickets(ctx, input)
}

func (s *Service) GetTicketStatus(ctx context.Context, input GetTicketStatusInput) (Ticket, error) {
	if input.TicketID == "" {
		return Ticket{}, fmt.Errorf("%w: ticket_id is required", ErrInvalidInput)
	}
	return s.repo.GetTicketStatus(ctx, input)
}

func (s *Service) ResendTicket(ctx context.Context, input ResendTicketInput) (ResendTicketOutput, error) {
	if input.TicketID == "" {
		return ResendTicketOutput{}, fmt.Errorf("%w: ticket_id is required", ErrInvalidInput)
	}
	return s.repo.ResendTicket(ctx, input)
}
