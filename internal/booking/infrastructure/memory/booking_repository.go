package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"coworking/internal/booking/application"
	"coworking/internal/booking/domain"
)

type BookingRepository struct {
	db executor
}

func NewBookingRepository(db executor) application.BookingRepo {
	return &BookingRepository{
		db: db,
	}
}

func (r *BookingRepository) WithTx(tx pgx.Tx) application.BookingRepo {
	return &BookingRepository{
		db: tx,
	}
}

func (r *BookingRepository) Save(ctx context.Context, booking *domain.Booking) error {
	_, err := r.db.Exec(ctx, `
	INSERT INTO bookings (
		booking_uuid,
		room_id,
		user_id,
		slot_from,
		slot_to,
		booking_status,
		booking_price_amount, booking_price_currency,
		external_id,
		tx_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (booking_uuid) DO UPDATE SET
		room_id = EXCLUDED.room_id,
		user_id = EXCLUDED.user_id,
		slot_from = EXCLUDED.slot_from,
		slot_to = EXCLUDED.slot_to,
		booking_status = EXCLUDED.booking_status,
		booking_price_amount = EXCLUDED.booking_price_amount,
		booking_price_currency = EXCLUDED.booking_price_currency,
		external_id = EXCLUDED.external_id,
		tx_id = EXCLUDED.tx_id
	`, booking.ID(), booking.RoomID(), booking.UserID(), booking.Slot().From, booking.Slot().To, booking.Status(), booking.Price().ToInt(), booking.Price().Currency, booking.IdempotencyKey(), booking.TransactionID())
	if err != nil {
		return fmt.Errorf("upsert booking: %w", err)
	}

	return nil
}

func (r *BookingRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	row := r.db.QueryRow(ctx, `
	SELECT
		booking_uuid,
		room_id,
		user_id,
		slot_from,
		slot_to,
		booking_status,
		booking_price_amount, booking_price_currency, 
		external_id,
		tx_id 
	FROM
		bookings
	WHERE booking_uuid = $1
	`, id)

	bk, err := r.scanOnce(row)
	if err != nil {
		return nil, fmt.Errorf("query booking by ID: %w", err)
	}

	return bk, nil
}

func (r *BookingRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	row := r.db.QueryRow(ctx, `
	SELECT
		booking_uuid,
		room_id,
		user_id,
		slot_from,
		slot_to,
		booking_status,
		booking_price_amount, booking_price_currency, 
		external_id,
		tx_id 
	FROM
		bookings
	WHERE external_id = $1
	`, key)

	bk, err := r.scanOnce(row)
	if err != nil {
		return nil, fmt.Errorf("query booking by idempotency key: %w", err)
	}

	return bk, nil
}

// TODO NEW: maybe implement as Scan method for domain.Booking
func (r *BookingRepository) scanOnce(row pgx.Row) (*domain.Booking, error) {
	var (
		bkID          uuid.UUID
		roomID        uuid.UUID
		userID        uuid.UUID
		slot          domain.DateRange
		amountInMinor int64
		currency      string
		status        domain.BookingStatus
		externalID    string
		txID          string
	)

	err := row.Scan(
		&bkID,
		&roomID,
		&userID,
		&slot.From,
		&slot.To,
		&status,
		&amountInMinor, &currency,
		&externalID,
		&txID,
	)
	if err != nil {
		return nil, err
	}

	price := domain.NewMoneyFromDecimal(
		decimal.New(amountInMinor, -2),
		currency,
	)

	bk, _ := domain.NewBooking(roomID, userID, slot, price)
	bk.SetIdempotencyKey(externalID)
	bk.SetTransactionID(txID)

	return bk, nil
}
