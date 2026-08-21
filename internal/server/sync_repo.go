package server

import (
	"context"
	"log/slog"

	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/d2cTool/rtmetrics/internal/storage"
)

// SyncSaveRepo оборачивает MemStorage и сразу сбрасывает снимок на диск.
type SyncSaveRepo struct {
	repo *storage.MemStorage
	cfg  *config.ServerConfig
	log  *slog.Logger
}

// NewSyncSaveRepo включает синхронную персистенцию при StoreInterval == 0.
func NewSyncSaveRepo(repo *storage.MemStorage, cfg *config.ServerConfig, log *slog.Logger) *SyncSaveRepo {
	return &SyncSaveRepo{repo: repo, cfg: cfg, log: log}
}

func (s *SyncSaveRepo) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	v, err := s.repo.SaveCounter(ctx, name, value)
	if err != nil {
		return v, err
	}
	if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
		SaveSnapshot(s.cfg, s.repo, s.log)
	}
	return v, nil
}

func (s *SyncSaveRepo) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	v, err := s.repo.SaveGauge(ctx, name, value)
	if err != nil {
		return v, err
	}
	if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
		SaveSnapshot(s.cfg, s.repo, s.log)
	}
	return v, nil
}

func (s *SyncSaveRepo) SaveBatch(ctx context.Context, metrics []model.Metrics) error {
	if err := s.repo.SaveBatch(ctx, metrics); err != nil {
		return err
	}
	if s.cfg.StoreInterval == 0 && s.cfg.FileStoragePath != "" {
		SaveSnapshot(s.cfg, s.repo, s.log)
	}
	return nil
}

func (s *SyncSaveRepo) GetCounter(ctx context.Context, name string) (int64, error) {
	return s.repo.GetCounter(ctx, name)
}

func (s *SyncSaveRepo) GetGauge(ctx context.Context, name string) (float64, error) {
	return s.repo.GetGauge(ctx, name)
}

func (s *SyncSaveRepo) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetAllCounters(ctx)
}

func (s *SyncSaveRepo) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.repo.GetAllGauges(ctx)
}

// Проверка, что *SyncSaveRepo реализует repository.MetricsRepository.
var _ repository.MetricsRepository = (*SyncSaveRepo)(nil)
