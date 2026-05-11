package handler

import (
	"encoding/json"
	"log/slog"
	client "my-currency-service/gateway/internal/clients"
	"my-currency-service/gateway/internal/dto"
	"net/http"
	"time"
)

type RateHandler struct {
	log    *slog.Logger
	client *client.Currency
}

func NewRateHandler(log *slog.Logger, c *client.Currency) *RateHandler {
	return &RateHandler{log: log, client: c}
}

type getRateQuery struct {
	Currency string
	From     time.Time
	To       time.Time
	Base     string
}

type invalidQueryError struct{ msg string }

func wrap(msg string) error {
	return &invalidQueryError{msg: msg}
}

func (e *invalidQueryError) Error() string { return e.msg }

func (e *invalidQueryError) Is(target error) bool {
	return target == errInvalidQuery
}

func parseRateQuery(r *http.Request) (getRateQuery, error) {
	q := r.URL.Query()
	currency := q.Get("currency")
	if currency == "" {
		return getRateQuery{}, wrap("currency is required")
	}
	if len(currency) != 3 {
		return getRateQuery{}, wrap("currency must be 3 letters")
	}

	fromStr := q.Get("date_from")
	if fromStr == "" {
		return getRateQuery{}, wrap("date_from is required")
	}
	from, err := time.Parse("2006-01-02", fromStr)

	if err != nil {
		return getRateQuery{}, wrap("date_from bust be YYYY-MM-DD")
	}

	toStr := q.Get("date_to")
	if toStr == "" {
		return getRateQuery{}, wrap("date_to is required")
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return getRateQuery{}, wrap("date_to must be YYYY-MM-DD")
	}

	if from.After(to) {
		return getRateQuery{}, wrap("date_from must be <= date_to")
	}

	if time.Since(from) > 365*24*time.Hour*5 {
		return getRateQuery{}, wrap("date_from cannot be eqrlier than 5 years ago")
	}

	return getRateQuery{
		Currency: currency,
		From:     from,
		To:       to,
		Base:     q.Get("base"),
	}, nil
}

func (h *RateHandler) Get(w http.ResponseWriter, r *http.Request) {
	q, err := parseRateQuery(r)
	if err != nil {
		WriteError(w, r, h.log, err)
		return
	}

	rates, err := h.client.GetRate(r.Context(), q.Currency, q.Base, q.From, q.To)
	if err != nil {
		WriteError(w, r, h.log, err)
		return
	}

	resp := dto.RateResponse{
		Currency: q.Currency,
		Rates:    make([]dto.RatePoint, 0, len(rates)),
	}
	for _, rate := range rates {
		resp.Rates = append(resp.Rates, dto.RatePoint{
			Date: rate.Date.Format("2006-01-02"),
			Rate: rate.Rate,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
