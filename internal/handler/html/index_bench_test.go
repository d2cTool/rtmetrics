package html

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/service"
)

func BenchmarkHTMLRender(b *testing.B) {
	repo := newMockRepository()
	repo.counters["PollCount"] = 7
	repo.gauges["Alloc"] = 123.4
	repo.gauges["Frees"] = 10
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(log, service.New(repo))

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}
