package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/example/coworking/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up structured logging.
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Set up PostgreSQL pool connection's
	pgPool, err := pgxpool.New(context.Background(), cfg.PostgresDSN())
	if err != nil {
		logger.Error("Failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	if err := pgPool.Ping(context.Background()); err != nil {
		logger.Error("Failed to ping postgres", "error", err)
		os.Exit(1)
	}
	logger.Info("Postgres pool initialized successfully")

	// Set up NATS
	drainDone := make(chan struct{})
	nc, err := nats.Connect(
		nats.DefaultURL,
		nats.DrainTimeout(6*time.Second), // between 8 and 5
		nats.ClosedHandler(func(_ *nats.Conn) {
			close(drainDone)
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			logger.Error("nats async error", "error", err)
		}),
	)
	if err != nil {
		logger.Error("Failed to connect to nats", "error", err)
		pgPool.Close()
		os.Exit(1)
	}

	var inflight sync.WaitGroup

	_, err = nc.Subscribe("test", func(m *nats.Msg) {
		inflight.Add(1)
		defer inflight.Done()

		msgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logger.Debug("New message in 'test' topic", "message", string(m.Data))

		// prevent errors
		_ = msgCtx
		_ = pgPool
	})
	if err != nil {
		logger.Error("Failed to subscribe", "error", err)
		nc.Close()
		pgPool.Close()
		os.Exit(1)
	}

	if err := nc.Publish("test", []byte("Hello World")); err != nil {
		logger.Error("Failed to publish", "error", err)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second) // docker signal timeout = 10s
	defer cancel()

	if err := nc.Drain(); err != nil {
		logger.Error("nats drain failed", "error", err)
	}

	// wait drain result or timeout
	select {
	case <-drainDone:
		logger.Info("nats drained")
	case <-shutdownCtx.Done():
		logger.Warn("nats drain timeout, forcing close")
		nc.Close()
	}

	// wait subscribed gorutines result
	waitDone := make(chan struct{})
	go func() {
		inflight.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
		logger.Info("inflight handlers finished")
	case <-shutdownCtx.Done():
		logger.Warn("inflight handlers timeout")
	}

	pgPool.Close()

	logger.Info("Server stopped. Bye!")
}
