package interceptors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Metrics(
	reqCount *prometheus.CounterVec,
	reqDuration *prometheus.HistogramVec,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		reqCount.WithLabelValues(info.FullMethod, code).Inc()
		reqDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		return resp, err
	}
}
