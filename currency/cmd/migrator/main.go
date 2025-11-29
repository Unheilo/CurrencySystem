package main

import (
	"embed"
	migrator "my-currency-service/currency/internal/migrations"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var MigrationsFS embed.FS

func main() {
	// --- (1) ----
	// Recover Migrator
	migrator := migrator.MustGetNewMigrator(MigrationsFS, migrationsDir)

	// --- (2) ----
	// Get the DB instance
	connectionStr := 

}
