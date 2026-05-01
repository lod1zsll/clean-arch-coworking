package outbox

import (
	"encoding/json"
	"time"

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

type Event struct {
	ID         uuid.UUID
	EventType  string
	EventData  json.RawMessage
	Status     EventStatus
	CreatedAt  time.Time
	ReservedTo *time.Time
}
