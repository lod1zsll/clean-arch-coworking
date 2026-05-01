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

type EventMsg struct {
	UUID uuid.UUID       `db:"event_id" json:"event_id" `
	Type string          `db:"event_type" json:"event_type"`
	Data json.RawMessage `db:"event_data" json:"event_data"`
}

type Event struct {
	EventMsg
	Status     EventStatus `db:"event_status" json:"event_status"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	ReservedTo *time.Time  `db:"reserved_to" json:"reserved_to,omitempty" `
}
