package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	interceptor := RequestID()
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	var got string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		got = RequestIDFromContext(ctx)
		return nil, nil
	}
	_, err := interceptor(context.Background(), nil, info, handler)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

func TestRequestID_UsesIncomingMetadata(t *testing.T) {
	interceptor := RequestID()
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	var got string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		got = RequestIDFromContext(ctx)
		return nil, nil
	}
	md := metadata.Pairs(MDHeaderRequestID, "xyz-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := interceptor(ctx, nil, info, handler)
	require.NoError(t, err)
	assert.Equal(t, "xyz-123", got)
}
