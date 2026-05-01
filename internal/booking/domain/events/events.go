package events

type EventItem interface{}

type RoomBooked struct {
	BookingID string
	RoomID    string
	UserID    string
}

type BookingConfirmed struct {
	BookingID string
	TxID      string
}
