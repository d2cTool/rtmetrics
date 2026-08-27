package grpcmetrics

import (
	"context"
	"log/slog"
	"net"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/proto"
	"github.com/d2cTool/rtmetrics/internal/realip"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/d2cTool/rtmetrics/internal/middleware/subnet"
)

func startTestGRPC(t *testing.T, network *net.IPNet) proto.MetricsClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.UnaryInterceptor(subnet.UnaryInterceptor(slog.Default(), network)))
	st := storage.New()
	proto.RegisterMetricsServer(srv, New(service.New(st), slog.Default(), nil))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	t.Cleanup(func() {
		_ = st
	})
	return proto.NewMetricsClient(conn)
}

func TestUpdateMetricsSavesBatch(t *testing.T) {
	st := storage.New()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	proto.RegisterMetricsServer(srv, New(service.New(st), slog.Default(), nil))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client := proto.NewMetricsClient(conn)
	_, err = client.UpdateMetrics(t.Context(), proto.UpdateMetricsRequest_builder{
		Metrics: []*proto.Metric{
			proto.Metric_builder{Id: "Alloc", Type: proto.Metric_GAUGE, Value: 42}.Build(),
			proto.Metric_builder{Id: "PollCount", Type: proto.Metric_COUNTER, Delta: 3}.Build(),
		},
	}.Build())
	require.NoError(t, err)

	gauge, err := st.GetGauge(t.Context(), "Alloc")
	require.NoError(t, err)
	assert.Equal(t, 42.0, gauge)
	counter, err := st.GetCounter(t.Context(), "PollCount")
	require.NoError(t, err)
	assert.Equal(t, int64(3), counter)
}

func TestUpdateMetricsDeniedOutsideSubnet(t *testing.T) {
	_, network, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)
	client := startTestGRPC(t, network)

	ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs(realip.Header, "8.8.8.8"))
	_, err = client.UpdateMetrics(ctx, proto.UpdateMetricsRequest_builder{
		Metrics: []*proto.Metric{
			proto.Metric_builder{Id: "Alloc", Type: proto.Metric_GAUGE, Value: 1}.Build(),
		},
	}.Build())
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGRPCClientSendsBatchWithRealIP(t *testing.T) {
	st := storage.New()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	proto.RegisterMetricsServer(srv, New(service.New(st), slog.Default(), nil))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// проверяем конвертацию тем же путём, что агент
	delta := int64(2)
	value := 9.5
	req := proto.UpdateMetricsRequest_builder{Metrics: proto.FromModel([]model.Metrics{
		{ID: "PollCount", MType: model.Counter, Delta: &delta},
		{ID: "Alloc", MType: model.Gauge, Value: &value},
	})}.Build()
	_, err = proto.NewMetricsClient(conn).UpdateMetrics(t.Context(), req)
	require.NoError(t, err)
	got, err := st.GetGauge(t.Context(), "Alloc")
	require.NoError(t, err)
	assert.Equal(t, 9.5, got)
}
