package handler

import (
	"context"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/currency/internal/repository"
	"time"
)

type CurrencyServiceMock struct {
	expectedMethods []string
}

func (s *CurrencyServiceMock) ExpectCall(method string) {
	s.expectedMethods = append(s.expectedMethods, method)
}

func (s *CurrencyServiceMock) GetCurrencyRatesInterval(ctx context.Context, reqDTO *dto.CurrencyRequestDTO) ([]repository.CurrencyRate, error) {
	found := false
	for _, m := range s.expectedMethods {
		if m == "GetCurrencyRatesInInterval" {
			found = true
		}
	}

	if !found {
		panic("GetCurrencyRatesInInterval call not expected")
	}

	res := []repository.CurrencyRate{
		{
			Date: time.Now(),
			Rate: 228,
		},
	}

	return res, nil
}

func (s *CurrencyServiceMock) FetchAndSaveCurrencyRates(ctx context.Context, baseCurrency string) error {
	return nil
}
