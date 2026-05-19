package application_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"coworking/internal/booking/application"
	"coworking/internal/booking/application/mocks"
	"coworking/internal/booking/domain"
)

func TestGetBooking_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockBookingRepo(ctrl)

	roomID := uuid.New()
	userID := uuid.New()
	from := time.Now().Add(24 * time.Hour)
	to := from.Add(24 * time.Hour)

	money := domain.NewMoneyFromMinor(1000, "USD")

	slot, err := domain.NewDateRange(from, to)
	if err != nil {
		t.Fatal(err)
	}
	booking, err := domain.NewBooking(roomID, userID, slot, money)
	if err != nil {
		t.Fatal(err)
	}

	mockRepo.EXPECT().
		FindByID(gomock.Any(), booking.ID()).
		Return(booking, nil)

	svc := application.NewService(mockRepo, nil, nil, nil, slog.Default())

	resp, err := svc.GetBooking(context.Background(), booking.ID())

	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)

	assert.Equal(resp.ID, booking.ID(), "booking id: got %v, want %v", resp.ID, booking.ID())
	assert.Equal(resp.RoomID, booking.RoomID(), "room id: got %v, want %v", resp.RoomID, booking.RoomID())
	assert.Equal(resp.UserID, booking.UserID(), "booking user id: got %v, want %v", resp.UserID, booking.UserID())
	assert.Equal(resp.From, from, "booking slot from: got %v, want %v", resp.From, from)
	assert.Equal(resp.To, to, "booking slot to: got %v, want %v", resp.To, to)
	assert.Equal(resp.PriceAmount, money.AmountString(), "booking price amount: got %v, want %v", resp.PriceAmount, money.AmountString())
}

func TestCreateBooking_InvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// for "unexpected call" instead of "panic"
	mockRepo := mocks.NewMockBookingRepo(ctrl)

	roomID := uuid.New()
	userID := uuid.New()
	from := time.Now().Add(24 * time.Hour)
	to := from.Add(-time.Hour)
	idemptKey := "test"

	input := application.CreateBookingInput{
		RoomID:         roomID,
		UserID:         userID,
		From:           from,
		To:             to,
		IdempotencyKey: idemptKey,
	}

	svc := application.NewService(mockRepo, nil, nil, nil, slog.Default())

	_, err := svc.CreateBooking(context.Background(), input)
	assert.ErrorIs(t, err, domain.ErrInvalidRange, "got %v, want %v", err, domain.ErrInvalidRange)
}

func TestCreateBooking_Idempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockBookingRepo(ctrl)

	roomID := uuid.New()
	userID := uuid.New()
	from := time.Now().Add(24 * time.Hour)
	to := from.Add(24 * time.Hour)
	idemptKey := "test"

	money := domain.NewMoneyFromMinor(1000, "USD")

	assert := assert.New(t)

	slot, err := domain.NewDateRange(from, to)
	assert.Nil(err, "new date range: %v", err)

	booking, err := domain.NewBooking(roomID, userID, slot, money)
	assert.Nil(err, "new booking: %v", err)

	input := application.CreateBookingInput{
		RoomID:         roomID,
		UserID:         userID,
		From:           from,
		To:             to,
		IdempotencyKey: idemptKey,
	}

	mockRepo.EXPECT().FindByIdempotencyKey(gomock.Any(), idemptKey).
		Return(booking, nil)

	svc := application.NewService(mockRepo, nil, nil, nil, slog.Default())

	idemptId, err := svc.CreateBooking(context.Background(), input)
	assert.Nil(err, "create booking: %v", err)
	assert.Equal(idemptId, booking.ID(), "got %v, want %v", idemptId, booking.ID())
}
