package handler

import (
	"context"
	"log/slog"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/pkg/currency"

	"github.com/prometheus/client_golang/prometheus"
)

type CurrencyService interface {
	GetCurrencyRatesInInterval(ctx context.Context, reqDTO *dto.CurrencyRequestDTO) ([]repository.CurrencyRate, error)
	FetchAndSaveCurrencyRates(ctx context.Context, baseCurrency string) error
}

// todo tests
type CurrencyServer struct {
	currency.UnimplementedCurrencyServiceServer
	service CurrencyService
	logger  *slog.Logger

	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	appUptime       *prometheus.Gauge
}

func NewCurrencyServer(svc CurrencyService, logger *slog.Logger,
	requestCount *prometheus.CounterVec, requestDuration *prometheus.HistogramVec,
	appUptime *prometheus.Gauge) *CurrencyServer {

	return &CurrencyServer{
		service:         svc,
		logger:          logger,
		requestCount:    requestCount,
		requestDuration: requestDuration,
		appUptime:       appUptime,
	}
}
