package main

import (
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	"my-currency-service/currency/internal/repository"
)

func main() {

	cfg := config.MustLoad()

	conn, err := db.NewDatabaseConnection(cfg.Database)
	if err != nil {
		panic(err)
	}

	// repo
	repoPrototype := repository.NewPostgresRepository(conn)
	repo := repository.NewCurrency(repoPrototype)

	//logger

	//client

	//cron

}
