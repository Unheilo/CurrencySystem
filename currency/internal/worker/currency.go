package worker

import (
	"context"
	"fmt"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
)

type CurrencyService interface {
	FetchAndSaveCurrencyRates(ctx context.Context, req *dto.CurrencyRequestDTO) error
}

type WorkerMetrics struct {
	FetchJobDuration *prometheus.HistogramVec
	FetchJobRuns     *prometheus.CounterVec
}

type Currency struct {
	currencyService CurrencyService
	cron            *gocron.Scheduler
	schedule        string
	timeout         time.Duration
	baseCurrency    string
	targetCurrency  string
	logger          *slog.Logger
	metrics         WorkerMetrics
}

func NewCurrency(
	cfg config.WorkerConfig,
	service CurrencyService,
	cron *gocron.Scheduler,
	logger *slog.Logger,
	metrics WorkerMetrics,
) *Currency {
	return &Currency{
		currencyService: service,
		cron:            cron,
		schedule:        cfg.Schedule,
		timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		baseCurrency:    cfg.CurrencyPair.BaseCurrency,
		targetCurrency:  cfg.CurrencyPair.TargetCurrency,
		logger:          logger,
		metrics:         metrics,
	}
}

func (w *Currency) runOnce(ctx context.Context, trigger string) {
	start := time.Now()

	err := w.currencyService.FetchAndSaveCurrencyRates(ctx, &dto.CurrencyRequestDTO{
		BaseCurrency:   w.baseCurrency,
		TargetCurrency: w.targetCurrency,
		DateFrom:       time.Now().UTC(),
		DateTo:         time.Now().UTC(),
	})

	w.metrics.FetchJobDuration.WithLabelValues(trigger).Observe(time.Since(start).Seconds())

	if err != nil {
		w.metrics.FetchJobRuns.WithLabelValues("failure").Inc()
		w.logger.Error("fetch job failed",
			slog.String("trigger", trigger),
			slog.Any("error", err))
		return
	}
	w.metrics.FetchJobRuns.WithLabelValues("success").Inc()
}

func (w *Currency) StartFetchingCurrencyRates(ctx context.Context) error {
	initialCtx, cancel := context.WithTimeout(ctx, w.timeout)
	go func() {
		defer cancel()
		w.runOnce(initialCtx, "startup")
	}()

	_, err := w.cron.Cron(w.schedule).Do(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), w.timeout)
		defer ccancel()
		w.runOnce(cctx, "cron")
	})
	if err != nil {
		return fmt.Errorf("cron.Do: %w", err)
	}

	w.cron.StartAsync()
	return nil
}

// func (w *Currency) StartFetchingCurrencyRates(ctx context.Context) error {
// 	initialCtx, cancel := context.WithTimeout(ctx, w.timeout)
// 	go func() {
// 		defer cancel()

// 		currencyData := dto.CurrencyRequestDTO{
// 			BaseCurrency:   w.baseCurrency,
// 			TargetCurrency: w.targetCurrency,
// 			DateFrom:       time.Now().UTC(),
// 			DateTo:         time.Now().UTC(),
// 		}

// 		err := w.currencyService.FetchAndSaveCurrencyRates(initialCtx, &currencyData)

// 		if err != nil {
// 			w.logger.Error("Failed to fetch currency rate immediately on startup",
// 				slog.Time("timestamp", time.Now().UTC()),
// 				slog.Any("error", err))
// 		}

// 	}()

// 	_, err := w.cron.Cron(w.schedule).Do(func() {
// 		cctx, ccancel := context.WithTimeout(context.Background(), w.timeout)
// 		defer ccancel()

// 		err := w.currencyService.FetchAndSaveCurrencyRates(cctx, &dto.CurrencyRequestDTO{
// 			BaseCurrency:   w.baseCurrency,
// 			TargetCurrency: w.targetCurrency,
// 		})
// 		if err != nil {
// 			w.logger.Error("Failed to fetch currency rate on schedule",
// 				slog.Time("timestamp", time.Now().UTC()),
// 				slog.Any("error", err),
// 				slog.String("schedule", w.schedule))
// 		}
// 	})

// 	if err != nil {
// 		return fmt.Errorf("cron.Do: %w", err)
// 	}

// 	w.cron.StartAsync()

// 	return nil
// }

func (w *Currency) Stop() error {
	w.cron.Stop()
	return nil
}
