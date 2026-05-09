package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Logging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		lvl := slog.LevelInfo
		if err != nil {
			lvl = slog.LevelWarn
		}
		log.LogAttrs(ctx, lvl, "grpc call",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", time.Since(start)),
			slog.String("code", status.Code(err).String()),
			slog.String("request_id", RequestIDFromContext(ctx)),
		)
		return resp, err
	}
}
