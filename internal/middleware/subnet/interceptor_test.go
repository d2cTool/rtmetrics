package subnet

import (
	"context"
	"log/slog"
	"net"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/realip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryInterceptorEmptySubnetAllows(t *testing.T) {
	interceptor := UnaryInterceptor(slog.Default(), nil)
	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/metrics.Metrics/UpdateMetrics"}, func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestUnaryInterceptorDeniedAndAllowed(t *testing.T) {
	_, network, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	interceptor := UnaryInterceptor(slog.Default(), network)

	denied := []context.Context{
		context.Background(),
		metadata.NewIncomingContext(context.Background(), metadata.Pairs(realip.Header, "192.168.1.1")),
	}
	for _, ctx := range denied {
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/m"}, func(ctx context.Context, req any) (any, error) {
			return nil, nil
		})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(realip.Header, "10.1.2.3"))
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/m"}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}
