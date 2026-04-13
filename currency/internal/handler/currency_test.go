package handler

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"my-currency-service/currency/internal/handler/mocks"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/pkg/currency"
	"testing"
	"time"

	"my-currency-service/currency/internal/dto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// newTestServer создаёт CurrencyServer с моком и заглушками метрик.
// Возвращает сервер и мок, чтобы в тесте настроить ожидания.
func newTestServer(t *testing.T) (*CurrencyServer, *mocks.CurrencyService) {
	service := mocks.NewCurrencyService(t)

	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "test_request_count", Help: "test"},
		[]string{"method"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "test_request_duration", Help: "test", Buckets: prometheus.DefBuckets},
		[]string{"method"},
	)
	appUptime := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_app_uptime", Help: "test"})

	server := NewCurrencyServer(service, slog.Default(), requestCount, requestDuration, &appUptime)
	return server, service
}

func TestGetRate_Success(t *testing.T) {
	server, service := newTestServer(t)

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.Anything).
		Return([]repository.CurrencyRate{
			{Date: now, Rate: 1.10},
			{Date: now.AddDate(0, 0, 1), Rate: 1.12},
		}, nil)

	req := &currency.GetRateRequest{
		Currency: "EUR",
		DataFrom: timestamppb.New(now),
		DateTo:   timestamppb.New(now.AddDate(0, 0, 1)),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "EUR", resp.Currency)
	assert.Len(t, resp.Rates, 2)
	assert.Equal(t, float32(1.10), resp.Rates[0].Rate)
	assert.Equal(t, float32(1.12), resp.Rates[1].Rate)
}

func TestGetRate_ServiceError(t *testing.T) {
	server, service := newTestServer(t)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.Anything).
		Return(nil, errors.New("db connection failed"))

	req := &currency.GetRateRequest{
		Currency: "EUR",
		DataFrom: timestamppb.New(time.Now()),
		DateTo:   timestamppb.New(time.Now()),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestGetRate_EmptyRates(t *testing.T) {
	server, service := newTestServer(t)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.Anything).
		Return([]repository.CurrencyRate{}, nil)

	req := &currency.GetRateRequest{
		Currency: "GBP",
		DataFrom: timestamppb.New(time.Now()),
		DateTo:   timestamppb.New(time.Now()),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "GBP", resp.Currency)
	assert.Empty(t, resp.Rates)
}

func TestGetRate_SingleRate(t *testing.T) {
	server, service := newTestServer(t)

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.Anything).
		Return([]repository.CurrencyRate{
			{Date: now, Rate: 0.85},
		}, nil)

	req := &currency.GetRateRequest{
		Currency: "GBP",
		DataFrom: timestamppb.New(time.Now()),
		DateTo:   timestamppb.New(time.Now()),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "GBP", resp.Currency)
	assert.Len(t, resp.Rates, 1)
	assert.Equal(t, float32(0.85), resp.Rates[0].Rate)
	assert.Equal(t, now, resp.Rates[0].Date.AsTime())

}

func TestGetRate_DefaultBaseCurrency(t *testing.T) {
	server, service := newTestServer(t)

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.MatchedBy(func(req *dto.CurrencyRequestDTO) bool {
		return req.BaseCurrency == "USD"
	})).Return([]repository.CurrencyRate{
		{Date: now, Rate: 1.15},
	}, nil)

	req := &currency.GetRateRequest{
		Currency: "EUR",
		DataFrom: timestamppb.New(time.Now()),
		DateTo:   timestamppb.New(time.Now()),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "EUR", resp.Currency)

}

func TestGetRate_DatesPassedCorrectly(t *testing.T) {
	server, service := newTestServer(t)

	dateFrom := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.MatchedBy(func(req *dto.CurrencyRequestDTO) bool {
		return req.DateFrom.Equal(dateFrom) && req.DateTo.Equal(dateTo)
	})).
		Return([]repository.CurrencyRate{
			{Date: dateFrom, Rate: 1.10},
			{Date: dateTo, Rate: 1.37},
		}, nil)

	req := &currency.GetRateRequest{
		Currency: "EUR",
		DataFrom: timestamppb.New(dateFrom),
		DateTo:   timestamppb.New(dateTo),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, dateFrom, resp.Rates[0].Date.AsTime())
	assert.Equal(t, dateTo, resp.Rates[1].Date.AsTime())

}

func TestGetRate_ManyRates(t *testing.T) {
	server, service := newTestServer(t)

	rates := make([]repository.CurrencyRate, 30)
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	numberOfTests := 30

	for i := 0; i < numberOfTests; i++ {
		rates[i] = repository.CurrencyRate{
			Date: baseDate.AddDate(0, 0, i),
			Rate: float32(0.5 + rand.Float64()*1.5),
		}
	}

	service.On("GetCurrencyRatesInInterval", mock.Anything, mock.Anything).
		Return(rates, nil)

	req := &currency.GetRateRequest{
		Currency: "EUR",
		DataFrom: timestamppb.New(rates[0].Date),
		DateTo:   timestamppb.New(rates[29].Date),
	}

	resp, err := server.GetRate(context.Background(), req)

	require.NoError(t, err)
	assert.Len(t, resp.Rates, 30)
	for i := 0; i < numberOfTests; i++ {
		assert.Equal(t, rates[i].Date, resp.Rates[i].Date.AsTime())
	}

}
