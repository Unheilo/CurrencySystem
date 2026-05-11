package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/repository"
	repomocks "my-currency-service/currency/internal/repository/mocks"
	clientmocks "my-currency-service/currency/internal/service/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newSUT(t *testing.T) (*Currency, *repomocks.ExchangeRateRepository, *clientmocks.EcbClient) {
	t.Helper()
	repo := repomocks.NewExchangeRateRepository(t)
	client := clientmocks.NewEcbClient(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewCurrency(repo, client, log)
	return svc, repo, client
}

func TestGetCurrencyRatesInInterval(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		input   *dto.CurrencyRequestDTO
		setup   func(repo *repomocks.ExchangeRateRepository)
		want    []repository.CurrencyRate
		wantErr bool
	}{
		{
			name:  "happy path - returns rates",
			input: &dto.CurrencyRequestDTO{BaseCurrency: "usd", TargetCurrency: "eur", DateFrom: now, DateTo: now.AddDate(0, 0, 5)},
			setup: func(repo *repomocks.ExchangeRateRepository) {
				repo.EXPECT().FindInInterval(mock.Anything, mock.MatchedBy(func(r *dto.CurrencyRequestDTO) bool {
					return r.BaseCurrency == "USD" && r.TargetCurrency == "EUR"
				})).Return([]repository.CurrencyRate{
					{Date: now, Rate: 1.10},
					{Date: now.AddDate(0, 0, 1), Rate: 1.12},
				}, nil)
			},
			want: []repository.CurrencyRate{
				{Date: now, Rate: 1.10},
				{Date: now.AddDate(0, 0, 1), Rate: 1.12},
			},
		},
		{
			name: "repo error - wrapped and propagadted",
			input: &dto.CurrencyRequestDTO{
				BaseCurrency: "USD", TargetCurrency: "EUR",
				DateFrom: now, DateTo: now,
			},
			setup: func(repo *repomocks.ExchangeRateRepository) {
				repo.EXPECT().FindInInterval(mock.Anything, mock.Anything).
					Return(nil, errors.New("db connection refused"))
			},
			wantErr: true,
		},
		{
			name: "empty result - no error, empty slice",
			input: &dto.CurrencyRequestDTO{
				BaseCurrency: "USD", TargetCurrency: "GBR",
				DateFrom: now, DateTo: now,
			},
			setup: func(repo *repomocks.ExchangeRateRepository) {
				repo.EXPECT().FindInInterval(mock.Anything, mock.Anything).
					Return([]repository.CurrencyRate{}, nil)
			},
			want: []repository.CurrencyRate{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, _ := newSUT(t)
			tt.setup(repo)

			got, err := svc.GetCurrencyRatesInInterval(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFetchAndSaveCurrencyRates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*repomocks.ExchangeRateRepository, *clientmocks.EcbClient)
		wantErr bool
	}{
		{
			name: "happy - fetches then saves",
			setup: func(repo *repomocks.ExchangeRateRepository, cli *clientmocks.EcbClient) {
				cli.EXPECT().FetchCurrentRates(mock.Anything, mock.Anything).
					Return(map[string]float64{"2025-03-10": 1.10}, nil)
				repo.EXPECT().Save(mock.Anything, mock.Anything, "USD", mock.Anything).
					Return(nil)
			},
		},
		{
			name: "ecb fails - no save attempted",
			setup: func(repo *repomocks.ExchangeRateRepository, cli *clientmocks.EcbClient) {
				cli.EXPECT().FetchCurrentRates(mock.Anything, mock.Anything).
					Return(nil, errors.New("ecb down"))
			},
			wantErr: true,
		},
		{
			name: "save fails",
			setup: func(repo *repomocks.ExchangeRateRepository, cli *clientmocks.EcbClient) {
				cli.EXPECT().FetchCurrentRates(mock.Anything, mock.Anything).
					Return(map[string]float64{"2025-03-10": 1.10}, nil)
				repo.EXPECT().Save(mock.Anything, mock.Anything, "USD", mock.Anything).
					Return(errors.New("insert conflict"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo, cli := newSUT(t)
			tt.setup(repo, cli)

			err := svc.FetchAndSaveCurrencyRates(context.Background(), &dto.CurrencyRequestDTO{
				BaseCurrency:   "usd",
				TargetCurrency: "eur",
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
