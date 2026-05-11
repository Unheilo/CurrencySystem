package currency

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Currency struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	metrics    ECBMetrics
	retry      RetryPolicy
}

type ECBMetrics struct {
	RequestDuration *prometheus.HistogramVec
	Errors          *prometheus.CounterVec
}

func New(cfg config.APIConfig, logger *slog.Logger, metrics ECBMetrics) (*Currency, error) {
	return &Currency{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipVerify},
			},
		},
		logger:  logger,
		metrics: metrics,
		retry: RetryPolicy{
			MaxAttempts: cfg.Retry.MaxAttempts,
			BaseDelay:   time.Duration(cfg.Retry.BaseDelayMs) * time.Millisecond,
			MaxDelay:    time.Duration(cfg.Retry.MaxDelayMs) * time.Millisecond,
		},
	}, nil
}

func (c *Currency) buildURL(ReqData *dto.CurrencyRequestDTO) (string, error) {
	if ReqData.BaseCurrency == "" || ReqData.TargetCurrency == "" ||
		ReqData.DateFrom.IsZero() || ReqData.DateTo.IsZero() {
		return "", fmt.Errorf("found zero value in request: BaseCurrency %s, TargetCurrency %s, DateFrom %s, DateTo %s",
			ReqData.BaseCurrency, ReqData.TargetCurrency, ReqData.DateFrom, ReqData.DateTo)
	}
	return fmt.Sprintf(c.baseURL,
		ReqData.BaseCurrency, ReqData.TargetCurrency,
		ReqData.DateFrom.Format("2006-01-02"), ReqData.DateTo.Format("2006-01-02")), nil
}

func (c *Currency) FetchCurrentRates(ctx context.Context, req *dto.CurrencyRequestDTO) (map[string]float64, error) {
	var rates map[string]float64
	err := c.retry.Do(ctx, func(attempt int) error {
		if attempt > 0 {
			c.logger.InfoContext(ctx, "ecb retry", slog.Int("attempt", attempt))
		}
		r, err := c.fetchOnce(ctx, req)
		if err != nil {
			return err
		}
		rates = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rates, nil
}

func (c *Currency) fetchOnce(ctx context.Context, req *dto.CurrencyRequestDTO) (map[string]float64, error) {
	messageURL, err := c.buildURL(req)
	if err != nil {
		return nil, err
	}

	c.logger.DebugContext(ctx, "sending request", slog.String("url", messageURL))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, messageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	// TODO: Вынести в конфиг формат xml
	httpReq.Header.Add("Accept", "application/vnd.sdmx.structurespecificdata+xml;version=2.1")

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start).Seconds()

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		c.metrics.RequestDuration.WithLabelValues("network_error").Observe(duration)
		c.metrics.Errors.WithLabelValues("network").Inc()
		return nil, Retriable(fmt.Errorf("http do: %w", err))
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Debug("body close failed", slog.Any("error", err))
		}
	}()

	switch {
	case resp.StatusCode == http.StatusOK:
		c.metrics.RequestDuration.WithLabelValues("success").Observe(duration)
	case resp.StatusCode == http.StatusTooManyRequests:
		c.metrics.RequestDuration.WithLabelValues("429").Observe(duration)
		c.metrics.Errors.WithLabelValues("rate_limited").Inc()
		return nil, Retriable(fmt.Errorf("rate limited: %s", resp.Status))
	case resp.StatusCode >= 500:
		c.metrics.RequestDuration.WithLabelValues("5xx").Observe(duration)
		c.metrics.Errors.WithLabelValues(fmt.Sprintf("http_%d", resp.StatusCode)).Inc()
		return nil, Retriable(fmt.Errorf("server error: %s", resp.Status))
	default:
		c.metrics.RequestDuration.WithLabelValues("4xx").Observe(duration)
		c.metrics.Errors.WithLabelValues(fmt.Sprintf("http_%d", resp.StatusCode)).Inc()
		return nil, fmt.Errorf("client error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if len(bytes.TrimSpace(body)) == 0 {
		c.logger.InfoContext(ctx, "ecb returned empty body — no data for requested interval",
			slog.String("url", messageURL))
		return map[string]float64{}, nil
	}

	points, err := extractObs(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	rates := make(map[string]float64, len(points))
	for _, p := range points {
		rates[p.Date.Format("2006-01-02")] = float64(p.Value)
	}
	return rates, nil
}

func extractObs(body io.Reader) ([]dto.RateRecordDTO, error) {
	var data StructureSpecificData
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	RateRecords := make([]dto.RateRecordDTO, 0, len(data.DataSet.Series.Obs))
	for _, obs := range data.DataSet.Series.Obs {
		if obs.TimePeriod == "" || obs.ObsValue == "" {
			continue
		}

		date, err := time.Parse("2006-01-02", obs.TimePeriod)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %q: %w", obs.TimePeriod, err)
		}

		val, err := strconv.ParseFloat(obs.ObsValue, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %q: %w", obs.ObsValue, err)
		}

		RateRecords = append(RateRecords, dto.RateRecordDTO{
			Date:  date,
			Value: float32(val),
		})

	}

	return RateRecords, nil
}
