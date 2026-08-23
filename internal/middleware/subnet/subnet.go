// Package subnet отклоняет запросы с X-Real-IP вне доверенной подсети.
package subnet

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/d2cTool/rtmetrics/internal/realip"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// New возвращает middleware проверки X-Real-IP. Подключать только при network != nil.
func New(log *slog.Logger, network *net.IPNet) func(next http.Handler) http.Handler {
	log = log.With(slog.String("component", "middleware/subnet"))
	log.Info("trusted subnet check enabled", slog.String("cidr", network.String()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get(realip.Header)
			if !realip.Allowed(network, raw) {
				log.Debug("real ip is not in trusted subnet",
					slog.String("ip", raw),
					slog.String("cidr", network.String()),
					slog.String("uri", r.RequestURI),
				)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UnaryInterceptor отклоняет gRPC-вызовы, если x-real-ip не входит в network.
// Пустой network — без ограничений.
func UnaryInterceptor(log *slog.Logger, network *net.IPNet) grpc.UnaryServerInterceptor {
	log = log.With(slog.String("component", "grpc/subnet"))
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if network == nil {
			return handler(ctx, req)
		}
		ip := metadataValue(ctx, realip.Header)
		if !realip.Allowed(network, ip) {
			log.Debug("real ip is not in trusted subnet",
				slog.String("ip", ip),
				slog.String("cidr", network.String()),
				slog.String("method", info.FullMethod),
			)
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
		return handler(ctx, req)
	}
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
