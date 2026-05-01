package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	bookinghttp "coworking/internal/booking/adapters/http"
	"coworking/internal/booking/application"
	busdummy "coworking/internal/booking/infrastructure/bus/dummy"
	"coworking/internal/booking/infrastructure/memory"
	"coworking/internal/booking/infrastructure/outbox"
	policydummy "coworking/internal/booking/infrastructure/policy/dummy"
	"coworking/internal/booking/infrastructure/transaction"
	"coworking/internal/config"
	"coworking/pkg/pg"
	"coworking/pkg/slogger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slogger.NewLogger(cfg.LogLevel)

	pgPool := pg.NewPool(logger, cfg.PostgresDSN())

	// Wire dependencies
	repo := memory.NewBookingRepository(pgPool)
	bus := busdummy.NewEventBus()
	availabilityChecker := policydummy.NewAvailabilityChecker(pgPool)
	priceCalculator := policydummy.NewPriceCalculator(pgPool)
	eventStore := outbox.NewEventStore(bus)
	uow := transaction.NewUnitOfWork(repo, eventStore)

	svc := application.NewService(repo, bus, availabilityChecker, priceCalculator, uow, logger)
	handler := bookinghttp.NewRouter(svc, logger)

	// Create HTTP server.
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting booking service", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	pgPool.Close()

	logger.Info("server stopped")
}
