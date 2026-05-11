package client

import (
	"context"
	"fmt"
	currencypb "my-currency-service/pkg/currency"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Currency struct {
	conn   *grpc.ClientConn
	client currencypb.CurrencyServiceClient
}

func NewCurrency(addr string) (*Currency, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc connect: %w", err)
	}
	return &Currency{
		conn:   conn,
		client: currencypb.NewCurrencyServiceClient(conn),
	}, nil
}

func (c *Currency) Close() error { return c.conn.Close() }

type Rate struct {
	Date time.Time
	Rate float32
}

func (c *Currency) GetRate(ctx context.Context, currency, base string, from, to time.Time) ([]Rate, error) {
	resp, err := c.client.GetRate(ctx, &currencypb.GetRateRequest{
		Currency:     currency,
		BaseCurrency: base,
		DataFrom:     timestamppb.New(from),
		DateTo:       timestamppb.New(to),
	})

	if err != nil {
		return nil, err // catch status.FromError
	}

	out := make([]Rate, 0, len(resp.Rates))
	for _, r := range resp.Rates {
		out = append(out, Rate{Date: r.Date.AsTime(), Rate: r.Rate})
	}
	return out, nil
}
