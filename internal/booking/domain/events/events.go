package events

type EventItem interface {
	isDomainEvent()
}

type RoomBooked struct {
	BookingID string `json:"booking_id"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
}

func (RoomBooked) isDomainEvent() {}

type BookingConfirmed struct {
	BookingID string `json:"booking_id"`
	TxID      string `json:"tx_id"`
}

func (BookingConfirmed) isDomainEvent() {}
