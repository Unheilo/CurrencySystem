package middleware_test

import (
	"io"
	"log/slog"
	"my-currency-service/gateway/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func makeToken(t *testing.T, secret []byte, exp time.Duration, alg jwt.SigningMethod) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(alg, claims)
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAuth_MissingHeader(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := middleware.Auth(log, middleware.AuthConfig{Secret: []byte("s")})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("should not call next") }),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_WrongPrefix(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := middleware.Auth(log, middleware.AuthConfig{Secret: []byte("s")})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("should not call next") }),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic xyz")
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_Expired(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := []byte("my-secret")
	tok := makeToken(t, secret, -1*time.Hour, jwt.SigningMethodHS256)

	h := middleware.Auth(log, middleware.AuthConfig{Secret: secret})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("should not call next") }),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_WrongSecret(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	tok := makeToken(t, []byte("other"), 1*time.Hour, jwt.SigningMethodHS256)

	h := middleware.Auth(log, middleware.AuthConfig{Secret: []byte("mine")})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("should not call next") }),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAuth_Success(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := []byte("my-secret")
	tok := makeToken(t, secret, 1*time.Hour, jwt.SigningMethodHS256)

	called := false
	h := middleware.Auth(log, middleware.AuthConfig{Secret: secret})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, ok := middleware.UserFromContext(r.Context())
			assert.True(t, ok)
			assert.Equal(t, "user-1", sub)
			called = true
		}),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestAuth_AlgNone(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := []byte("my-secret")

	claims := jwt.RegisteredClaims{
		Subject:   "evil",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	h := middleware.Auth(log, middleware.AuthConfig{Secret: secret})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("should not call next") }),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+s)
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}
