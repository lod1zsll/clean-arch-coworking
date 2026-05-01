package memory

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID
	Type       string
	Data       json.RawMessage
	Status     string
	CreatedAt  time.Time
	ReservedTo *time.Time
}
