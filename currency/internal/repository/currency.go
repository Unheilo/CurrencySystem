package repository

import (
	"context"
	"my-currency-service/currency/internal/dto"
	"time"
)

type ExchangeRateRepository interface {
	Save(ctx context.Context, date time.Time, baseCurrency string, rates map[string]float64) error
	FindInInterval(ctx context.Context, dto *dto.CurrencyRequestDTO) ([]CurrencyRate, error)
}

type Currency struct {
	repo ExchangeRateRepository
}

func NewCurrency(repo ExchangeRateRepository) *Currency {
	return &Currency{repo: repo}
}
