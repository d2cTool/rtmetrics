package agent

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkBatch(t *testing.T) {
	t.Parallel()

	batch := make([]m.Metrics, 10)

	tests := []struct {
		name       string
		parts      int
		wantChunks []int
	}{
		{name: "один воркер — один чанк", parts: 1, wantChunks: []int{10}},
		{name: "ровное деление", parts: 5, wantChunks: []int{2, 2, 2, 2, 2}},
		{name: "остаток уходит в последний чанк", parts: 3, wantChunks: []int{4, 4, 2}},
		{name: "воркеров больше, чем метрик", parts: 20, wantChunks: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{name: "невалидное число частей", parts: 0, wantChunks: []int{10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chunks := chunkBatch(batch, tt.parts)
			sizes := make([]int, 0, len(chunks))
			total := 0
			for _, chunk := range chunks {
				sizes = append(sizes, len(chunk))
				total += len(chunk)
			}

			assert.Equal(t, tt.wantChunks, sizes)
			assert.Equal(t, len(batch), total, "метрики не должны теряться")
		})
	}

	assert.Nil(t, chunkBatch(nil, 3))
}

func TestRunnerSnapshotIncludesRuntimeAndSystemMetrics(t *testing.T) {
	t.Parallel()

	r := NewRunner(nil, slog.Default(), time.Second, time.Second, 1)
	r.updateRuntimeMetrics()
	r.system = SystemMetrics{TotalMemory: 100, FreeMemory: 50, CPUUtilization: []float64{1, 2}}

	names := make(map[string]struct{})
	for _, metric := range r.snapshot() {
		names[metric.ID] = struct{}{}
	}

	for _, name := range []string{"Alloc", "PollCount", "TotalMemory", "FreeMemory", "CPUutilization1", "CPUutilization2"} {
		assert.Contains(t, names, name)
	}
}

func TestRunnerCollectsAndReportsInSeparateGoroutines(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	runner := NewRunner(NewClient(server.URL, "", slog.Default()), slog.Default(), 10*time.Millisecond, 30*time.Millisecond, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runner.Run(ctx)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "runner не завершился после отмены контекста")
	}

	assert.Positive(t, requests.Load(), "метрики должны уйти на сервер")

	runner.mu.RLock()
	defer runner.mu.RUnlock()
	assert.Positive(t, runner.counters.PollCount, "runtime-метрики должны опрашиваться")
	assert.Positive(t, runner.system.TotalMemory, "системные метрики должны опрашиваться")
}
