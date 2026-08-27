package agent

import (
	"context"

	m "github.com/d2cTool/rtmetrics/internal/model"
)

// BatchSender отправляет пачку метрик на сервер (HTTP или gRPC).
type BatchSender interface {
	SendBatch(ctx context.Context, metrics []m.Metrics) error
}
