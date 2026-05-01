package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	if len(eventItems) == 0 {
		return nil
	}

	var b strings.Builder
	b.Grow(len(eventItems) * 10)
	b.WriteString("INSERT INTO events (event_type, event_data) VALUES ")

	args := make([]any, 0, len(eventItems)*2)

	for i, item := range eventItems {
		eType, eData := eventItemToRecord(item)

		if i > 0 {
			b.WriteString(",")
		}

		fmt.Fprintf(&b, "($%d, $%d)", i*2+1, i*2+2)

		args = append(args, eType, eData)
	}

	_, err := r.db.Exec(ctx, b.String(), args...)
	return err
}

func eventItemToRecord(event events.EventItem) (outbox.EventType, json.RawMessage) {
	switch event.(type) {
	case events.RoomBooked:
		dataBytes, _ := json.Marshal(event)

		return outbox.EventTypeBooking, dataBytes
	case events.BookingConfirmed:
		dataBytes, _ := json.Marshal(event)

		return outbox.EventTypeConfirm, dataBytes
	default:
		return outbox.EventTypeUnknown, nil
	}
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
			&e.UUID,
			&e.Type,
			&e.Data,
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
		ids[i] = e.UUID
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
