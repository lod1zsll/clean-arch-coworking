package outbox

import (
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type EventStatus string

const (
	EventStatusNew  EventStatus = "new"
	EventStatusDone EventStatus = "done"
)

type EventType string

const (
	EventTypeUnknown EventType = "unknown"
	EventTypeBooking EventType = "room_booked"
	EventTypeConfirm EventType = "booking_confirmed"
)

type EventMsg struct {
	UUID uuid.UUID
	Type string
	Data json.RawMessage
}

type Event struct {
	EventMsg
	Status     EventStatus
	CreatedAt  time.Time
	ReservedTo *time.Time
}
