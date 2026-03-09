package booking

type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusConfirmed ReservationStatus = "confirmed"
	ReservationStatusCanceled  ReservationStatus = "canceled"
)

type Reservation struct {
	ID      string
	EventID string
	UserID  string
	Status  ReservationStatus
}
