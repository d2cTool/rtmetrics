// Package grpcmetrics реализует gRPC-сервис Metrics.
package grpcmetrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/d2cTool/rtmetrics/internal/audit"
	"github.com/d2cTool/rtmetrics/internal/proto"
	"github.com/d2cTool/rtmetrics/internal/realip"
	"github.com/d2cTool/rtmetrics/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server принимает UpdateMetrics и пишет метрики через service.
type Server struct {
	proto.UnimplementedMetricsServer
	svc     service.MetricsService
	log     *slog.Logger
	auditor *audit.Subject
}

// New собирает gRPC-сервис над svc.
func New(svc service.MetricsService, log *slog.Logger, auditor *audit.Subject) *Server {
	return &Server{svc: svc, log: log, auditor: auditor}
}

// UpdateMetrics сохраняет пачку метрик.
func (s *Server) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	if req == nil {
		return &proto.UpdateMetricsResponse{}, nil
	}

	batch := proto.ToModel(req.GetMetrics())
	if len(batch) == 0 {
		return &proto.UpdateMetricsResponse{}, nil
	}

	if err := s.svc.UpdateBatch(ctx, batch); err != nil {
		s.log.Error("grpc update metrics failed", slog.String("error", err.Error()), slog.Int("count", len(batch)))
		return nil, status.Error(codes.Internal, "failed to save metrics")
	}

	if s.auditor != nil {
		names := make([]string, 0, len(batch))
		for _, m := range batch {
			names = append(names, m.ID)
		}
		s.auditor.Notify(audit.Event{TS: time.Now().Unix(), Metrics: names, IPAddress: metadataIP(ctx)})
	}

	s.log.Debug("grpc metrics batch saved", slog.Int("count", len(batch)))
	return &proto.UpdateMetricsResponse{}, nil
}

func metadataIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(realip.Header)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
