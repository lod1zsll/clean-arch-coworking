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
//   - 100 USD -> 100.00
//   - 100 RUB -> 100.00
//   - 100 JPY -> 100
//   - 100 BHD -> 100.000
func (m Money) ToInt() int64 {
	decimals, ok := currencyDecimals[m.Currency]
	if !ok {
		panic("unknown currency: " + m.Currency)
	}
	return m.Amount.Shift(decimals).IntPart()
}

// TODO: Use with DB
var currencyDecimals = map[string]int32{
	"USD": 2,
	"EUR": 2,
	"RUB": 2,
	"JPY": 0,
	"BHD": 3,
}
