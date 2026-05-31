package consumer

import (
	"context"
	"coworking/internal/booking/application"
	"coworking/pkg/natser"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type EventsHandler struct {
	logger *slog.Logger
	nats   *natser.NatsWrapper
	rdb    *redis.Client

	topicIn  string
	dedupTTL time.Duration

	mu       sync.Mutex
	started  bool
	inflight sync.WaitGroup
	cancel   context.CancelFunc
	sub      *nats.Subscription
}

var ErrHandlerAlreadyStarted = errors.New("handler already started")

func NewEventsHandler(logger *slog.Logger, natsWrapper *natser.NatsWrapper, rdb *redis.Client, topicIn string, dedupTTL time.Duration) application.NatsHandler {
	if dedupTTL < 1 {
		dedupTTL = time.Hour * 24
	}

	return &EventsHandler{
		logger:   logger,
		nats:     natsWrapper,
		rdb:      rdb,
		topicIn:  topicIn,
		dedupTTL: dedupTTL,
	}
}

func (h *EventsHandler) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.started {
		return ErrHandlerAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	sub, err := h.nats.GetConn().Subscribe(h.topicIn, func(msg *nats.Msg) {
		h.Handle(ctx, msg)
	})
	if err != nil {
		cancel()
		return fmt.Errorf("subscribe: %w", err)
	}

	h.sub = sub
	h.cancel = cancel
	h.started = true

	return nil
}

func (h *EventsHandler) Close(ctx context.Context) error {
	h.mu.Lock()
	if !h.started {
		h.mu.Unlock()
		return nil
	}

	sub := h.sub
	cancel := h.cancel

	h.started = false
	h.cancel = nil
	h.sub = nil
	h.mu.Unlock()

	if err := sub.Unsubscribe(); err != nil {
		h.logger.Error("Failed to unsubscribe", "error", err)
	}

	done := make(chan struct{})
	go func() {
		h.inflight.Wait()
		close(done)
	}()

	select {
	case <-done:
		cancel()
		return nil
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

func (h *EventsHandler) Handle(ctx context.Context, msg *nats.Msg) {
	h.inflight.Add(1)
	defer h.inflight.Done()

	msgIdHeader, _ := msg.Header["Nats-Msg-Id"]
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

	key := fmt.Sprintf("outbox:dedup:%s:%s", h.topicIn, msgId)
	set, err := h.rdb.SetNX(ctx, key, 1, h.dedupTTL).Result()
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
