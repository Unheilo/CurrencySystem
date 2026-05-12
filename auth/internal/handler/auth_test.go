package handler_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"my-currency-service/auth/internal/config"
	"my-currency-service/auth/internal/handler"
	"my-currency-service/auth/internal/users"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-definitely-long-enough"

func newServer(t *testing.T) *handler.Server {
	t.Helper()
	store := users.New([]config.User{
		{Login: "alice", Password: "wonderland"},
	})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return handler.NewServer(log, store, []byte(testSecret), 5*time.Minute, "test-issuer")
}

func postJSON(t *testing.T, s *handler.Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	s.Auth(rw, r)
	return rw
}

func TestAuth_Success(t *testing.T) {
	s := newServer(t)
	rw := postJSON(t, s, `{"login":"alice","password":"wonderland"}`)

	require.Equal(t, http.StatusOK, rw.Code)
	var resp struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Token)

	claims := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(resp.Token, &claims,
		func(t *jwt.Token) (any, error) { return []byte(testSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}),
	)
	require.NoError(t, err)
	assert.Equal(t, "alice", claims.Subject)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestAuth_WrongPassword(t *testing.T) {
	rw := postJSON(t, newServer(t), `{"login":"alice","password":"nope"}`)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_UnknownUser(t *testing.T) {
	rw := postJSON(t, newServer(t), `{"login":"ghost","password":"any"}`)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_MissingFields(t *testing.T) {
	rw := postJSON(t, newServer(t), `{"login":"alice"}`)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAuth_MalformedJSON(t *testing.T) {
	rw := postJSON(t, newServer(t), `{"login":}`)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAuth_UnknownField(t *testing.T) {
	rw := postJSON(t, newServer(t),
		`{"login":"alice","password":"wonderland","role":"admin"}`)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAuth_WrongMethod(t *testing.T) {
	s := newServer(t)
	r := httptest.NewRequest(http.MethodGet, "/auth", nil)
	rw := httptest.NewRecorder()
	s.Auth(rw, r)
	assert.Equal(t, http.StatusMethodNotAllowed, rw.Code)
}

func TestAuth_WrongContentType(t *testing.T) {
	s := newServer(t)
	r := httptest.NewRequest(http.MethodPost, "/auth",
		strings.NewReader(`{"login":"alice","password":"wonderland"}`))
	r.Header.Set("Content-Type", "text/plain")
	rw := httptest.NewRecorder()
	s.Auth(rw, r)
	assert.Equal(t, http.StatusUnsupportedMediaType, rw.Code)
}
