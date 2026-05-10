package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	"my-currency-service/currency/internal/logger"
	"my-currency-service/currency/internal/metrics"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/currency/internal/service"
	"my-currency-service/currency/internal/worker"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}
}

func run() error {
	cfg := config.MustLoad()

	loggerInstance, err := logger.SetupLogger(cfg.Service.Env)
	if err != nil {
		return fmt.Errorf("error creating logger: %v", err)
	}

	conn, err := db.NewDatabaseConnection(cfg.Database)
	if err != nil {
		return fmt.Errorf("error creating repository: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			loggerInstance.Error("db close", slog.Any("error", err))
		}
	}()

	m := metrics.New()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

	metricsSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Worker.MetricsPort),
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		loggerInstance.Info("Prometheus metrics server running", slog.String("addr", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			loggerInstance.Error("metrics server", slog.Any("error", err))
		}
	}()

	repo := repository.NewPostgresRepository(conn, repository.RepositoryMetrics{
		DBQueryDuration: m.DBQueryDuration,
	})

	client, err := currency.New(cfg.API, loggerInstance, currency.ECBMetrics{
		RequestDuration: m.ECBRequestDuration,
		Errors:          m.ECBErrors,
	})
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	svc := service.NewCurrency(repo, client, loggerInstance)

	c := gocron.NewScheduler(time.UTC)

	currencyWorker := worker.NewCurrency(cfg.Worker, svc, c, loggerInstance, worker.WorkerMetrics{
		FetchJobDuration: m.FetchJobDuration,
		FetchJobRuns:     m.FetchJobRuns,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := currencyWorker.StartFetchingCurrencyRates(ctx); err != nil {
		loggerInstance.Error("Error start fetching currency rates",
			slog.Time("timestamp", time.Now().UTC()),
			slog.Any("error", err))
	}

	<-ctx.Done()

	if err := currencyWorker.Stop(); err != nil {
		loggerInstance.Error("worker stop", slog.Any("error", err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		loggerInstance.Error("metrics server shutdown", slog.Any("error", err))
	}

	loggerInstance.Info("Shutting down gracefully, press Ctrl+C again to force")

	return nil
}
