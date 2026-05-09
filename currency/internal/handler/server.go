package handler

import (
	"context"
	"log/slog"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/pkg/currency"
)

type CurrencyService interface {
	GetCurrencyRatesInInterval(ctx context.Context, reqDTO *dto.CurrencyRequestDTO) ([]repository.CurrencyRate, error)
}

// todo tests
type CurrencyServer struct {
	currency.UnimplementedCurrencyServiceServer
	service             CurrencyService
	logger              *slog.Logger
	defaultBaseCurrency string
}

func NewCurrencyServer(svc CurrencyService, logger *slog.Logger,
	defaultBaseCurrency string) *CurrencyServer {

	return &CurrencyServer{
		service:             svc,
		logger:              logger,
		defaultBaseCurrency: defaultBaseCurrency,
	}
}
