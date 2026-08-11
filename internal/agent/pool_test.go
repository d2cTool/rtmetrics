package agent

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPoolLimitsConcurrentRequests(t *testing.T) {
	const (
		workers = 3
		jobs    = 20
	)

	var (
		mu      sync.Mutex
		current int
		peak    int
		total   int
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		current++
		total++
		peak = max(peak, current)
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pool := NewWorkerPool(NewClient(server.URL, "", slog.Default()), slog.Default(), workers)

	ctx := context.Background()
	pool.Start(ctx)

	batch := []m.Metrics{*m.NewGauge("Alloc", 1)}
	for range jobs {
		require.True(t, pool.Submit(ctx, batch))
	}
	pool.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, jobs, total, "должны уйти все задания")
	assert.LessOrEqual(t, peak, workers, "одновременных запросов больше лимита: %d", peak)
}

func TestWorkerPoolSubmitStopsOnCancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pool := NewWorkerPool(NewClient(server.URL, "", slog.Default()), slog.Default(), 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, pool.Submit(ctx, []m.Metrics{*m.NewGauge("Alloc", 1)}))
	assert.True(t, pool.Submit(ctx, nil), "пустое задание не должно ставиться в очередь")
}

func TestNewWorkerPoolNormalizesWorkers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, NewWorkerPool(nil, slog.Default(), 0).Workers())
	assert.Equal(t, 1, NewWorkerPool(nil, slog.Default(), -5).Workers())
	assert.Equal(t, 4, NewWorkerPool(nil, slog.Default(), 4).Workers())
}
