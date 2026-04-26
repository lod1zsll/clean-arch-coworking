package domain_test

import (
	"testing"

	"github.com/example/coworking/internal/booking/domain"
	"github.com/shopspring/decimal"
)

func TestNewMoney(t *testing.T) {
	m := domain.NewMoney(1500, "EUR")
	if !decimal.NewFromFloat(1500).Equal(m.Amount) {
		t.Errorf("expected amount 1500, got %v", m.Amount)
	}
	if m.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", m.Currency)
	}

	md := domain.NewMoneyFromDecimal(decimal.NewFromFloat(55.555), "EUR")
	if !decimal.NewFromFloat(55.555).Equal(md.Amount) {
		t.Errorf("expected decimal 55.555, got %v", m.Amount)
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	a := domain.NewMoney(100.02, "USD")
	b := domain.NewMoney(250.01, "USD")

	result := a.Add(b)
	if !decimal.NewFromFloat(350.03).Equal(result.Amount) {
		t.Errorf("expected 350.03, got %v", result.Amount)
	}
	if result.Currency != "USD" {
		t.Errorf("expected USD, got %s", result.Currency)
	}
}

func TestMoney_Add_CurrencyMismatch_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on currency mismatch, got none")
		}
	}()

	a := domain.NewMoney(100, "USD")
	b := domain.NewMoney(200, "EUR")
	a.Add(b)
}

func TestMoney_ToInt(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		want     int64
	}{
		{"an integer", float64(100), "USD", 10000},
		{"an integer with zeros", float64(100.00), "USD", 10000},
		{"an float", float64(100.01), "USD", 10001},
		{"an float", float64(100.01), "BHD", 100010},
		{"an float", float64(100), "JPY", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := domain.NewMoneyFromDecimal(decimal.NewFromFloat(tt.amount), tt.currency)
			result := m.ToInt()
			t.Logf("decimal: %s; result: %d", m.Amount.String(), result)
			if result != tt.want {
				t.Errorf("wanted %d, got %d", tt.want, result)
			}
		})
	}
}
func TestMoney_ToInt_CurrencyNotExist_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on not exists currency, got none")
		}
	}()

	m := domain.NewMoneyFromDecimal(decimal.NewFromFloat(5.55), "TEST")
	m.ToInt()
}
