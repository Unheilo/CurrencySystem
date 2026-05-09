package interceptors

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey struct{ name string }

var requestIDKey = ctxKey{"request_id"}

const MDHeaderRequestID = "x-request-id"

func RequestID() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		id := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(MDHeaderRequestID); len(vals) > 0 {
				id = vals[0]
			}
		}
		if id == "" {
			id = uuid.NewString()
		}
		ctx = context.WithValue(ctx, requestIDKey, id)
		return handler(ctx, req)
	}
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
