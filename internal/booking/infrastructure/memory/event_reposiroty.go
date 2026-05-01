package memory

import (
	"context"
	"fmt"

	"coworking/internal/booking/application"
	"coworking/internal/booking/application/outbox"
	"coworking/internal/booking/domain/events"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type EventsRepository struct {
	db executor
}

var _ application.EventsRepo = (*EventsRepository)(nil)

func NewEventsRepository(db executor) *EventsRepository {
	return &EventsRepository{
		db: db,
	}
}

func (r *EventsRepository) WithTx(tx pgx.Tx) application.EventsRepo {
	return &EventsRepository{
		db: tx,
	}
}

func (r *EventsRepository) SaveEvents(ctx context.Context, eventItems []events.EventItem) error {

	return nil
}

func (r *EventsRepository) PullNewEvents(ctx context.Context, batchSize, reserveTTLSec int) ([]outbox.Event, error) {
	rows, err := r.db.Query(ctx, `
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
	`, batchSize, reserveTTLSec)
	// TODO NEW: think about order by creted at ASC
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	events := make([]outbox.Event, 0, batchSize)

	for rows.Next() {
		var e outbox.Event

		if err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.EventData,
			&e.Status,
			&e.CreatedAt,
			&e.ReservedTo,
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

func (r *EventsRepository) MarkDoneEvents(ctx context.Context, events []outbox.Event) error {
	ids := make([]uuid.UUID, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}

	_, err := r.db.Exec(ctx, `
		UPDATE events
		SET
			event_status = 'done'
			reserved_to = NULL
		WHERE event_id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("mark done events by uuid: %w", err)
	}

	return nil
}
