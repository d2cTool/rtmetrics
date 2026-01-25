package post

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestNew_SaveCounter_Success(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/counter/testCounter/10", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Body.String())
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, int64(10), repo.counters["testCounter"])
}

func TestNew_SaveCounter_Accumulation(t *testing.T) {
	repo := newMockRepository()
	repo.counters["testCounter"] = 5
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/counter/testCounter/10", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "15", w.Body.String())
}

func TestNew_SaveGauge_Success(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/gauge/testGauge/3.14", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "3.14", w.Body.String())
	assert.InDelta(t, 3.14, repo.gauges["testGauge"], 0.001)
}

func TestNew_SaveGauge_Overwrite(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["testGauge"] = 1.0
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/gauge/testGauge/2.5", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2.5", w.Body.String())
	assert.InDelta(t, 2.5, repo.gauges["testGauge"], 0.001)
}

func TestNew_InvalidMetricType(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/invalid/testName/10", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid metric type. Must be 'gauge' or 'counter'\n", w.Body.String())
}

func TestNew_InvalidCounterValue(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/counter/testCounter/invalid", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid value parameter\n", w.Body.String())
}

func TestNew_InvalidGaugeValue(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/gauge/testGauge/invalid", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid gauge parameter\n", w.Body.String())
}

func TestNew_CounterRepositoryError(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("repository error")
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/counter/testCounter/10", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to save counter\n", w.Body.String())
}

func TestNew_GaugeRepositoryError(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("repository error")
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/gauge/testGauge/3.14", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to save gauge\n", w.Body.String())
}

func TestNew_CounterNegativeValue(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/counter/testCounter/-5", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "-5", w.Body.String())
}

func TestNew_GaugeNegativeValue(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodPost, "/gauge/testGauge/-3.14", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/{mtype}/{name}/{value}", handler)

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "-3.14", w.Body.String())
}
