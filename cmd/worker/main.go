package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coworking/internal/booking/infrastructure/memory"
	"coworking/internal/booking/infrastructure/outbox"
	"coworking/internal/config"
	"coworking/pkg/natser"
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

	natsWrapper, err := natser.NewNatsWrapper(logger, cfg.NatsDSN())
	if err != nil {
		pgPool.Close()
		os.Exit(1)
	}

	eventsRepo := memory.NewEventsRepository(pgPool)
	eventsPoller := outbox.NewPoller(logger, natsWrapper.GetConn(), eventsRepo, cfg.TopicOut)

	if err := eventsPoller.Start(); err != nil {
		natsWrapper.Close(context.Background())
		pgPool.Close()
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Close poller
	eventsPoller.Close(shutdownCtx)

	// Drain -> Close for nats connection
	natsWrapper.Close(shutdownCtx)

	// Close postgres pool
	pgPool.Close()

	logger.Info("Server stopped. Bye!")
}
