package updates

import (
	"context"
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
	m.counters[name] += value
	return m.counters[name], nil
}

func (m *mockRepository) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	m.gauges[name] = value
	return value, nil
}

func (m *mockRepository) GetCounter(ctx context.Context, name string) (int64, error) {
	return m.counters[name], nil
}

func (m *mockRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	return m.gauges[name], nil
}

func (m *mockRepository) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return m.counters, nil
}

func (m *mockRepository) GetAllGauges(ctx context.Context) (map[string]float64, error) {
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

func doRequest(t *testing.T, repo *mockRepository, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := New(slog.Default(), service.New(repo))
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/updates/", handler)
	r.ServeHTTP(w, req)

	return w
}

func TestNew_Batch_Success(t *testing.T) {
	repo := newMockRepository()
	body := `[{"id":"PollCount","type":"counter","delta":5},{"id":"Alloc","type":"gauge","value":3.14}]`

	w := doRequest(t, repo, body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(5), repo.counters["PollCount"])
	assert.InDelta(t, 3.14, repo.gauges["Alloc"], 0.001)
}

func TestNew_Batch_CounterAccumulatesDuplicates(t *testing.T) {
	repo := newMockRepository()
	body := `[{"id":"PollCount","type":"counter","delta":5},{"id":"PollCount","type":"counter","delta":3}]`

	w := doRequest(t, repo, body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int64(8), repo.counters["PollCount"])
}

func TestNew_EmptyBatch_OK(t *testing.T) {
	repo := newMockRepository()
	w := doRequest(t, repo, `[]`)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestNew_InvalidJSON_BadRequest(t *testing.T) {
	repo := newMockRepository()
	w := doRequest(t, repo, `{invalid`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNew_InvalidMetricType_BadRequest(t *testing.T) {
	repo := newMockRepository()
	w := doRequest(t, repo, `[{"id":"x","type":"unknown"}]`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNew_CounterMissingDelta_BadRequest(t *testing.T) {
	repo := newMockRepository()
	w := doRequest(t, repo, `[{"id":"x","type":"counter"}]`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNew_MissingContentType_UnsupportedMediaType(t *testing.T) {
	handler := New(slog.Default(), service.New(newMockRepository()))
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(`[]`))
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/updates/", handler)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestNew_RepositoryError_InternalServerError(t *testing.T) {
	repo := newMockRepository()
	repo.err = errors.New("db error")
	w := doRequest(t, repo, `[{"id":"PollCount","type":"counter","delta":5}]`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
