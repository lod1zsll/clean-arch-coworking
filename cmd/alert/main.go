package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coworking/internal/booking/adapters/consumer"
	"coworking/internal/config"
	"coworking/pkg/natser"
	"coworking/pkg/pg"
	"coworking/pkg/slogger"

	"github.com/redis/go-redis/v9"
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	natsWrapper, err := natser.NewNatsWrapper(logger, cfg.NatsDSN())
	if err != nil {
		os.Exit(1)
	}

	handler := consumer.NewEventsHandler(logger, rdb, cfg.TopicIn, time.Duration(cfg.DedupTTLSec)*time.Second)

	// Graceful shutdown on SIGINT / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	// Close subscribe
	_ = handler.Close(shutdownCtx)

	// Drain -> Close for nats connection
	natsWrapper.Close(shutdownCtx)

	logger.Info("Server stopped. Bye!")
}
