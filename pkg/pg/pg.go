package pg

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(logger *slog.Logger, dsn string) *pgxpool.Pool {
	pgPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		logger.Error("failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	if err := pgPool.Ping(context.Background()); err != nil {
		logger.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}
	logger.Info("Postgres pool initialized successfully")

	return pgPool
}
