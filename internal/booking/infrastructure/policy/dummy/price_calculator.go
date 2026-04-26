package dummy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/example/coworking/internal/booking/domain"
)

// FIXME: hardcoded price — should calculate based on room hourly rate and slot duration.
type PriceCalculator struct {
	pg *pgxpool.Pool
}

func NewPriceCalculator(pgPool *pgxpool.Pool) *PriceCalculator {
	return &PriceCalculator{
		pg: pgPool,
	}
}

// TODO NEW: Add mock test's
func (p *PriceCalculator) CalculatePrice(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) (domain.Money, error) {
	var (
		amountMinor int64
		currency    string
		exponent    int32
	)

	// TODO NEW: (?) Optimizate query with amount-service and remove currency join fetching
	err := p.pg.QueryRow(ctx, `
	SELECT 
		r.amount_per_day, r.amount_currency,
		c.currency_exp
	FROM
		rooms r
	LEFT JOIN
		currency c ON r.amount_currency = c.currency_iso
	WHERE
		r.room_id = $1
	`, roomID.String()).Scan(&amountMinor, &currency, &exponent)
	if err != nil {
		return domain.Money{}, fmt.Errorf("calc price scan: %w", err)
	}

	slotDaysCount := (slot.To.Unix() - slot.From.Unix()) / (60 * 60 * 24)
	calculatedAmount := decimal.New(amountMinor, -exponent).Mul(decimal.NewFromInt(slotDaysCount))

	return domain.NewMoneyFromDecimal(
		calculatedAmount,
		currency,
	), nil
}
