package application_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
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

	svc := application.NewService(mockRepo, nil, nil, nil, nil, slog.Default())

	resp, err := svc.GetBooking(context.Background(), booking.ID())

	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != booking.ID() {
		t.Errorf("booking id: got %v, want %v", resp.ID, booking.ID())
	}
	if resp.RoomID != roomID {
		t.Errorf("booking room id: got %v, want %v", resp.RoomID, booking.RoomID())
	}
	if resp.UserID != userID {
		t.Errorf("booking user id: got %v, want %v", resp.UserID, booking.UserID())
	}
	if resp.From != from {
		t.Errorf("booking slot from: got %v, want %v", resp.From, from)
	}
	if resp.To != to {
		t.Errorf("booking slot to: got %v, want %v", resp.To, to)
	}
	if resp.PriceAmount != money.AmountString() {
		t.Errorf("booking price amount: got %v, want %v", resp.PriceAmount, money.AmountString())
	}
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

	svc := application.NewService(mockRepo, nil, nil, nil, nil, slog.Default())

	_, err := svc.CreateBooking(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidRange) {
		t.Errorf("got %v, want %v", err, domain.ErrInvalidRange)
	}
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

	slot, err := domain.NewDateRange(from, to)
	if err != nil {
		t.Fatal(err)
	}
	booking, err := domain.NewBooking(roomID, userID, slot, money)
	if err != nil {
		t.Fatal(err)
	}

	input := application.CreateBookingInput{
		RoomID:         roomID,
		UserID:         userID,
		From:           from,
		To:             to,
		IdempotencyKey: idemptKey,
	}

	mockRepo.EXPECT().FindByIdempotencyKey(gomock.Any(), idemptKey).
		Return(booking, nil)

	svc := application.NewService(mockRepo, nil, nil, nil, nil, slog.Default())

	idemptId, err := svc.CreateBooking(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if idemptId != booking.ID() {
		t.Errorf("got %v, want %v", idemptId, booking.ID())
	}
}
