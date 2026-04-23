//go:build integration

package currency

import (
	"context"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/logger"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) Currency {
	cfg := config.MustLoad()

	loggerInstance, err := logger.SetupLogger(cfg.Service.Env)
	if err != nil {
		t.Fatalf("error creating logger: %v", err)
	}

	client, err := New(cfg.API, loggerInstance)
	require.NoError(t, err)
	return client
}

func TestFetchCurrentRates_RealAPI(t *testing.T) {
	client := newTestClient(t)

	req := &dto.CurrencyRequestDTO{
		BaseCurrency:   "USD",
		TargetCurrency: "EUR",
		DateFrom:       time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
		DateTo:         time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC),
	}

	rates, err := client.FetchCurrentRates(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, rates)

	for date, rate := range rates {
		assert.NotEmpty(t, date)
		assert.Greater(t, rate, 0.0)
	}
}
