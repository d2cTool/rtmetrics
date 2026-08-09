package agent

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/update"
	"github.com/d2cTool/rtmetrics/internal/handler/updates"
	compressmw "github.com/d2cTool/rtmetrics/internal/middleware/compress"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentServerIntegration(t *testing.T) {
	log := slog.Default()
	st := storage.New()
	svc := service.New(st)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/update", update.New(log, svc))
	r.Get("/value/{mtype}/{name}", get.New(log, svc))

	server := httptest.NewServer(r)
	defer server.Close()

	client := NewClient(server.URL, log)

	// Отправляем метрики так же, как это делает агент
	require.NoError(t, client.SendGauge("Alloc", 12345.67))
	require.NoError(t, client.SendGauge("RandomValue", 0.42))
	require.NoError(t, client.SendCounter("PollCount", 7))

	// Проверяем, что сервер сохранил и отдаёт значения
	assertMetric(t, server.URL, "gauge", "Alloc", "12345.67")
	assertMetric(t, server.URL, "gauge", "RandomValue", "0.42")
	assertMetric(t, server.URL, "counter", "PollCount", "7")

	// Повторная отправка counter — накопление
	require.NoError(t, client.SendCounter("PollCount", 3))
	assertMetric(t, server.URL, "counter", "PollCount", "10")
}

func TestAgentServerBatchIntegration(t *testing.T) {
	log := slog.Default()
	st := storage.New()
	svc := service.New(st)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(compressmw.New(log))
	r.Post("/updates/", updates.New(log, svc))
	r.Get("/value/{mtype}/{name}", get.New(log, svc))

	server := httptest.NewServer(r)
	defer server.Close()

	client := NewClient(server.URL, log)

	batch := BuildBatch(GaugeMetrics{Alloc: 100.5, RandomValue: 0.5}, CountMetrics{PollCount: 4})
	require.NoError(t, client.SendBatch(context.Background(), batch))

	assertMetric(t, server.URL, "gauge", "Alloc", "100.5")
	assertMetric(t, server.URL, "counter", "PollCount", "4")

	// Повторный батч — counter накапливается.
	require.NoError(t, client.SendBatch(context.Background(), BuildBatch(GaugeMetrics{}, CountMetrics{PollCount: 6})))
	assertMetric(t, server.URL, "counter", "PollCount", "10")
}

func assertMetric(t *testing.T, baseURL, mtype, name, expected string) {
	t.Helper()
	resp, err := http.Get(baseURL + "/value/" + mtype + "/" + name)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /value/%s/%s", mtype, name)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body), "metric %s/%s", mtype, name)
}
