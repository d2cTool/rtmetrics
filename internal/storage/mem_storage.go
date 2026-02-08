package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrCounterNotFound = errors.New("counter not found")
	ErrGaugeNotFound   = errors.New("gauge not found")
)

type MemStorage struct {
	mu       sync.RWMutex
	Counters map[string]int64
	Gauges   map[string]float64
}

func New() *MemStorage {
	return &MemStorage{
		Counters: make(map[string]int64),
		Gauges:   make(map[string]float64),
	}
}

func (s *MemStorage) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, e := s.Counters[name]
	if e {
		s.Counters[name] = v + value
	} else {
		s.Counters[name] = value
	}

	return s.Counters[name], nil
}

func (s *MemStorage) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Gauges[name] = value
	return value, nil
}

func (s *MemStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, e := s.Counters[name]
	if e {
		return v, nil
	}

	return 0, fmt.Errorf("%w: %s", ErrCounterNotFound, name)
}

func (s *MemStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, e := s.Gauges[name]
	if e {
		return v, nil
	}

	return 0, fmt.Errorf("%w: %s", ErrGaugeNotFound, name)
}

func (s *MemStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64, len(s.Counters))
	for k, v := range s.Counters {
		result[k] = v
	}
	return result, nil
}

func (s *MemStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64, len(s.Gauges))
	for k, v := range s.Gauges {
		result[k] = v
	}
	return result, nil
}

// Restore заменяет текущее состояние хранилища загруженными данными.
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
