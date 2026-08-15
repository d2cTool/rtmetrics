// Package get реализует GET /value/{mtype}/{name}.
package get

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New возвращает хендлер GET /value/{mtype}/{name}: тело ответа — значение метрики.
func New(log *slog.Logger, svc service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.get.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		mtype := chi.URLParam(r, "mtype")
		name := chi.URLParam(r, "name")

		if mtype != metrics.Counter && mtype != metrics.Gauge {
			log.Error("invalid metric type",
				slog.String("mtype", mtype),
			)
			http.Error(w, "Invalid metric type. Must be 'gauge' or 'counter'", http.StatusNotFound)
			return
		}

		resp := ""

		if mtype == metrics.Counter {
			value, err := svc.GetCounter(r.Context(), name)
			if err != nil {
				log.Error("failed to get counter",
					slog.String("error", err.Error()),
					slog.String("name", name),
				)
				http.Error(w, "Counter not found", http.StatusNotFound)
				return
			}
			resp = strconv.FormatInt(value, 10)
		}

		if mtype == metrics.Gauge {
			value, err := svc.GetGauge(r.Context(), name)
			if err != nil {
				log.Error("failed to get gauge",
					slog.String("error", err.Error()),
					slog.String("name", name),
				)
				http.Error(w, "Gauge not found", http.StatusNotFound)
				return
			}
			resp = strconv.FormatFloat(value, 'f', -1, 64)
		}

		log.Info("data retrieved",
			slog.String("mtype", mtype),
			slog.String("name", name),
			slog.String("value", resp),
		)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, resp)
	}
}
