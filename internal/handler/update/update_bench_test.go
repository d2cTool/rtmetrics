package update

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func BenchmarkUpdateGauge(b *testing.B) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(log, service.New(newMockRepository()))
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/update", handler)

	body := `{"id":"Alloc","type":"gauge","value":1.5}`
	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}
