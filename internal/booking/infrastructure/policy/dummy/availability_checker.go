package dummy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/coworking/internal/booking/domain"
)

type AvailabilityChecker struct {
	pg *pgxpool.Pool
}

func NewAvailabilityChecker(pgPool *pgxpool.Pool) *AvailabilityChecker {
	return &AvailabilityChecker{
		pg: pgPool,
	}
}

func (a *AvailabilityChecker) CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error {
	var isAvailable bool

	// TODO NEW: Add index on (room_id, slot_from, slot_to) and optimize query & prevent race condition (gist)
	err := a.pg.QueryRow(ctx, `
	SELECT NOT EXISTS (
		SELECT
			1 FROM bookings
		WHERE 
			room_id = $1
		AND 
			tstzrange(slot_from, slot_to, '[)') && tstzrange($2, $3, '[)')
	)
	`, roomID, slot.From, slot.To).Scan(&isAvailable)
	if err != nil {
		return fmt.Errorf("pg query: %w", err)
	}
	if !isAvailable {
		return domain.ErrRoomNotAvailable
	}

	return nil
}
