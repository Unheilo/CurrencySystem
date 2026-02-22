package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"my-currency-service/currency/internal/dto"
	"time"

	_ "github.com/lib/pq"
)

// PostgresRepository implements ExchangeRateRepository for PostgreSQL.
type PostgresRepository struct {
	DB *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{DB: db}
}

type CurrencyRate struct {
	Date time.Time
	Rate float32
}

func (repo *PostgresRepository) Save(
	ctx context.Context,
	date time.Time,
	baseCurrency string,
	rates map[string]float64,
) error {
	ratesJSON, err := json.Marshal(rates)
	if err != nil {
		return fmt.Errorf("failed to marshal currency rates: %w", err)
	}

	_, err = repo.DB.ExecContext(
		ctx,
		`INSERT INTO exchange_rates (date, base_currency, currency_rates)
				VALUES ($1, $2, $3)
				ON CONFLICT (date, base_currency)
				DO UPDATE SET 
				currency_rates = exchange_rates.currency_rates || EXCLUDED.currency_rates`,
		date, baseCurrency, ratesJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save exchange rates: %w", err)
	}
	return nil
}

func (repo *PostgresRepository) FindInInterval(
	ctx context.Context,
	dto *dto.CurrencyRequestDTO,
) ([]CurrencyRate, error) {
	query := `
		SELECT date, (currency_rates ->> $1)::float 
		FROM exchange_rates
		WHERE date::date BETWEEN $2 AND $3 AND base_currency = $4
	`

	rows, err := repo.DB.QueryContext(
		ctx,
		query,
		dto.TargetCurrency,
		dto.DateFrom.Format("2006-01-02"),
		dto.DateTo.Format("2006-01-02"),
		dto.BaseCurrency,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query exchange rates: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var rates []CurrencyRate
	for rows.Next() {
		var rate CurrencyRate
		if err := rows.Scan(&rate.Date, &rate.Rate); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return rates, nil
}
