package worker

import (
	"context"
	"fmt"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/service"
	"time"

	"github.com/go-co-op/gocron"
)

type Currency struct {
	currencyService service.Currency
	cron            *gocron.Scheduler
	schedule        string
	baseCurrency    string
	targetCurrency  string
	logger          *slog.Logger
}

func NewCurrency(
	cfg config.WorkerConfig,
	service service.Currency,
	cron *gocron.Scheduler,
	logger *slog.Logger,
) *Currency {
	return &Currency{
		currencyService: service,
		cron:            cron,
		schedule:        cfg.Schedule,
		baseCurrency:    cfg.CurrencyPair.BaseCurrency,
		targetCurrency:  cfg.CurrencyPair.TargetCurrency,
		logger:          logger,
	}
}

func (w *Currency) StartFetchingCurrencyRates() error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5) // TODO: move to config
		defer cancel()

		currencyData := dto.CurrencyRequestDTO{
			BaseCurrency:   w.baseCurrency,
			TargetCurrency: w.targetCurrency,
			DateFrom:       time.Now().UTC(),
			DateTo:         time.Now().UTC(),
		}

		err := w.currencyService.FetchAndSaveCurrencyRates(ctx, &currencyData)

		if err != nil {
			w.logger.Error("Failed to fetch currency rate immediately on startup",
				slog.Time("timestamp", time.Now().UTC()),
				slog.Any("error", err))
		}

	}()

	_, err := w.cron.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5) //TODO: move to config
		defer cancel()

		err := w.currencyService.FetchAndSaveCurrencyRates(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   w.baseCurrency,
			TargetCurrency: w.targetCurrency,
		})
		if err != nil {
			w.logger.Error("Failed to fetch currency rate on schedule",
				slog.Time("timestamp", time.Now().UTC()),
				slog.Any("error", err),
				slog.String("schedule", w.schedule))
		}
	})

	if err != nil {
		return fmt.Errorf("cron.Do: %w", err)
	}

	w.cron.StartBlocking()

	return nil
}
