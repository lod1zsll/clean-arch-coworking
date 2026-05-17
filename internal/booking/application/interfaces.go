package application

import (
	"context"

	"coworking/internal/booking/application/outbox"
	"coworking/internal/booking/domain"
	"coworking/internal/booking/domain/events"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

//go:generate mockgen -destination=mocks/mock_interfaces.go -package=mocks -source=$GOFILE

type BookingService interface {
	CreateBooking(ctx context.Context, input CreateBookingInput) (uuid.UUID, error)
	GetBooking(ctx context.Context, id uuid.UUID) (*BookingResponse, error)
	ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) error
}

type BookingRepo interface {
	Save(ctx context.Context, b *domain.Booking) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error)
}

type EventBus interface {
	Publish(ctx context.Context, eventItems []events.EventItem) error
}

type PaymentGateway interface {
	Charge(ctx context.Context, bookingID string, amount decimal.Decimal, currency string) (string, error)
}

type AvailabilityChecker interface {
	CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error
}

type PriceCalculator interface {
	CalculatePrice(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) (domain.Money, error)
}

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(BookingRepo, EventStore) error) error
}

type EventStore interface {
	SaveEvents(ctx context.Context, events []events.EventItem) error
}

type EventsRepo interface {
	EventStore

	PullNewEvents(ctx context.Context, batchSize, reserveTTLSec int) ([]outbox.Event, error)
	MarkDoneEvents(ctx context.Context, events []outbox.Event) error
}
