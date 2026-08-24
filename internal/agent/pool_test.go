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

	pool, err := NewWorkerPool(NewClient(server.URL, "", slog.Default()), slog.Default(), workers)
	require.NoError(t, err)

	ctx := context.Background()
	pool.Start()

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

	pool, err := NewWorkerPool(NewClient(server.URL, "", slog.Default()), slog.Default(), 1)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, pool.Submit(ctx, []m.Metrics{*m.NewGauge("Alloc", 1)}))
	assert.True(t, pool.Submit(ctx, nil), "пустое задание не должно ставиться в очередь")
}

func TestWorkerPoolDrainsQueuedJobsAfterStop(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	var total int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		total++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pool, err := NewWorkerPool(NewClient(server.URL, "", slog.Default()), slog.Default(), 1)
	require.NoError(t, err)

	pool.Start()
	batch := []m.Metrics{*m.NewGauge("Alloc", 1)}
	require.True(t, pool.Submit(context.Background(), batch))
	<-started
	require.True(t, pool.Submit(context.Background(), batch))

	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.Stop()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "пул не дождался отправки очереди")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, total, "задания в очереди должны уйти после Stop")
}

func TestNewWorkerPoolRejectsInvalidWorkers(t *testing.T) {
	t.Parallel()

	for _, workers := range []int{0, -5} {
		pool, err := NewWorkerPool(nil, slog.Default(), workers)
		require.Error(t, err, "workers=%d", workers)
		assert.Nil(t, pool)
	}

	pool, err := NewWorkerPool(nil, slog.Default(), 4)
	require.NoError(t, err)
	assert.Equal(t, 4, pool.Workers())
}
