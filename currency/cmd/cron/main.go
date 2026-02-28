package main

import (
	"fmt"
	"log"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	"my-currency-service/currency/internal/logger"
	"my-currency-service/currency/internal/repository"
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
		panic(err)
	}

	// repo
	repoPrototype := repository.NewPostgresRepository(conn)
	repo, err := repository.NewCurrency(repoPrototype)
	if err != nil {
		return fmt.Errorf("error creating repository: %v", err)
	}

	//logger
	log := logger.SetupLogger(cfg.Service.Env)

	//client

	//cron

	return nil
}
