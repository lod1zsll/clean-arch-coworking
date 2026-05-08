package dummy

import (
	"context"

	"coworking/internal/booking/domain/events"
)

// TODO: implement real event bus (e.g. NATS, RabbitMQ) for cross-service communication.
type EventBus struct{}

func (EventBus) Publish(ctx context.Context, events []events.EventItem) error {
	return nil
}

func NewEventBus() EventBus {
	return EventBus{}
}
