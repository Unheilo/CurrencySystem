package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.ErrorContext(r.Context(), "panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", string(debug.Stack())),
					)
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"internal server error"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
