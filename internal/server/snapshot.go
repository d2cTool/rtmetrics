// Package server — восстановление и периодический снимок in-memory хранилища.
package server

import (
	"context"
	"log/slog"
	"time"

	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/storage"
)

// RestoreIfNeeded загружает снимок из файла, если включён Restore.
func RestoreIfNeeded(cfg *config.ServerConfig, st *storage.MemStorage, log *slog.Logger) {
	if !cfg.Restore || cfg.FileStoragePath == "" {
		return
	}
	counters, gauges, err := storage.Load(cfg.FileStoragePath)
	if err != nil {
		log.Warn("failed to restore from file", slog.String("error", err.Error()), slog.String("path", cfg.FileStoragePath))
		return
	}
	if len(counters) == 0 && len(gauges) == 0 {
		return
	}
	st.Restore(counters, gauges)
	log.Info("restored metrics from file",
		slog.String("path", cfg.FileStoragePath),
		slog.Int("counters", len(counters)),
		slog.Int("gauges", len(gauges)),
	)
}

// SaveSnapshot пишет текущие метрики в FileStoragePath.
func SaveSnapshot(cfg *config.ServerConfig, st *storage.MemStorage, log *slog.Logger) {
	if cfg.FileStoragePath == "" {
		return
	}
	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	gauges, _ := st.GetAllGauges(ctx)
	if err := storage.Save(ctx, cfg.FileStoragePath, counters, gauges); err != nil {
		log.Error("failed to save metrics to file", slog.String("error", err.Error()), slog.String("path", cfg.FileStoragePath))
	} else {
		log.Debug("metrics saved to file", slog.String("path", cfg.FileStoragePath))
	}
}

// RunPeriodicSave пишет снимок каждые StoreInterval секунд, пока ctx не отменён.
func RunPeriodicSave(ctx context.Context, cfg *config.ServerConfig, st *storage.MemStorage, log *slog.Logger) {
	if cfg.StoreInterval <= 0 || cfg.FileStoragePath == "" {
		return
	}
	ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			SaveSnapshot(cfg, st, log)
		}
	}
}
