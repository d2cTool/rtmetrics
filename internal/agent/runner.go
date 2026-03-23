package agent

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Runner struct {
	client         *Client
	logger         *slog.Logger
	pollInterval   time.Duration
	reportInterval time.Duration

	mu       sync.RWMutex
	gauges   GaugeMetrics
	counters CountMetrics
}

func NewRunner(client *Client, logger *slog.Logger, pollInterval, reportInterval time.Duration) *Runner {
	return &Runner{
		client:         client,
		logger:         logger,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

func (r *Runner) Start(ctx context.Context) {
	// Инициализируем метрики при старте
	r.updateMetrics()

	// Горутина для периодического сбора метрик
	go r.pollMetrics(ctx)

	// Горутина для периодической отправки метрик
	go r.reportMetrics(ctx)
}

func (r *Runner) pollMetrics(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	r.logger.Info("started polling metrics", slog.Duration("interval", r.pollInterval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopped polling metrics")
			return
		case <-ticker.C:
			r.updateMetrics()
			r.logger.Debug("metrics collected")
		}
	}
}

func (r *Runner) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(r.reportInterval)
	defer ticker.Stop()

	r.logger.Info("started reporting metrics", slog.Duration("interval", r.reportInterval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopped reporting metrics")
			return
		case <-ticker.C:
			r.sendCurrentMetrics()
		}
	}
}

func (r *Runner) updateMetrics() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges = Collect()
	r.counters.PollCount++
}

func (r *Runner) sendCurrentMetrics() {
	r.mu.RLock()
	gaugeMetrics := r.gauges
	counterMetrics := r.counters
	r.mu.RUnlock()

	if err := r.client.SendGaugeMetrics(gaugeMetrics); err != nil {
		r.logger.Error("failed to send gauge metrics", slog.String("error", err.Error()))
	} else {
		r.logger.Debug("gauge metrics sent successfully")
	}

	if err := r.client.SendCounterMetrics(counterMetrics); err != nil {
		r.logger.Error("failed to send counter metrics", slog.String("error", err.Error()))
	} else {
		r.logger.Debug("counter metrics sent successfully")
	}
}
