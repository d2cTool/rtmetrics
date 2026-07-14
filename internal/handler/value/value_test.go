package value

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	v, ok := m.counters[name]
	if !ok {
		return 0, errors.New("counter not found")
	}
	return v, nil
}

func (m *mockRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	v, ok := m.gauges[name]
	if !ok {
		return 0, errors.New("gauge not found")
	}
	return v, nil
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

func TestNew_GetCounter_Success(t *testing.T) {
	repo := newMockRepository()
	repo.counters["testCounter"] = 42
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"testCounter","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var resp metrics.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "testCounter", resp.ID)
	assert.Equal(t, "counter", resp.MType)
	require.NotNil(t, resp.Delta)
	assert.Equal(t, int64(42), *resp.Delta)
}

func TestNew_GetGauge_Success(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["testGauge"] = 3.14
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"testGauge","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var resp metrics.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "testGauge", resp.ID)
	assert.Equal(t, "gauge", resp.MType)
	require.NotNil(t, resp.Value)
	assert.InDelta(t, 3.14, *resp.Value, 0.001)
}

func TestNew_MissingContentType_UnsupportedMediaType(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"x","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	assert.Contains(t, w.Body.String(), "Content-Type must be application/json")
}

func TestNew_InvalidContentType_UnsupportedMediaType(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"x","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	assert.Contains(t, w.Body.String(), "Content-Type must be application/json")
}

func TestNew_InvalidJSON_BadRequest(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNew_InvalidMetricType_BadRequest(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"x","type":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid metric type")
}

func TestNew_CounterNotFound(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"nonexistent","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Counter not found")
}

func TestNew_GaugeNotFound(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"nonexistent","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Gauge not found")
}

func TestNew_ContentTypeWithCharset_Accepted(t *testing.T) {
	repo := newMockRepository()
	repo.counters["c"] = 1
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"c","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp metrics.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Delta)
	assert.Equal(t, int64(1), *resp.Delta)
}

func TestNew_GetCounter_ZeroValue(t *testing.T) {
	repo := newMockRepository()
	repo.counters["zeroCounter"] = 0
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"zeroCounter","type":"counter"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp metrics.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Delta)
	assert.Equal(t, int64(0), *resp.Delta)
}

func TestNew_GetGauge_ZeroValue(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["zeroGauge"] = 0.0
	log := slog.Default()
	handler := New(log, service.New(repo))

	body := `{"id":"zeroGauge","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/value", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp metrics.Metrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Value)
	assert.Equal(t, 0.0, *resp.Value)
}
