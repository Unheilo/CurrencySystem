package migrator

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var MigrationsFS embed.FS

type Migrator struct {
	srcDriver source.Driver
}

func MustGetNewMigrator(sqlFiles embed.FS, dirName string) *Migrator {
	d, err := iofs.New(sqlFiles, dirName)
	if err != nil {
		panic(err)
	}
	return &Migrator{
		srcDriver: d,
	}
}

func (m *Migrator) ApplyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("unable to create db instance: %v", err)
	}

	migrator, err := migrate.NewWithInstance("migration_embeded_sql_files", m.srcDriver, "psql_db", driver)
	if err != nil {
		return fmt.Errorf("unable to create migration: %v", err)
	}

	defer func() {
		sourceErr, dbErr := migrator.Close()
		if sourceErr != nil {
			fmt.Printf("error closing migration source: %v\n", sourceErr)
		}
		if dbErr != nil {
			fmt.Printf("error closing migration database: %v\n", dbErr)
		}
	}()

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("unable to apply migrations %v", err)
	}

	return nil
}
