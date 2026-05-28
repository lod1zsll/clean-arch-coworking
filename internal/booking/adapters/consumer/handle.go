package consumer

import (
	"context"
	"coworking/internal/booking/application"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
)

type EventsHandler struct {
	inflight sync.WaitGroup
	logger   *slog.Logger
	done     chan struct{}
}

func NewEventsHadnler(logger *slog.Logger) application.NatsHandler {
	return &EventsHandler{
		logger: logger,
		done:   make(chan struct{}),
	}
}

func (h *EventsHandler) Handle(msg *nats.Msg) {
	h.inflight.Add(1)
	defer h.inflight.Done()

	// TODO NEED: Deduplication
	h.logger.Info("New msg", "msg_data", string(msg.Data))
}

func (h *EventsHandler) Close(ctx context.Context) error {
	go func() {
		h.inflight.Wait()
		close(h.done)
	}()

	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
