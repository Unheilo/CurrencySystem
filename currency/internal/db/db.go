package db

import (
	"database/sql"
	"fmt"
	"my-currency-service/currency/internal/config"
)

func NewDatabaseConnection(cfg config.DatabaseConfig) (*sql.DB, string, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open database: #{err}")
	}

	if err := db.Ping(); err != nil {
		return nil, "", fmt.Errorf("failed to connect to database: #{err}")
	}

	return db, dsn, nil
}
