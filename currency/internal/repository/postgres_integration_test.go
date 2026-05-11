//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"my-currency-service/currency/internal/dto"
	migrator "my-currency-service/currency/internal/migrations"
	"my-currency-service/currency/internal/repository"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func newPostgres(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pgC, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgC.Terminate(context.Background())
	})

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	migrateDB, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, migrateDB.Ping())

	m := migrator.MustGetNewMigrator(migrator.MigrationsFS, ".")
	require.NoError(t, m.ApplyMigrations(migrateDB))
	_ = migrateDB.Close()

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func newTestMetrics() repository.RepositoryMetrics {
	return repository.RepositoryMetrics{
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "test_db_query_duration_seconds",
				Help: "test",
			},
			[]string{"operation"},
		),
	}
}

func TestPostgresRepository_SaveFind(t *testing.T) {
	// exclude t.Parallel() because docker usage is expensive
	db := newPostgres(t)
	repo := repository.NewPostgresRepository(db, newTestMetrics())

	ctx := context.Background()
	date := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)

	t.Run("insert new", func(t *testing.T) {
		err := repo.Save(ctx, date, "USD", map[string]float64{"EUR": 1.10, "GBP": 0.85})
		require.NoError(t, err)

		rates, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "USD",
			TargetCurrency: "EUR",
			DateFrom:       date,
			DateTo:         date,
		})
		require.NoError(t, err)
		require.Len(t, rates, 1)
		assert.Equal(t, float32(1.10), rates[0].Rate)
	})

	t.Run("upsert merges JSONB", func(t *testing.T) {
		// adding new exchange currency to existent write
		err := repo.Save(ctx, date, "USD", map[string]float64{"JPY": 145.0})
		require.NoError(t, err)

		jpy, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "USD",
			TargetCurrency: "JPY",
			DateFrom:       date,
			DateTo:         date,
		})
		require.NoError(t, err)
		require.Len(t, jpy, 1)
		assert.Equal(t, float32(145.0), jpy[0].Rate)

		// check that old writes are preserved
		eur, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "USD",
			TargetCurrency: "EUR",
			DateFrom:       date,
			DateTo:         date,
		})
		require.NoError(t, err)
		require.Len(t, eur, 1)
	})

	t.Run("find in interval filters dates correctly", func(t *testing.T) {
		d1 := time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC)
		d2 := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Save(ctx, d1, "USD", map[string]float64{"EUR": 1.11}))
		require.NoError(t, repo.Save(ctx, d2, "USD", map[string]float64{"EUR": 1.12}))

		rates, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "USD",
			TargetCurrency: "EUR",
			DateFrom:       d1,
			DateTo:         d2,
		})
		require.NoError(t, err)
		assert.Len(t, rates, 2)
	})

	t.Run("different base_currency isolated", func(t *testing.T) {
		d := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
		require.NoError(t, repo.Save(ctx, d, "USD", map[string]float64{"EUR": 1.10}))
		require.NoError(t, repo.Save(ctx, d, "GBP", map[string]float64{"EUR": 1.25}))

		usd, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "USD",
			TargetCurrency: "EUR",
			DateFrom:       d,
			DateTo:         d,
		})
		require.NoError(t, err)
		require.Len(t, usd, 1)
		assert.InDelta(t, 1.10, usd[0].Rate, 0.001)

		gbp, err := repo.FindInInterval(ctx, &dto.CurrencyRequestDTO{
			BaseCurrency:   "GBP",
			TargetCurrency: "EUR",
			DateFrom:       d,
			DateTo:         d,
		})
		require.NoError(t, err)
		require.Len(t, gbp, 1)
		assert.InDelta(t, 1.25, gbp[0].Rate, 0.001)
	})
}
