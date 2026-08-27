package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	m "github.com/d2cTool/rtmetrics/internal/model"
)

// sendTimeout даёт воркеру время дослать батч при остановке, включая ретраи 1s+3s+5s.
const sendTimeout = 15 * time.Second

// WorkerPool ограничивает число одновременных исходящих запросов агента.
type WorkerPool struct {
	client  BatchSender
	logger  *slog.Logger
	workers int
	jobs    chan []m.Metrics
	wg      sync.WaitGroup
	once    sync.Once
}

// NewWorkerPool создаёт пул. workers должен быть >= 1.
func NewWorkerPool(client BatchSender, logger *slog.Logger, workers int) (*WorkerPool, error) {
	if workers < 1 {
		return nil, fmt.Errorf("rate limit must be >= 1, got %d", workers)
	}

	return &WorkerPool{
		client:  client,
		logger:  logger,
		workers: workers,
		jobs:    make(chan []m.Metrics, workers),
	}, nil
}

// Workers возвращает размер пула.
func (p *WorkerPool) Workers() int {
	return p.workers
}

// Start запускает воркеры. Они живут до Stop, чтобы дослать уже принятые задания.
func (p *WorkerPool) Start() {
	p.wg.Add(p.workers)
	for i := range p.workers {
		go p.work(i + 1)
	}

	p.logger.Info("worker pool started", slog.Int("workers", p.workers))
}

// Submit ставит батч в очередь. false, если ctx уже отменён.
func (p *WorkerPool) Submit(ctx context.Context, batch []m.Metrics) bool {
	if len(batch) == 0 {
		return true
	}

	select {
	case <-ctx.Done():
		return false
	default:
	}

	select {
	case <-ctx.Done():
		return false
	case p.jobs <- batch:
		return true
	}
}

// Stop закрывает очередь и ждёт воркеров. Повторный вызов безопасен.
func (p *WorkerPool) Stop() {
	p.once.Do(func() {
		close(p.jobs)
	})
	p.wg.Wait()
	p.logger.Info("worker pool stopped")
}

func (p *WorkerPool) work(id int) {
	defer p.wg.Done()

	for batch := range p.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		err := p.client.SendBatch(ctx, batch)
		cancel()
		if err != nil {
			p.logger.Error("failed to send metrics batch",
				slog.Int("worker", id),
				slog.Int("count", len(batch)),
				slog.String("error", err.Error()))
			continue
		}

		p.logger.Debug("metrics batch sent",
			slog.Int("worker", id),
			slog.Int("count", len(batch)))
	}
}
