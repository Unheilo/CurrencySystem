package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateHandler_MissingCurrency(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate?date_from=2025-01-01&date_to=2025-01-10", nil)
	// ... собрать router с handler + mocked client, вызвать ServeHTTP
	// проверить rw.Code == 400 и body содержит "currency is required"
	_ = req
	t.Skip("self-exercise")
}
