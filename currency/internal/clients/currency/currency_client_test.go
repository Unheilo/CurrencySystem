package currency

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testXML = `<?xml version="1.0" encoding="UTF-8"?>
  <StructureSpecificData>
      <DataSet>
          <Series>
              <Obs TIME_PERIOD="2024-05-01" OBS_VALUE="1.0823"/>
              <Obs TIME_PERIOD="2024-05-02" OBS_VALUE="1.0791"/>
          </Series>
      </DataSet>
  </StructureSpecificData>`

func newTestMetrics() ECBMetrics {
	reg := prometheus.NewRegistry()
	rd := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "test_ecb_request_duration_seconds"},
		[]string{"status"},
	)
	errs := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "test_ecb_errors_total"},
		[]string{"type"},
	)
	reg.MustRegister(rd, errs)
	return ECBMetrics{RequestDuration: rd, Errors: errs}
}

func newHTTPTestClient(t *testing.T, baseURL string, maxAttempts int) Currency {
	t.Helper()
	cli, err := New(
		config.APIConfig{
			BaseURL:        baseURL + "/%s/%s?from=%s&to=%s",
			TimeoutSeconds: 5,
			Retry: config.APIRetryConfig{
				MaxAttempts: maxAttempts,
				BaseDelayMs: 1,
				MaxDelayMs:  10,
			},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		newTestMetrics(),
	)
	require.NoError(t, err)
	return cli
}

func TestExtractObs(t *testing.T) {
	rates, err := extractObs(strings.NewReader(testXML))

	require.NoError(t, err)

	require.Len(t, rates, 2)
	assert.Equal(t, "2024-05-01", rates[0].Date.Format("2006-01-02"))
	assert.InDelta(t, 1.0823, rates[0].Value, 0.0001)

	assert.Equal(t, "2024-05-02", rates[1].Date.Format("2006-01-02"))
	assert.InDelta(t, 1.0791, rates[1].Value, 0.0001)
}

func TestFetchCurrentRates_RetriesOn5xx(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(testXML))
	}))
	defer ts.Close()

	cli := newHTTPTestClient(t, ts.URL, 3)

	req := &dto.CurrencyRequestDTO{
		BaseCurrency:   "USD",
		TargetCurrency: "EUR",
		DateFrom:       time.Now(),
		DateTo:         time.Now(),
	}
	rates, err := cli.FetchCurrentRates(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.NotEmpty(t, rates)
}

func TestFetchCurrentRates_EmptyBodyReturnsNoData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cli := newHTTPTestClient(t, ts.URL, 3)

	req := &dto.CurrencyRequestDTO{
		BaseCurrency:   "USD",
		TargetCurrency: "EUR",
		DateFrom:       time.Now(),
		DateTo:         time.Now(),
	}
	rates, err := cli.FetchCurrentRates(context.Background(), req)
	require.NoError(t, err)
	assert.Empty(t, rates)
}

func TestFetchCurrentRates_FailsFastOn400(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer ts.Close()

	cli := newHTTPTestClient(t, ts.URL, 3)

	req := &dto.CurrencyRequestDTO{
		BaseCurrency:   "USD",
		TargetCurrency: "EUR",
		DateFrom:       time.Now(),
		DateTo:         time.Now(),
	}
	_, err := cli.FetchCurrentRates(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, 1, attempts)
}
