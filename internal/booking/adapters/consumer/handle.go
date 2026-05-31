package consumer

import (
	"context"
	"coworking/internal/booking/application"
	"coworking/pkg/natser"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type EventsHandler struct {
	logger *slog.Logger
	rdb    *redis.Client

	topicIn  string
	dedupTTL time.Duration

	inflight sync.WaitGroup
	started  atomic.Bool
	done     chan struct{}
}

var ErrHandlerAlreadyStarted = errors.New("handler already started")

func NewEventsHandler(logger *slog.Logger, rdb *redis.Client, topicIn string, dedupTTL time.Duration) application.NatsHandler {
	return &EventsHandler{
		logger:   logger,
		rdb:      rdb,
		topicIn:  topicIn,
		dedupTTL: dedupTTL,
		done:     make(chan struct{}),
	}
}

func (h *EventsHandler) Handle(msg *nats.Msg) {
	ctx := context.Background()

	h.inflight.Add(1)
	defer h.inflight.Done()

	msgIdHeader, _ := msg.Header["Nats-Msg-Id"]
	h.logger.Debug("msg headers", "headers", msg.Header)
	if len(msgIdHeader) == 0 {
		h.logger.Error("Failed to handle message; Header 'Nats-Msg-Id' is empty")
		return
	}
	if len(msgIdHeader) != 1 {
		h.logger.Error("Failed to handle message; Header 'Nats-Msg-Id' is not single item array")
		return
	}

	msgId := msgIdHeader[0]

	_, err := uuid.Parse(msgId)
	if err != nil {
		h.logger.Error("Failed to handle message; 'Nats-Msg-Id' is not valid uuid")
		return
	}

	set, err := h.rdb.SetNX(ctx, fmt.Sprintf("outbox:dedup:%s:%s", h.topicIn, msgId), 1, h.dedupTTL).Result()
	if err != nil {
		h.logger.Error("Failed to handle message; Redis SetNX", "error", err)
		return
	}
	if !set {
		h.logger.Warn("Skip already processed message", "msg_id", msgId)
		return
	}

	h.logger.Info("New msg", "msg_data", string(msg.Data))
}

func (h *EventsHandler) Start(natsWrapper *natser.NatsWrapper) error {
	if h.started.Load() {
		return ErrHandlerAlreadyStarted
	}

	sub, err := natsWrapper.GetConn().Subscribe(h.topicIn, h.Handle)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	h.started.Store(true)

	return nil
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
