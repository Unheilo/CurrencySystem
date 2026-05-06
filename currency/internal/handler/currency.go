package handler

import (
	"context"
	"fmt"
	"my-currency-service/currency/internal/dto"
	"my-currency-service/pkg/currency"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s CurrencyServer) GetRate(ctx context.Context, request *currency.GetRateRequest) (*currency.GetRateResponse, error) {

	err := rateRequestValidation(request)
	if err != nil {
		return nil, err
	}

	//start := time.Now()
	reqDTO := dto.CurrencyRequestDTOFromProtobuf(request, s.defaultBaseCurrency)

	// TODO: метрики в мидлвары
	//s.requestCount.WithLabelValues("GetRate").Inc()
	rates, err := s.service.GetCurrencyRatesInInterval(ctx, reqDTO)
	if err != nil {
		return nil, fmt.Errorf("service.GetCurrencyRatesInInterval: %w", err)
	}

	if len(rates) == 0 {
		return nil, status.Errorf(codes.NotFound, "no rates in the given period")
	}

	rateRecords := make([]*currency.RateRecord, len(rates))
	for i, rate := range rates {
		rateRecords[i] = &currency.RateRecord{
			Date: timestamppb.New(rate.Date),
			Rate: rate.Rate,
		}
	}

	//s.requestDuration.WithLabelValues("GetExchangeRate").Observe(time.Since(start).Seconds())
	out := &currency.GetRateResponse{
		Currency: reqDTO.TargetCurrency,
		Rates:    rateRecords,
	}

	// TODO: add from future interceptors logging, tracing, metrics
	_ = time.Now()

	return out, nil
}

func rateRequestValidation(req *currency.GetRateRequest) error {

	if req.GetCurrency() == "" {
		return status.Error(codes.InvalidArgument, "exchange currency is required")
	}

	if req.GetDataFrom() == nil {
		return status.Error(codes.InvalidArgument, "data from is required")
	}

	if err := req.GetDataFrom().CheckValid(); err != nil {
		return status.Error(codes.InvalidArgument, "data_from is invalid")
	}

	if req.GetDateTo() == nil {
		return status.Error(codes.InvalidArgument, "data to is required")
	}

	if err := req.GetDateTo().CheckValid(); err != nil {
		return status.Error(codes.InvalidArgument, "data_to is invalid")
	}

	if req.GetDateTo().AsTime().Before(req.GetDataFrom().AsTime()) {
		return status.Error(codes.InvalidArgument, "date_to must be after data_from")
	}

	return nil

}
