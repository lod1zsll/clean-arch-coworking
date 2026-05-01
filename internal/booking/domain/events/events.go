package events

type EventItem interface{}

type RoomBooked struct {
	BookingID string `json:"booking_id"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
}

type BookingConfirmed struct {
	BookingID string `json:"booking_id"`
	TxID      string `json:"tx_id"`
}
