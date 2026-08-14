package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	m "github.com/d2cTool/rtmetrics/internal/model"
)

type WorkerPool struct {
	client  *Client
	logger  *slog.Logger
	workers int
	jobs    chan []m.Metrics
	wg      sync.WaitGroup
	once    sync.Once
}

func NewWorkerPool(client *Client, logger *slog.Logger, workers int) (*WorkerPool, error) {
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

func (p *WorkerPool) Workers() int {
	return p.workers
}

func (p *WorkerPool) Start(ctx context.Context) {
	p.wg.Add(p.workers)
	for i := range p.workers {
		go p.work(ctx, i+1)
	}

	p.logger.Info("worker pool started", slog.Int("workers", p.workers))
}

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

func (p *WorkerPool) Stop() {
	p.once.Do(func() {
		close(p.jobs)
	})
	p.wg.Wait()
	p.logger.Info("worker pool stopped")
}

func (p *WorkerPool) work(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-p.jobs:
			if !ok {
				return
			}

			if err := p.client.SendBatch(ctx, batch); err != nil {
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
}
