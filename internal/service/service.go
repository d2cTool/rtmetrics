// Package service содержит прикладную логику учёта метрик над хранилищем.
package service

import (
	"context"

	"github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
)

// MetricsService — операции, которые вызывают HTTP-хендлеры.
type MetricsService interface {
	UpdateCounter(ctx context.Context, name string, value int64) (int64, error)
	UpdateGauge(ctx context.Context, name string, value float64) (float64, error)
	UpdateBatch(ctx context.Context, metrics []model.Metrics) error
	GetCounter(ctx context.Context, name string) (int64, error)
	GetGauge(ctx context.Context, name string) (float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
}

// Service реализует MetricsService через MetricsRepository.
type Service struct {
	repo repository.MetricsRepository
}

// New возвращает сервис над repo.
func New(repo repository.MetricsRepository) *Service {
	return &Service{repo: repo}
}

// UpdateCounter прибавляет value к counter и возвращает актуальное значение.
func (s *Service) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	return s.repo.SaveCounter(ctx, name, value)
}

// UpdateGauge записывает gauge и возвращает записанное значение.
func (s *Service) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	return s.repo.SaveGauge(ctx, name, value)
}

// UpdateBatch сохраняет пачку метрик.
func (s *Service) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	return s.repo.SaveBatch(ctx, metrics)
}

// GetCounter возвращает counter по имени.
func (s *Service) GetCounter(ctx context.Context, name string) (int64, error) {
	return s.repo.GetCounter(ctx, name)
}

// GetGauge возвращает gauge по имени.
func (s *Service) GetGauge(ctx context.Context, name string) (float64, error) {
	return s.repo.GetGauge(ctx, name)
}

// GetAllCounters возвращает копию всех counter.
func (s *Service) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetAllCounters(ctx)
}

// GetAllGauges возвращает копию всех gauge.
func (s *Service) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.repo.GetAllGauges(ctx)
}

// Проверка, что *Service реализует MetricsService.
var _ MetricsService = (*Service)(nil)
