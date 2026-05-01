package outbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"coworking/internal/booking/application"
	"coworking/internal/booking/application/outbox"
	"coworking/internal/booking/domain/events"
)

type EventStore struct {
	mu     sync.RWMutex
	events []outbox.Event
	bus    application.EventBus
}

func NewEventStore(bus application.EventBus) application.EventStore {
	return &EventStore{
		events: make([]outbox.Event, 0),
		bus:    bus,
	}
}

func (s *EventStore) SaveEvents(ctx context.Context, eventItems []events.EventItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, event := range eventItems {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		outboxEvent := outbox.Event{
			ID:        uuid.New(),
			EventType: getEventType(event),
			EventData: data,
			Status:    outbox.EventStatusNew,
			CreatedAt: time.Now(),
		}

		s.events = append(s.events, outboxEvent)
	}

	// TODO: replace goroutine-based publish with a reliable polling publisher.
	// Current approach may lose events if the process crashes before publishing.
	go s.publishPendingEvents(ctx)

	return nil
}

func (s *EventStore) publishPendingEvents(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var domainEvents []events.EventItem
	for i, event := range s.events {
		if event.Status == outbox.EventStatusNew {
			var domainEvent events.EventItem
			switch event.EventType {
			case "RoomBooked":
				var e events.RoomBooked
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e
				}
			case "BookingConfirmed":
				var e events.BookingConfirmed
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e
				}
			}

			if domainEvent != nil {
				domainEvents = append(domainEvents, domainEvent)
				s.events[i].Status = outbox.EventStatusDone
			}
		}
	}

	if len(domainEvents) > 0 {
		_ = s.bus.Publish(ctx, domainEvents)
	}
}

func getEventType(event events.EventItem) string {
	switch event.(type) {
	case events.RoomBooked:
		return "RoomBooked"
	case events.BookingConfirmed:
		return "BookingConfirmed"
	default:
		return "Unknown"
	}
}
