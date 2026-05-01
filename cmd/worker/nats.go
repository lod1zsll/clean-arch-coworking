package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsWrapper struct {
	nc          *nats.Conn
	logger      *slog.Logger
	drainDoneCh chan struct{}
}

func NewNatsWrapper(logger *slog.Logger) (*NatsWrapper, error) {
	// set up nats

	drainDone := make(chan struct{})
	nc, err := nats.Connect(
		nats.DefaultURL,
		nats.DrainTimeout(6*time.Second), // between 8 and 5
		nats.ClosedHandler(func(_ *nats.Conn) {
			close(drainDone)
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			logger.Error("Nats async error", "error", err)
		}),
	)
	if err != nil {
		logger.Error("Failed to connect to nats", "error", err)
		return nil, err
	}

	logger.Info("NATS connected sucessfully")
	return &NatsWrapper{
		nc:          nc,
		logger:      logger,
		drainDoneCh: drainDone,
	}, nil
}

func (w *NatsWrapper) GetConn() *nats.Conn {
	return w.nc
}

func (w *NatsWrapper) Close(ctx context.Context) {
	if err := w.nc.Drain(); err != nil {
		w.logger.Error("Nats drain failed", "error", err)
	}

	// wait drain result or timeout
	select {
	case <-w.drainDoneCh:
		w.logger.Info("Nats drained successfully")
	case <-ctx.Done():
		w.logger.Error("Nats drain timeout, forcing close")
		w.nc.Close()
	}
}
