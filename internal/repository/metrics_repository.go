// Package repository задаёт контракт хранилища метрик.
package repository

import (
	"context"

	"github.com/d2cTool/rtmetrics/internal/model"
)

// MetricsRepository сохраняет и читает counter/gauge. Реализации — память и PostgreSQL.
type MetricsRepository interface {
	SaveCounter(ctx context.Context, name string, value int64) (int64, error)
	SaveGauge(ctx context.Context, name string, value float64) (float64, error)

	GetCounter(ctx context.Context, name string) (int64, error)
	GetGauge(ctx context.Context, name string) (float64, error)

	GetAllCounters(ctx context.Context) (map[string]int64, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)

	SaveBatch(ctx context.Context, metrics []model.Metrics) error
}
