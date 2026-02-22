package main

import (
	"database/sql"
	"fmt"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	migrator "my-currency-service/currency/internal/migrations"
)

func main() {

	// Recover Migrator
	m := migrator.MustGetNewMigrator(migrator.MigrationsFS, ".")

	// Get the DB instance
	cfg := config.MustLoad()

	conn, err := db.NewDatabaseConnection(cfg.Database)
	if err != nil {
		panic(err)
	}

	defer func(conn *sql.DB) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	err = m.ApplyMigrations(conn)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Migrations applied!!")
}
