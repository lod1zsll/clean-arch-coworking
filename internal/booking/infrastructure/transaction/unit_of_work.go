package transaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"coworking/internal/booking/application"
	"coworking/internal/booking/domain"
)

type txBookingRepo interface {
	application.BookingRepo
	WithTx(tx pgx.Tx) application.BookingRepo
}

type txEventStore interface {
	application.EventsRepo
	WithTx(tx pgx.Tx) application.EventsRepo
}

type unitOfWork struct {
	bookingRepo txBookingRepo
	eventRepo   txEventStore
	pg          *pgxpool.Pool
}

func NewUnitOfWork(pgPool *pgxpool.Pool, bookingRepo txBookingRepo, eventRepo txEventStore) application.UnitOfWork {
	return &unitOfWork{
		pg:          pgPool,
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
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
	store := u.eventRepo.WithTx(tx)

	txRepository := &txRepo{
		repo:   repo,
		events: make([]domain.EventItem, 0),
	}

	txEventRepository := &txEventRepo{
		repo: txRepository,
	}

	if err := fn(txRepository, txEventRepository); err != nil {
		return err
	}

	if len(txRepository.events) > 0 {
		if err := store.SaveEvents(ctx, txRepository.events); err != nil {
			return fmt.Errorf("save events: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	committed = true
	return nil
}

type txRepo struct {
	repo   application.BookingRepo
	events []domain.EventItem
}

func (t *txRepo) Save(ctx context.Context, b *domain.Booking) error {
	if err := t.repo.Save(ctx, b); err != nil {
		return err
	}

	domainEvents := b.PullEvents()
	t.events = append(t.events, domainEvents...)

	return nil
}

func (t *txRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return t.repo.FindByID(ctx, id)
}

func (t *txRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	return t.repo.FindByIdempotencyKey(ctx, key)
}

type txEventRepo struct {
	repo *txRepo
}

func (t *txEventRepo) SaveEvents(ctx context.Context, eventItems []domain.EventItem) error {
	t.repo.events = append(t.repo.events, eventItems...)

	return nil
}
