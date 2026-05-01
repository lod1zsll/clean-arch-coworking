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

type Event struct {
	ID         uuid.UUID
	EventType  string
	EventData  json.RawMessage
	Status     EventStatus
	CreatedAt  time.Time
	ReservedTo *time.Time
}
