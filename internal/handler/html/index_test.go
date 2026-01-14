package html

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository - мок для тестирования
type mockRepository struct {
	counters          map[string]int64
	gauges            map[string]float64
	err               error
	getAllCountersErr error
	getAllGaugesErr   error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *mockRepository) SaveCounter(name string, value int64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.counters == nil {
		m.counters = make(map[string]int64)
	}
	m.counters[name] += value
	return m.counters[name], nil
}

func (m *mockRepository) SaveGauge(name string, value float64) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.gauges == nil {
		m.gauges = make(map[string]float64)
	}
	m.gauges[name] = value
	return value, nil
}

func (m *mockRepository) GetCounter(name string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	value, ok := m.counters[name]
	if !ok {
		return 0, errors.New("counter not found")
	}
	return value, nil
}

func (m *mockRepository) GetGauge(name string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	value, ok := m.gauges[name]
	if !ok {
		return 0, errors.New("gauge not found")
	}
	return value, nil
}

func (m *mockRepository) GetAllCounters() (map[string]int64, error) {
	if m.getAllCountersErr != nil {
		return nil, m.getAllCountersErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.counters, nil
}

func (m *mockRepository) GetAllGauges() (map[string]float64, error) {
	if m.getAllGaugesErr != nil {
		return nil, m.getAllGaugesErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.gauges, nil
}

func TestNew_RenderWithData_Success(t *testing.T) {
	repo := newMockRepository()
	repo.counters["counter1"] = 10
	repo.counters["counter2"] = 20
	repo.gauges["gauge1"] = 3.14
	repo.gauges["gauge2"] = 2.71

	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

	body := w.Body.String()

	// Проверяем наличие основных элементов HTML
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "Metrics Dashboard")

	// Проверяем наличие счетчиков
	assert.Contains(t, body, "counter1")
	assert.Contains(t, body, "10")
	assert.Contains(t, body, "counter2")
	assert.Contains(t, body, "20")

	// Проверяем наличие gauges
	assert.Contains(t, body, "gauge1")
	assert.Contains(t, body, "3.14")
	assert.Contains(t, body, "gauge2")
	assert.Contains(t, body, "2.71")
}

func TestNew_RenderEmptyData_Success(t *testing.T) {
	repo := newMockRepository()
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем наличие сообщений об отсутствии данных
	assert.Contains(t, body, "No counters available")
	assert.Contains(t, body, "No gauges available")
}

func TestNew_RenderWithCountersOnly(t *testing.T) {
	repo := newMockRepository()
	repo.counters["counter1"] = 42
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем наличие счетчика
	assert.Contains(t, body, "counter1")
	assert.Contains(t, body, "42")

	// Проверяем наличие сообщения об отсутствии gauges
	assert.Contains(t, body, "No gauges available")
}

func TestNew_RenderWithGaugesOnly(t *testing.T) {
	repo := newMockRepository()
	repo.gauges["gauge1"] = 1.5
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем наличие gauge
	assert.Contains(t, body, "gauge1")
	assert.Contains(t, body, "1.5")

	// Проверяем наличие сообщения об отсутствии counters
	assert.Contains(t, body, "No counters available")
}

func TestNew_RepositoryError_Counters(t *testing.T) {
	repo := newMockRepository()
	repo.getAllCountersErr = errors.New("repository error")
	repo.gauges["gauge1"] = 1.0
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Handler должен обработать ошибку и показать пустые counters
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем, что при ошибке показывается сообщение об отсутствии counters
	assert.Contains(t, body, "No counters available")

	// Gauges должны отображаться нормально
	assert.Contains(t, body, "gauge1")
}

func TestNew_RepositoryError_Gauges(t *testing.T) {
	repo := newMockRepository()
	repo.getAllGaugesErr = errors.New("repository error")
	repo.counters["counter1"] = 10
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Handler должен обработать ошибку и показать пустые gauges
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем, что при ошибке показывается сообщение об отсутствии gauges
	assert.Contains(t, body, "No gauges available")

	// Counters должны отображаться нормально
	assert.Contains(t, body, "counter1")
}

func TestNew_RepositoryError_Both(t *testing.T) {
	repo := newMockRepository()
	repo.getAllCountersErr = errors.New("repository error")
	repo.getAllGaugesErr = errors.New("repository error")
	log := slog.Default()
	handler := New(log, repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Handler должен обработать обе ошибки и показать пустые данные
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// Проверяем наличие сообщений об отсутствии данных
	assert.Contains(t, body, "No counters available")
	assert.Contains(t, body, "No gauges available")
}
