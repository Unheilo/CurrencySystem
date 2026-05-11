package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey struct{ name string }

var RequestIDKey = ctxKey{"request_id"}

const HeaderRequestID = "X-Request-ID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(HeaderRequestID, id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}
