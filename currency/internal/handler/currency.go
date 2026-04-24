package handler

import (
	"context"
	"fmt"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/pkg/currency"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s CurrencyServer) GetRate(ctx context.Context, request *currency.GetRateRequest) (*currency.GetRateResponse, error) {

	// ValidationErr := rateRequestValidation(request)
	// if err != nil {
	// 	return nil, ValidationErr
	// }

	start := time.Now()
	reqDTO := dto.CurrencyRequestDTOFromProtobuf(request)

	// TODO: метрики в мидлвары
	s.requestCount.WithLabelValues("GetRate").Inc()
	rates, err := s.service.GetCurrencyRatesInInterval(ctx, reqDTO)
	if err != nil {
		return nil, fmt.Errorf("service.GetCurrencyRatesInInterval: %w", err)
	}

	rateRecords := make([]*currency.RateRecord, len(rates))
	for i, rate := range rates {
		rateRecords[i] = &currency.RateRecord{
			Date: timestamppb.New(rate.Date),
			Rate: rate.Rate,
		}
	}

	s.requestDuration.WithLabelValues("GetExchangeRate").Observe(time.Since(start).Seconds())
	return &currency.GetRateResponse{
		Currency: reqDTO.TargetCurrency,
		Rates:    rateRecords,
	}, nil
}

// func rateRequestValidation(req *currecy.GetRateRequest) error {

// 	if req.GetBaseCurrency() == "" {
// 		return nil, status.Error(codes.InvalidArgument, "currency is required")
// 	}

// 	if req.GetCurrency() == "" {
// 		return nil, status.Error(codes.InvalidArgument,"exchange currency is required")
// 	}

// 	if req.GetDataFrom()

// 	if req.

// }
