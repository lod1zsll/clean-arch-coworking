package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"coworking/internal/config"
	"coworking/pkg/natser"
	"coworking/pkg/pg"
	"coworking/pkg/slogger"

	"github.com/nats-io/nats.go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slogger.NewLogger(cfg.LogLevel)

	pgPool := pg.NewPool(logger, cfg.PostgresDSN())
	defer pgPool.Close()

	natsWrapper, err := natser.NewNatsWrapper(logger, cfg.NatsDSN())
	if err != nil {
		os.Exit(1)
	}

	var inflight sync.WaitGroup
	sub, err := natsWrapper.GetConn().Subscribe(cfg.TopicIn, func(msg *nats.Msg) {
		inflight.Add(1)
		defer inflight.Done()

		logger.Info("New msg", "msg_data", string(msg.Data))
	})

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	// Close subscribe
	sub.Unsubscribe()

	inflight.Wait()

	// Drain -> Close for nats connection
	natsWrapper.Close(shutdownCtx)

	logger.Info("Server stopped. Bye!")
}
