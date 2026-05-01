package memory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventsRepository struct {
	pg *pgxpool.Pool
}

func NewEventsRepository(pgPool *pgxpool.Pool) *EventsRepository {
	return &EventsRepository{
		pg: pgPool,
	}
}

func (r *EventsRepository) FetchNewEvents(ctx context.Context) ([]Event, error) {
	batchSize := 100
	reserveTTLSeconds := 2 * 60

	rows, err := r.pg.Query(ctx, `
	WITH locked_events AS (
		SELECT event_id
		FROM events
		WHERE 
			event_status = 'new'
			AND (
				reserved_to IS NULL
				OR reserved_to < now()
			)
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	)
	UPDATE events e
	SET reserved_to = now() + ($2 * interval '1 second')
	FROM locked_events le
	WHERE e.event_id = le.event_id
	RETURNING
		e.event_id,
		e.event_type,
		e.event_data,
		e.event_status,
		e.created_at,
		e.reserved_to
	`, batchSize, reserveTTLSeconds)
	// TODO NEW: think about order by creted at ASC
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0, batchSize)

	for rows.Next() {
		var e Event

		if err := rows.Scan(
			&e.ID,
			&e.Type,
			&e.Data,
			&e.Status,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows: %w", err)
	}

	return events, nil
}
