package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"coworking/internal/booking/application"
	"coworking/internal/booking/application/outbox"

	"github.com/nats-io/nats.go"
)

const (
	batchSize                  = 500
	batchTTLSec                = 3 * 60
	tickInterval time.Duration = time.Millisecond * 500
)

var ErrPollerAlreadyStarted = errors.New("poller already started")

type Poller struct {
	logger   *slog.Logger
	nc       *nats.Conn
	repo     application.EventsRepo
	outTopic string

	mu      sync.Mutex
	started atomic.Bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewPoller(logger *slog.Logger, nc *nats.Conn, repo application.EventsRepo, outTopic string) *Poller {
	return &Poller{
		logger:   logger,
		nc:       nc,
		repo:     repo,
		outTopic: outTopic,
	}
}

func (p *Poller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started.Load() {
		return ErrPollerAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	p.started.Store(true)
	p.cancel = cancel
	p.done = make(chan struct{})

	go p.run(ctx, p.done)

	return nil
}

func (p *Poller) run(ctx context.Context, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	p.runTick(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			p.runTick(ctx)
		}
	}
}

func (p *Poller) runTick(ctx context.Context) {
	if err := p.tick(ctx); err != nil {
		p.logger.Error("Poll tick failed", "error", err)
	}
}

func (p *Poller) tick(ctx context.Context) error {
	events, err := p.repo.PullNewEvents(ctx, batchSize, batchTTLSec)
	if err != nil {
		return fmt.Errorf("pull events: %w", err)
	}

	errEvents := make([]outbox.Event, 0)

	for i, e := range events {
		eBytes, err := json.Marshal(e.EventMsg)
		if err != nil {
			p.logger.Error("Failed to marshal event message", "error", err)

			// We don't need to change status in DB events table, because we have event TTL
			// And we don't need to write error in some DB, because we already log this error and we can set up alerts
			// We can remove TTL for this events, but I don't see an urgent need for this, because the error will occur EXTREMELY rarely
			errEvents = append(errEvents, e)
			events = append(events[:i], events[i+1:]...)
			continue
		}

		err = p.nc.Publish(p.outTopic, eBytes)
		if err != nil {
			p.logger.Error("Failed to publish message", "error", err)
			continue
		}
	}

	err = p.repo.MarkDoneEvents(ctx, events)
	if err != nil {
		return err
	}

	return nil
}

func (p *Poller) Close(ctx context.Context) {
	p.mu.Lock()
	if !p.started.Load() {
		p.mu.Unlock()
		return
	}

	cancel := p.cancel
	done := p.done

	p.started.Store(false)
	p.cancel = nil
	p.done = nil
	p.mu.Unlock()

	cancel()

	select {
	case <-done:
		p.logger.Info("Poller stopped")
	case <-ctx.Done():
		p.logger.Error("Poller shutdown timeout")
	}
}
