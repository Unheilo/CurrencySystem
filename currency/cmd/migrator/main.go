package main

import (
	"embed"
	"fmt"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/db"
	migrator "my-currency-service/currency/internal/migrations"
)

const migrationsDir = "currency/internal/migrations"

//go:embed my-currency-service/currency/internal/migrations/*.sql
var MigrationsFS embed.FS

func main() {
	// --- (1) ----
	// Recover Migrator
	migrator := migrator.MustGetNewMigrator(MigrationsFS, migrationsDir)

	// --- (2) ----
	// Get the DB instance
	//TODO: поправить на нормальный коннект

	cfg := config.MustLoad()

	conn, err := db.NewDatabaseConnection(cfg.Database)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	err = migrator.ApplyMigrations(conn)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Migrations applied!!")
}
