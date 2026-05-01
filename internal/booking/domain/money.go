package domain

import (
	"github.com/shopspring/decimal"
)

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

func NewMoney(amount float64, currency string) Money {
	return Money{Amount: decimal.NewFromFloat(amount), Currency: currency}
}

func NewMoneyFromMinor(minorUnits int64, currency string) Money {
	decimals, ok := currencyDecimals[currency]
	if !ok {
		panic("unknown currency: " + currency)
	}
	return Money{
		Amount:   decimal.New(minorUnits, -decimals),
		Currency: currency,
	}
}

func NewMoneyFromDecimal(amount decimal.Decimal, currency string) Money {
	return Money{Amount: amount, Currency: currency}
}

func (m Money) Add(other Money) Money {
	if m.Currency != other.Currency {
		panic("currency mismatch")
	}

	return Money{
		Amount:   m.Amount.Add(other.Amount),
		Currency: m.Currency,
	}
}

// ToInt -> amount in minor units;
// Examples:
//   - 100 USD -> 100.00 -> 10000
//   - 100 RUB -> 100.00 -> 10000
//   - 100 JPY -> 100 -> 100
//   - 100 BHD -> 100.000 -> 100000
func (m Money) ToInt() int64 {
	decimals, ok := currencyDecimals[m.Currency]
	if !ok {
		panic("unknown currency: " + m.Currency)
	}
	return m.Amount.Shift(decimals).IntPart()
}

// AmountString returns amount formatted with the fixed number of decimals for the currency.
// Examples:
//   - 100 USD -> "100.00"
//   - 100 JPY -> "100"
//   - 100 BHD -> "100.000"
func (m Money) AmountString() string {
	decimals, ok := currencyDecimals[m.Currency]
	if !ok {
		panic("unknown currency: " + m.Currency)
	}
	return m.Amount.StringFixed(decimals)
}

// TODO NEW: Switch to DB logic & add e2e/mock tests with DB; Check price_calculator realisation
var currencyDecimals = map[string]int32{
	"USD": 2,
	"EUR": 2,
	"RUB": 2,
	"JPY": 0,
	"BHD": 3,
}
