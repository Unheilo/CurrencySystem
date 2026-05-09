package interceptors

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	interceptor := Recovery(log)

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("boom")
	}

	resp, err := interceptor(context.Background(), nil, info, handler)
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRecovery_PassesThrough(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	interceptor := Recovery(log)
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	resp, err := interceptor(context.Background(), nil, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}
