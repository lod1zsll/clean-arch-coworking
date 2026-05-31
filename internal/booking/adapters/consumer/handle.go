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
	cancel   context.CancelFunc

	sub *nats.Subscription
}

var ErrHandlerAlreadyStarted = errors.New("handler already started")

func NewEventsHandler(logger *slog.Logger, rdb *redis.Client, topicIn string, dedupTTL time.Duration) application.NatsHandler {
	if dedupTTL < 1 {
		dedupTTL = time.Hour * 24
	}

	return &EventsHandler{
		logger:   logger,
		rdb:      rdb,
		topicIn:  topicIn,
		dedupTTL: dedupTTL,
	}
}

func (h *EventsHandler) Start(ctx context.Context, natsWrapper *natser.NatsWrapper) error {
	if h.started.Load() {
		return ErrHandlerAlreadyStarted
	}

	sub, err := natsWrapper.GetConn().Subscribe(h.topicIn, func(msg *nats.Msg) {
		h.Handle(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	h.sub = sub
	h.cancel = cancel
	h.done = make(chan struct{})
	h.started.Store(true)

	return nil
}

func (h *EventsHandler) Close(ctx context.Context) error {
	if !h.started.Load() {
		return nil
	}

	done := h.done
	cancel := h.cancel

	go func() {
		h.inflight.Wait()
		close(h.done)
	}()

	h.started.Store(false)
	h.cancel = nil
	h.done = nil

	cancel()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *EventsHandler) Handle(ctx context.Context, msg *nats.Msg) {
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
