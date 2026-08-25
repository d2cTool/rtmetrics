package agent

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/proto"
	"github.com/d2cTool/rtmetrics/internal/realip"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/d2cTool/rtmetrics/internal/grpcmetrics"
	"github.com/d2cTool/rtmetrics/internal/grpctls"
)

func TestGRPCClientSendsBatchAndRealIP(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	cert, certPEM, _, err := grpctls.Generate([]string{"127.0.0.1"})
	require.NoError(t, err)
	caFile := filepath.Join(t.TempDir(), "grpc.crt")
	require.NoError(t, os.WriteFile(caFile, certPEM, 0o600))

	var gotIP string
	st := storage.New()
	srv := grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if vals := md.Get(realip.Header); len(vals) > 0 {
					gotIP = vals[0]
				}
			}
			return handler(ctx, req)
		}),
	)
	proto.RegisterMetricsServer(srv, grpcmetrics.New(service.New(st), slog.Default(), nil))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	client, err := NewGRPCClient(lis.Addr().String(), caFile, slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	batch := []m.Metrics{*m.NewGauge("Alloc", 11), *m.NewCounter("PollCount", 4)}
	require.NoError(t, client.SendBatch(t.Context(), batch))

	gauge, err := st.GetGauge(t.Context(), "Alloc")
	require.NoError(t, err)
	assert.Equal(t, 11.0, gauge)
	require.NotEmpty(t, gotIP)
	require.NotNil(t, net.ParseIP(gotIP))
}
