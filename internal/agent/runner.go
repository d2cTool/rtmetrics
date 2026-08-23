package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	m "github.com/d2cTool/rtmetrics/internal/model"
)

// Runner крутит опрос runtime/system и отправку батчей через WorkerPool.
type Runner struct {
	pool           *WorkerPool
	logger         *slog.Logger
	pollInterval   time.Duration
	reportInterval time.Duration

	mu       sync.Mutex
	gauges   GaugeMetrics
	system   SystemMetrics
	counters CountMetrics
}

// NewRunner проверяет интервалы и rateLimit (>= 1) и собирает Runner.
func NewRunner(client BatchSender, logger *slog.Logger, pollInterval, reportInterval time.Duration, rateLimit int) (*Runner, error) {
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be > 0, got %s", pollInterval)
	}
	if reportInterval <= 0 {
		return nil, fmt.Errorf("report interval must be > 0, got %s", reportInterval)
	}

	pool, err := NewWorkerPool(client, logger, rateLimit)
	if err != nil {
		return nil, err
	}

	return &Runner{
		pool:           pool,
		logger:         logger,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}, nil
}

// Run блокируется до отмены ctx: опрос, репорт, финальный сброс метрик, остановка пула.
func (r *Runner) Run(ctx context.Context) {
	r.pool.Start()

	var wg sync.WaitGroup
	for _, loop := range []func(context.Context){r.pollRuntime, r.pollSystem, r.report} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loop(ctx)
		}()
	}

	wg.Wait()
	r.flush()
	r.pool.Stop()
}

func (r *Runner) pollRuntime(ctx context.Context) {
	r.updateRuntimeMetrics()

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	r.logger.Info("started polling runtime metrics", slog.Duration("interval", r.pollInterval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopped polling runtime metrics")
			return
		case <-ticker.C:
			r.updateRuntimeMetrics()
			r.logger.Debug("runtime metrics collected")
		}
	}
}

func (r *Runner) pollSystem(ctx context.Context) {
	r.updateSystemMetrics(ctx)

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	r.logger.Info("started polling system metrics", slog.Duration("interval", r.pollInterval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopped polling system metrics")
			return
		case <-ticker.C:
			r.updateSystemMetrics(ctx)
			r.logger.Debug("system metrics collected")
		}
	}
}

func (r *Runner) report(ctx context.Context) {
	ticker := time.NewTicker(r.reportInterval)
	defer ticker.Stop()

	r.logger.Info("started reporting metrics", slog.Duration("interval", r.reportInterval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopped reporting metrics")
			return
		case <-ticker.C:
			r.flush()
		}
	}
}

func (r *Runner) updateRuntimeMetrics() {
	gauges := Collect()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges = gauges
	r.counters.PollCount++
}

func (r *Runner) updateSystemMetrics(ctx context.Context) {
	system, err := CollectSystem(ctx)
	if err != nil {
		if ctx.Err() == nil {
			r.logger.Error("failed to collect system metrics", slog.String("error", err.Error()))
		}
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.system = system
}

func (r *Runner) flush() {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	r.submitCurrentMetrics(ctx)
}

func (r *Runner) submitCurrentMetrics(ctx context.Context) {
	batch, polls := r.takeBatch()
	if len(batch) == 0 {
		return
	}

	for _, chunk := range chunkBatch(batch, r.pool.Workers()) {
		if !r.pool.Submit(ctx, chunk) {
			r.addPollCount(polls)
			return
		}
	}
}

func (r *Runner) snapshot() []m.Metrics {
	batch, _ := r.copyBatch()
	return batch
}

func (r *Runner) takeBatch() ([]m.Metrics, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	batch := append(BuildBatch(r.gauges, r.counters), r.system.Gauges()...)
	polls := r.counters.PollCount
	r.counters.PollCount = 0
	return batch, polls
}

func (r *Runner) copyBatch() ([]m.Metrics, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append(BuildBatch(r.gauges, r.counters), r.system.Gauges()...), r.counters.PollCount
}

func (r *Runner) addPollCount(n int64) {
	if n == 0 {
		return
	}
	r.mu.Lock()
	r.counters.PollCount += n
	r.mu.Unlock()
}

func chunkBatch(batch []m.Metrics, parts int) [][]m.Metrics {
	if len(batch) == 0 {
		return nil
	}
	if parts < 1 {
		parts = 1
	}

	size := (len(batch) + parts - 1) / parts
	chunks := make([][]m.Metrics, 0, parts)
	for start := 0; start < len(batch); start += size {
		chunks = append(chunks, batch[start:min(start+size, len(batch))])
	}

	return chunks
}
