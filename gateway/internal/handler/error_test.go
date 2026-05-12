package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func decode(t *testing.T, body []byte) ErrorResponse {
	t.Helper()
	var out ErrorResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestWriteError_LeakyUnknownIsMasked(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	leakyErr := status.Error(codes.Unknown, "sql: column \"secret\" does not exist")

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WriteError(rw, r, log, leakyErr)

	body := decode(t, rw.Body.Bytes())
	assert.Equal(t, "internal server error", body.Error.Message,
		"Unknown leaks SQL detail to client")
	assert.Equal(t, "Unknown", body.Error.Code)
}

func TestWriteError_InternalIsMasked(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := status.Error(codes.Internal, "panic at /usr/local/go/...")

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WriteError(rw, r, log, err)

	assert.Equal(t, "internal server error", decode(t, rw.Body.Bytes()).Error.Message)
}

func TestWriteError_InvalidArgumentIsPassedThrough(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := status.Error(codes.InvalidArgument, "currency must be 3 letters")

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WriteError(rw, r, log, err)

	body := decode(t, rw.Body.Bytes())
	assert.Equal(t, "currency must be 3 letters", body.Error.Message)
	assert.Equal(t, 400, rw.Code)
}

func TestWriteError_NotFoundIsPassedThrough(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := status.Error(codes.NotFound, "no rates in the given period")

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WriteError(rw, r, log, err)

	assert.Equal(t, "no rates in the given period",
		decode(t, rw.Body.Bytes()).Error.Message)
	assert.Equal(t, 404, rw.Code)
}

func TestWriteError_PlainErrorTreatedAsInternal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := errors.New("oops: connection refused tcp 10.0.0.5:5432")

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WriteError(rw, r, log, err)

	body := decode(t, rw.Body.Bytes())
	assert.Equal(t, "internal server error", body.Error.Message)
	assert.NotContains(t, rw.Body.String(), "10.0.0.5",
		"raw error must not leak to client")
}
