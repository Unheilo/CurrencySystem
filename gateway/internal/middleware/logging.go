package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

type requestInfo struct {
	user string
}

type requestInfoKey struct{}

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			info := &requestInfo{}
			ctx := context.WithValue(r.Context(), requestInfoKey{}, info)
			r = r.WithContext(ctx)

			next.ServeHTTP(rw, r)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestIDFromContext(r.Context())),
			}
			if info.user != "" {
				attrs = append(attrs, slog.String("user", info.user))
			}
			log.InfoContext(r.Context(), "http", attrs...)
		})
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func setRequestUser(ctx context.Context, sub string) {
	if info, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		info.user = sub
	}
}
