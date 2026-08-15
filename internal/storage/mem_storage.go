// Package storage — in-memory и файловое хранилище метрик.
package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/d2cTool/rtmetrics/internal/model"
)

var (
	// ErrCounterNotFound возвращается, если counter с таким именем нет.
	ErrCounterNotFound = errors.New("counter not found")
	// ErrGaugeNotFound возвращается, если gauge с таким именем нет.
	ErrGaugeNotFound = errors.New("gauge not found")
)

// MemStorage хранит метрики в памяти. Безопасен для одновременного доступа.
type MemStorage struct {
	mu sync.RWMutex
	// Counters — накопительные метрики по имени.
	Counters map[string]int64
	// Gauges — мгновенные значения по имени.
	Gauges map[string]float64
}

// New создаёт пустое in-memory хранилище.
func New() *MemStorage {
	return &MemStorage{
		Counters: make(map[string]int64),
		Gauges:   make(map[string]float64),
	}
}

// SaveCounter прибавляет value к counter name и возвращает новое значение.
func (s *MemStorage) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Counters[name] += value
	return s.Counters[name], nil
}

// SaveGauge записывает gauge name. Предыдущее значение заменяется.
func (s *MemStorage) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Gauges[name] = value
	return value, nil
}

// SaveBatch применяет пачку метрик атомарно относительно мьютекса хранилища.
func (s *MemStorage) SaveBatch(ctx context.Context, metrics []model.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case model.Counter:
			if m.Delta != nil {
				s.Counters[m.ID] += *m.Delta
			}
		case model.Gauge:
			if m.Value != nil {
				s.Gauges[m.ID] = *m.Value
			}
		}
	}
	return nil
}

// GetCounter возвращает counter или ErrCounterNotFound.
func (s *MemStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, e := s.Counters[name]
	if e {
		return v, nil
	}

	return 0, fmt.Errorf("%w: %s", ErrCounterNotFound, name)
}

// GetGauge возвращает gauge или ErrGaugeNotFound.
func (s *MemStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, e := s.Gauges[name]
	if e {
		return v, nil
	}

	return 0, fmt.Errorf("%w: %s", ErrGaugeNotFound, name)
}

// GetAllCounters копирует все counter. Вызывающий может менять карту.
func (s *MemStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = v
	}
	return result, nil
}

// GetAllGauges копирует все gauge. Вызывающий может менять карту.
func (s *MemStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = v
	}
	return result, nil
}

// Restore заменяет содержимое хранилища копиями переданных карт.
func (s *MemStorage) Restore(counters map[string]int64, gauges map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if counters != nil {
		s.Counters = make(map[string]int64, len(counters))
		for k, v := range counters {
			s.Counters[k] = v
		}
	}
	if gauges != nil {
		s.Gauges = make(map[string]float64, len(gauges))
		for k, v := range gauges {
			s.Gauges[k] = v
		}
	}
}
