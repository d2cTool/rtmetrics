package get

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository - мок для тестирования
type mockRepository struct {
	counters map[string]int64
	gauges   map[string]float64
	err      error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *mockRepository) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.counters == nil {
		m.counters = make(map[string]int64)
	}
	m.counters[name] += value
	return m.counters[name], nil
}

func (m *mockRepository) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.gauges == nil {
		m.gauges = make(map[string]float64)
	}
	m.gauges[name] = value
	return value, nil
}

func (m *mockRepository) GetCounter(ctx context.Context, name string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	value, ok := m.counters[name]
	if !ok {
		return 0, errors.New("counter not found")
	}
	return value, nil
}

func (m *mockRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	value, ok := m.gauges[name]
	if !ok {
		return 0, errors.New("gauge not found")
	}
	return value, nil
}

func (m *mockRepository) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.counters, nil
}

func (m *mockRepository) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.gauges, nil
}

func (m *mockRepository) SaveBatch(ctx context.Context, batch []metrics.Metrics) error {
	if m.err != nil {
		return m.err
	}
	for _, mt := range batch {
		switch mt.MType {
		case metrics.Counter:
			if mt.Delta != nil {
				m.counters[mt.ID] += *mt.Delta
			}
		case metrics.Gauge:
			if mt.Value != nil {
				m.gauges[mt.ID] = *mt.Value
			}
		}
	}
	return nil
}

func TestNew_GetCounter_Success(t *testing.T) {
	repo := newMockRepository()
	repo.counters["testCounter"] = 42

	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/counter/testCounter", nil)
	w := httptest.NewRecorder()

	// Создаем роутер для тестирования с параметрами URL
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "42", w.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestNew_GetGauge_Success(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["testGauge"] = 3.14

	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/testGauge", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "3.14", w.Body.String())
}

func TestNew_InvalidMetricType(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/invalid/testName", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Invalid metric type. Must be 'gauge' or 'counter'\n", w.Body.String())
}

func TestNew_CounterNotFound(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/counter/nonexistent", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Counter not found\n", w.Body.String())
}

func TestNew_GaugeNotFound(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/nonexistent", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Gauge not found\n", w.Body.String())
}

func TestNew_CounterZeroValue(t *testing.T) {
	repo := newMockRepository()
	repo.counters["zeroCounter"] = 0

	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/counter/zeroCounter", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0", w.Body.String())
}

func TestNew_GaugeZeroValue(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["zeroGauge"] = 0.0

	log := slog.Default()
	handler := New(log, service.New(repo))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/zeroGauge", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/value/{mtype}/{name}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0", w.Body.String())
}
