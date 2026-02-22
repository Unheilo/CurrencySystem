package dto

import "time"

const (
	DefaultBaseCurrency = "USD"
)

const (
	DefaultBaseExchangeCurrency = "EUR"
)

const (
	DefaultEurounionExchangeAdress = "https://data-api.ecb.europa.eu/service/data/EXR/D.%s.%s.SP00.A?startPeriod=%s&endPeriod=%s"
)

type ExchangeRateRequestDTO struct {
	BasicCurrency    string
	ExchangeCurrency string
	StartPeriod      string
	EndPeriod        string
}

type CurrencyResponseDTO struct {
	Currency string
	Rates    []RateRecordDTO
}

type RateRecordDTO struct {
	Date  time.Time
	Value float32
}
