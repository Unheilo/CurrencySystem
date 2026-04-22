package service

import (
	"context"
	"fmt"
	"log/slog"
	"my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/repository"
	"strings"
	"time"
)

type Currency struct {
	currencyRepo repository.ExchangeRateRepository
	client       currency.Currency
	logger       *slog.Logger
}

func NewCurrency(
	repo repository.ExchangeRateRepository,
	client currency.Currency,
	logger *slog.Logger,
) *Currency {
	return &Currency{
		currencyRepo: repo,
		client:       client,
		logger:       logger,
	}
}

func (s *Currency) GetCurrencyRatesInInterval(ctx context.Context, reqDTO *dto.CurrencyRequestDTO) ([]repository.CurrencyRate, error) {

	reqDTO.BaseCurrency = strings.ToUpper(reqDTO.BaseCurrency)
	reqDTO.TargetCurrency = strings.ToUpper(reqDTO.TargetCurrency)

	rates, err := s.currencyRepo.FindInInterval(ctx, reqDTO)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch currency rates in interval: %w", err)
	}

	return rates, nil

}

func (s *Currency) FetchAndSaveCurrencyRates(ctx context.Context, reqDTO *dto.CurrencyRequestDTO) error {

	var dayNow = time.Now()
	var dayYesterday = time.Now().AddDate(0, 0, -1)

	reqDTO.DateFrom = dayYesterday
	reqDTO.DateTo = dayNow
	reqDTO.BaseCurrency = strings.ToUpper(reqDTO.BaseCurrency)
	reqDTO.TargetCurrency = strings.ToUpper(reqDTO.TargetCurrency)

	rates, err := s.client.FetchCurrentRates(ctx, reqDTO)

	if err != nil {
		return fmt.Errorf("failed to fetch currency rates in interval: %w", err)
	}

	if err := s.currencyRepo.Save(ctx, dayNow, reqDTO.BaseCurrency, rates); err != nil {
		return fmt.Errorf("failed to save currency rates in interval: %w", err)
	}

	s.logger.Info("successfully saved currency rates", slog.Any("rates", rates))
	return nil

}
