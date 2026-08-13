package agent

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/update"
	"github.com/d2cTool/rtmetrics/internal/handler/updates"
	"github.com/d2cTool/rtmetrics/internal/hash"
	compressmw "github.com/d2cTool/rtmetrics/internal/middleware/compress"
	signmw "github.com/d2cTool/rtmetrics/internal/middleware/sign"
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

	client := NewClient(server.URL, "", log)

	ctx := t.Context()

	// Отправляем метрики так же, как это делает агент
	require.NoError(t, client.SendGauge(ctx, "Alloc", 12345.67))
	require.NoError(t, client.SendGauge(ctx, "RandomValue", 0.42))
	require.NoError(t, client.SendCounter(ctx, "PollCount", 7))

	// Проверяем, что сервер сохранил и отдаёт значения
	assertMetric(t, server.URL, "gauge", "Alloc", "12345.67")
	assertMetric(t, server.URL, "gauge", "RandomValue", "0.42")
	assertMetric(t, server.URL, "counter", "PollCount", "7")

	// Повторная отправка counter — накопление
	require.NoError(t, client.SendCounter(ctx, "PollCount", 3))
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

	client := NewClient(server.URL, "", log)

	batch := BuildBatch(GaugeMetrics{Alloc: 100.5, RandomValue: 0.5}, CountMetrics{PollCount: 4})
	require.NoError(t, client.SendBatch(t.Context(), batch))

	assertMetric(t, server.URL, "gauge", "Alloc", "100.5")
	assertMetric(t, server.URL, "counter", "PollCount", "4")

	// Повторный батч — counter накапливается.
	require.NoError(t, client.SendBatch(t.Context(), BuildBatch(GaugeMetrics{}, CountMetrics{PollCount: 6})))
	assertMetric(t, server.URL, "counter", "PollCount", "10")
}

// Батч уходит сжатым, а подписывается до сжатия — сервер считает хеш после
// распаковки, поэтому обе стороны должны сойтись именно на исходном JSON.
func TestAgentServerSignedBatchIntegration(t *testing.T) {
	const key = "super-secret"

	log := slog.Default()
	st := storage.New()
	svc := service.New(st)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(compressmw.New(log))
	r.Use(signmw.New(log, key))
	r.Post("/updates/", updates.New(log, svc))
	r.Get("/value/{mtype}/{name}", get.New(log, svc))

	server := httptest.NewServer(r)
	defer server.Close()

	batch := BuildBatch(GaugeMetrics{Alloc: 100.5}, CountMetrics{PollCount: 4})
	require.NoError(t, NewClient(server.URL, key, log).SendBatch(t.Context(), batch))

	assertMetric(t, server.URL, "gauge", "Alloc", "100.5")
	assertMetric(t, server.URL, "counter", "PollCount", "4")

	// Агент с чужим ключом получает 400 и метрики не меняет.
	stale := BuildBatch(GaugeMetrics{Alloc: 999}, CountMetrics{PollCount: 100})
	require.Error(t, NewClient(server.URL, "wrong-key", log).SendBatch(t.Context(), stale))
	assertMetric(t, server.URL, "gauge", "Alloc", "100.5")

	// Ответ сервера тоже подписан.
	resp, err := http.Get(server.URL + "/value/gauge/Alloc")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, hash.Sign(body, key), resp.Header.Get(hash.Header))
}

func TestClientDoesNotSendOnCancelledContext(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", slog.Default())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, client.SendGauge(ctx, "Alloc", 1), context.Canceled)
	require.ErrorIs(t, client.SendCounter(ctx, "PollCount", 1), context.Canceled)
	require.ErrorIs(t, client.SendBatch(ctx, BuildBatch(GaugeMetrics{}, CountMetrics{PollCount: 1})), context.Canceled)
	require.ErrorIs(t, client.SendGaugeMetrics(ctx, GaugeMetrics{}), context.Canceled)

	assert.Zero(t, requests.Load(), "запросы не должны уходить при отменённом контексте")
}

func TestClientAbortsInFlightRequestOnCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", slog.Default())

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := client.SendGauge(ctx, "Alloc", 1)
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, time.Second, "запрос должен прерваться по отмене, а не дожидаться ответа или повтора")
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
