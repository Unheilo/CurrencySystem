package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	"my-currency-service/currency/internal/logger"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/currency/internal/service"
	"my-currency-service/currency/internal/worker"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}

}

func run() error {
	cfg := config.MustLoad()

	conn, err := db.NewDatabaseConnection(cfg.Database)

	if err != nil {
		return fmt.Errorf("error creating repository: %v", err)
	}

	// repo
	repo := repository.NewPostgresRepository(conn)

	//TODO: непонятно как тут с интерфейсами логами и надо ли под это делать
	//repo, err := repository.NewCurrency(repoPrototype)

	//loggerInstance
	loggerInstance, err := logger.SetupLogger(cfg.Service.Env)
	if err != nil {
		return fmt.Errorf("error creating logger: %v", err)
	}

	//client
	client, err := currency.New(cfg.API, loggerInstance)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	//svc
	svc := service.NewCurrency(repo, client, loggerInstance)

	//cron
	c := gocron.NewScheduler(time.UTC)

	//TODO: сделать worker модуль и от него реализовать cron
	currencyWorker := worker.NewCurrency(cfg.Worker, svc, c, loggerInstance)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := currencyWorker.StartFetchingCurrencyRates(); err != nil {
		loggerInstance.Error("Error start fetching currency rates",
			slog.Time("timestamp", time.Now().UTC()),
			slog.Any("error", err))
	}

	<-ctx.Done()

	loggerInstance.Info("Shutting down gracefully, press Ctrl+C again to force")

	return nil
}
