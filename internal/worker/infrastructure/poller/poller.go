package poller

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

type Poller struct {
	logger   *slog.Logger
	nc       *nats.Conn
	pgPool   *pgxpool.Pool
	inflight sync.WaitGroup
}

func NewPoller(logger *slog.Logger, nc *nats.Conn, pgPool *pgxpool.Pool) *Poller {
	return &Poller{
		logger: logger,
		nc:     nc,
		pgPool: pgPool,
	}
}

func (p *Poller) Start() error {
	_, err := p.nc.Subscribe("test", func(m *nats.Msg) {
		p.inflight.Add(1)
		defer p.inflight.Done()

		msgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		p.logger.Debug("New message in 'test' topic", "message", string(m.Data))

		// prevent errors
		_ = msgCtx
	})
	if err != nil {
		p.logger.Error("Failed to subscribe", "error", err)
		return err
	}

	return nil
}

func (p *Poller) Close(ctx context.Context) {
	waitDone := make(chan struct{})
	go func() {
		p.inflight.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
		p.logger.Info("Inflight handlers finished")
	case <-ctx.Done():
		p.logger.Error("Inflight handlers timeout")
	}
}
