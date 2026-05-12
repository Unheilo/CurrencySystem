package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type userCtxKey struct{}

var UserKey = userCtxKey{}

type AuthConfig struct {
	Secret []byte
	Issuer string
}

func Auth(log *slog.Logger, cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := extractBearer(r)
			if tok == "" {
				writeUnauth(log, w, r, "missing token")
				return
			}
			claims, err := validate(tok, cfg)
			if err != nil {
				writeUnauth(log, w, r, err.Error())
				return
			}
			setRequestUser(r.Context(), claims.Subject)
			ctx := context.WithValue(r.Context(), UserKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(UserKey).(string)
	return v, ok
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}

	const prefix = "Bearer "

	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimPrefix(h, prefix)
}

func validate(tok string, cfg AuthConfig) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tok, claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return cfg.Secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithLeeway(10*time.Second),
	) // clock skew support
	if err != nil {
		return nil, err
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}
	if cfg.Issuer != "" && claims.Issuer != cfg.Issuer {
		return nil, errors.New("unexpected issuer")
	}
	return claims, nil
}

func writeUnauth(log *slog.Logger, w http.ResponseWriter, r *http.Request, reason string) {
	log.InfoContext(r.Context(), "auth failed", slog.String("reason", reason))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="currency"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHENTICATED","message":"unauthorized"}}`))
}
