package updates

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
)

// New возвращает обработчик POST /updates/, принимающий пакет метрик []Metrics.
func New(log *slog.Logger, svc service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.updates.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			log.Error("unsupported content type", slog.String("content_type", r.Header.Get("Content-Type")))
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var batch []metrics.Metrics
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			log.Error("cannot decode request JSON body", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(batch) == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}

		for i, m := range batch {
			switch m.MType {
			case metrics.Counter:
				if m.Delta == nil {
					log.Error("missing counter value", slog.Int("index", i), slog.String("id", m.ID))
					http.Error(w, "Invalid counter value", http.StatusBadRequest)
					return
				}
			case metrics.Gauge:
				if m.Value == nil {
					log.Error("missing gauge value", slog.Int("index", i), slog.String("id", m.ID))
					http.Error(w, "Invalid gauge value", http.StatusBadRequest)
					return
				}
			default:
				log.Error("invalid metric type", slog.Int("index", i), slog.String("mtype", m.MType))
				http.Error(w, "Invalid metric type. Must be 'gauge' or 'counter'", http.StatusBadRequest)
				return
			}
		}

		if err := svc.UpdateBatch(r.Context(), batch); err != nil {
			log.Error("failed to save metrics batch", slog.String("error", err.Error()))
			http.Error(w, "Failed to save metrics", http.StatusInternalServerError)
			return
		}

		log.Info("metrics batch saved", slog.Int("count", len(batch)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}
