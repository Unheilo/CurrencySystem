package repository

import (
	"context"
	"my-currency-service/currency/internal/dto"
	"time"
)

type ExchangeRateRepository interface {
	Save(ctx context.Context, date time.Time, baseCurrency string, exchangeCurrency string, rate float32) error
	GetByDateRange(ctx context.Context, baseCurrency string, exchangeCurrency string, from time.Time, to time.Time) ([]dto.CurrencyResponseDTO, error)
	GetByDate(ctx context.Context, baseCurrency string, exchangeCurrency string, date time.Time) (*dto.CurrencyResponseDTO, error)
}

type Currency struct {
	DB ExchangeRateRepository
}

func NewCurrency(repo ExchangeRateRepository) *Currency {
	return &Currency{DB: repo}
}
