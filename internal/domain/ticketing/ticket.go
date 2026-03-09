package ticketing

type TicketStatus string

const (
	TicketStatusIssued TicketStatus = "issued"
	TicketStatusSent   TicketStatus = "sent"
)

type Ticket struct {
	ID            string
	ReservationID string
	Status        TicketStatus
}
