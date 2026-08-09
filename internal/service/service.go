package service

import (
	"context"

	"github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
)

type MetricsService interface {
	UpdateCounter(ctx context.Context, name string, value int64) (int64, error)
	UpdateGauge(ctx context.Context, name string, value float64) (float64, error)
	UpdateBatch(ctx context.Context, metrics []model.Metrics) error
	GetCounter(ctx context.Context, name string) (int64, error)
	GetGauge(ctx context.Context, name string) (float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
}

type Service struct {
	repo repository.MetricsRepository
}

// New создаёт сервис поверх переданного репозитория.
func New(repo repository.MetricsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	return s.repo.SaveCounter(ctx, name, value)
}

func (s *Service) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	return s.repo.SaveGauge(ctx, name, value)
}

func (s *Service) UpdateBatch(ctx context.Context, metrics []model.Metrics) error {
	return s.repo.SaveBatch(ctx, metrics)
}

func (s *Service) GetCounter(ctx context.Context, name string) (int64, error) {
	return s.repo.GetCounter(ctx, name)
}

func (s *Service) GetGauge(ctx context.Context, name string) (float64, error) {
	return s.repo.GetGauge(ctx, name)
}

func (s *Service) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetAllCounters(ctx)
}

func (s *Service) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.repo.GetAllGauges(ctx)
}

// Проверка, что *Service реализует MetricsService.
var _ MetricsService = (*Service)(nil)
