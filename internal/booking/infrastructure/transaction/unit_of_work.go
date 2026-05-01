package transaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"coworking/internal/booking/application"
	"coworking/internal/booking/domain"
	"coworking/internal/booking/domain/events"
)

type txBookingRepo interface {
	application.BookingRepo
	WithTx(tx pgx.Tx) application.BookingRepo
}

type txEventStore interface {
	application.EventStore
	WithTx(tx pgx.Tx) application.EventStore
}

type unitOfWork struct {
	bookingRepo txBookingRepo
	eventStore  txEventStore
	pg          *pgxpool.Pool
}

func NewUnitOfWork(pgPool *pgxpool.Pool, bookingRepo txBookingRepo, eventStore txEventStore) application.UnitOfWork {
	return &unitOfWork{
		pg:          pgPool,
		bookingRepo: bookingRepo,
		eventStore:  eventStore,
	}
}

func (u *unitOfWork) Execute(ctx context.Context, fn func(application.BookingRepo, application.EventStore) error) error {
	tx, err := u.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	repo := u.bookingRepo.WithTx(tx)
	store := u.eventStore.WithTx(tx)

	transactionalRepo := &transactionalRepo{
		repo:   repo,
		events: make([]events.EventItem, 0),
	}

	transactionalEventStore := &transactionalEventStore{
		repo: transactionalRepo,
	}

	if err := fn(transactionalRepo, transactionalEventStore); err != nil {
		return err
	}

	if len(transactionalRepo.events) > 0 {
		if err := store.SaveEvents(ctx, transactionalRepo.events); err != nil {
			return fmt.Errorf("save events: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	committed = true
	return nil
}

type transactionalRepo struct {
	repo   application.BookingRepo
	events []events.EventItem
}

func (t *transactionalRepo) Save(ctx context.Context, b *domain.Booking) error {
	if err := t.repo.Save(ctx, b); err != nil {
		return err
	}

	domainEvents := b.PullEvents()
	t.events = append(t.events, domainEvents...)

	return nil
}

func (t *transactionalRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return t.repo.FindByID(ctx, id)
}

func (t *transactionalRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	return t.repo.FindByIdempotencyKey(ctx, key)
}

type transactionalEventStore struct {
	repo *transactionalRepo
}

func (t *transactionalEventStore) SaveEvents(ctx context.Context, eventItems []events.EventItem) error {
	t.repo.events = append(t.repo.events, eventItems...)

	return nil
}
