package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	ticketingusecase "github.com/turserg/go-service-template/internal/usecase/ticketing"
)

type TicketRepository struct {
	mu            sync.RWMutex
	seq           uint64
	tickets       map[string]ticketingusecase.Ticket
	ticketByOrder map[string]string
}

func NewTicketRepository() *TicketRepository {
	return &TicketRepository{
		tickets:       make(map[string]ticketingusecase.Ticket),
		ticketByOrder: make(map[string]string),
	}
}

func (r *TicketRepository) IssueTickets(_ context.Context, input ticketingusecase.IssueTicketsInput) (ticketingusecase.IssueTicketsOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingTicketID, ok := r.ticketByOrder[input.OrderID]; ok {
		existingTicket := r.tickets[existingTicketID]
		return ticketingusecase.IssueTicketsOutput{
			Tickets: []ticketingusecase.Ticket{existingTicket},
		}, nil
	}

	ticketID := fmt.Sprintf("tkt_%06d", atomic.AddUint64(&r.seq, 1))
	ticket := ticketingusecase.Ticket{
		ID:            ticketID,
		ReservationID: strings.TrimPrefix(input.OrderID, "ord_"),
		Status:        "issued",
	}

	r.tickets[ticketID] = ticket
	r.ticketByOrder[input.OrderID] = ticketID

	return ticketingusecase.IssueTicketsOutput{
		Tickets: []ticketingusecase.Ticket{ticket},
	}, nil
}

func (r *TicketRepository) GetTicketStatus(_ context.Context, input ticketingusecase.GetTicketStatusInput) (ticketingusecase.Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ticket, ok := r.tickets[input.TicketID]
	if !ok {
		return ticketingusecase.Ticket{}, ticketingusecase.ErrTicketNotFound
	}
	return ticket, nil
}

func (r *TicketRepository) ResendTicket(_ context.Context, input ticketingusecase.ResendTicketInput) (ticketingusecase.ResendTicketOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ticket, ok := r.tickets[input.TicketID]
	if !ok {
		return ticketingusecase.ResendTicketOutput{}, ticketingusecase.ErrTicketNotFound
	}
	ticket.Status = "resent"
	r.tickets[input.TicketID] = ticket

	return ticketingusecase.ResendTicketOutput{
		TicketID: input.TicketID,
		Status:   ticket.Status,
	}, nil
}
