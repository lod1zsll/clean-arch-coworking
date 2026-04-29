package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
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
	nc, _ := nats.Connect(nats.DefaultURL)
	nc.Subscribe("test", func(m *nats.Msg) {
		logger.Debug("New message in 'test' topic", "message", string(m.Data))
	})
	nc.Publish("test", []byte("Hello World"))

	// Graceful shutdown on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nc.Drain() // Maybe remove
	nc.Close()
	pgPool.Close()

	logger.Info("Server stopped. Bye!")
}
