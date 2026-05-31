package domain

// TODO: think about marshal/umarshal test's
type EventItem interface {
	isDomainEvent()
}

type EventRoomBooked struct {
	BookingID string
	RoomID    string
	UserID    string
}

func (EventRoomBooked) isDomainEvent() {}

type EventBookingConfirmed struct {
	BookingID string
	TxID      string
}

func (EventBookingConfirmed) isDomainEvent() {}
