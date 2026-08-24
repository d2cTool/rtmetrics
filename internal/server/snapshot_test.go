package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestRunPeriodicSaveStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.ServerConfig{StoreInterval: 60, FileStoragePath: t.TempDir() + "/metrics.json"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		RunPeriodicSave(ctx, cfg, storage.New(), slog.Default())
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		require.Fail(t, "периодическое сохранение не остановилось")
	}
}
