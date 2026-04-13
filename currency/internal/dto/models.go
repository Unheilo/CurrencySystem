package dto

import (
	"my-currency-service/pkg/currency"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: Вынести эти тестовые константы в конфиг
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

type CurrencyRequestDTO struct {
	BaseCurrency   string
	TargetCurrency string
	DateFrom       time.Time
	DateTo         time.Time
}

type CurrencyResponseDTO struct {
	Currency string
	Rates    []RateRecordDTO
}

type RateRecordDTO struct {
	Date  time.Time
	Value float32
}

func CurrencyRequestDTOFromProtobuf(req *currency.GetRateRequest, baseCurrency string) *CurrencyRequestDTO {
	return &CurrencyRequestDTO{
		BaseCurrency:   baseCurrency,
		TargetCurrency: req.Currency,
		DateFrom:       req.DataFrom.AsTime(),
		DateTo:         req.DateTo.AsTime(),
	}
}

func (dto *CurrencyResponseDTO) ToProtobuf() *currency.GetRateResponse {
	rateRecords := make([]*currency.RateRecord, 0, len(dto.Rates))

	for _, record := range dto.Rates {
		rateRecords = append(
			rateRecords, &currency.RateRecord{
				Date: timestamppb.New(record.Date),
				Rate: record.Value,
			},
		)
	}

	return &currency.GetRateResponse{
		Currency: dto.Currency,
		Rates:    rateRecords,
	}
}
