package agent

import (
	"context"
	"fmt"
	"log/slog"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/grpctls"
	"github.com/d2cTool/rtmetrics/internal/proto"
	"github.com/d2cTool/rtmetrics/internal/realip"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCClient шлёт батчи через Metrics.UpdateMetrics и кладёт x-real-ip в metadata.
type GRPCClient struct {
	client proto.MetricsClient
	conn   *grpc.ClientConn
	logger *slog.Logger
}

// NewGRPCClient подключается к gRPC-серверу по addr (host:port) по TLS.
// caFile — PEM сертификата сервера (self-signed или CA); пустой путь шифрует без проверки имени.
func NewGRPCClient(addr, caFile string, logger *slog.Logger) (*GRPCClient, error) {
	creds, err := grpctls.ClientCredentials(caFile)
	if err != nil {
		return nil, fmt.Errorf("grpc tls: %w", err)
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(realIPClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial grpc %s: %w", addr, err)
	}
	return &GRPCClient{
		client: proto.NewMetricsClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

// Close закрывает соединение.
func (c *GRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// SendBatch формирует UpdateMetricsRequest и вызывает UpdateMetrics.
func (c *GRPCClient) SendBatch(ctx context.Context, metrics []m.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	req := proto.UpdateMetricsRequest_builder{Metrics: proto.FromModel(metrics)}.Build()
	_, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("grpc update metrics: %w", err)
	}
	c.logger.Debug("grpc metrics batch sent", slog.Int("count", len(metrics)))
	return nil
}

func realIPClientInterceptor() grpc.UnaryClientInterceptor {
	ip := realip.Host()
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, realip.Header, ip)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
