package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coworking/internal/booking/infrastructure/outbox"
	"coworking/internal/config"
	"coworking/pkg/pg"
	"coworking/pkg/slogger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slogger.NewLogger(cfg.LogLevel)
	pgPool := pg.NewPool(logger, cfg.PostgresDSN())

	natsWrapper, err := NewNatsWrapper(logger)
	if err != nil {
		pgPool.Close()
		os.Exit(1)
	}

	eventsPoller := outbox.NewPoller(logger, natsWrapper.GetConn(), pgPool)
	err = eventsPoller.Start()
	if err != nil {
		pgPool.Close()
		os.Exit(1)
	}

	// graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second) // docker signal timeout = 10s
	defer cancel()

	// Drain -> Close for nats connection
	natsWrapper.Close(shutdownCtx)

	// close poller
	eventsPoller.Close(shutdownCtx)

	// close pool
	pgPool.Close()

	logger.Info("Server stopped. Bye!")
}
