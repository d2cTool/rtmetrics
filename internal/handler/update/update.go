// Package update реализует POST /update — запись одной метрики в JSON.
package update

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/d2cTool/rtmetrics/internal/audit"
	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
)

// New возвращает хендлер POST /update и POST /update/: JSON одной метрики.
func New(log *slog.Logger, svc service.MetricsService) http.HandlerFunc {
	return NewWithAudit(log, svc, nil)
}

// NewWithAudit как New, после успешного сохранения уведомляет auditor.
func NewWithAudit(log *slog.Logger, svc service.MetricsService, auditor *audit.Subject) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.update.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			log.Error("unsupported content type", slog.String("content_type", r.Header.Get("Content-Type")))
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var req metrics.Metrics
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			log.Error("cannot decode request JSON body",
				slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.MType != metrics.Counter && req.MType != metrics.Gauge {
			log.Error("invalid metric type",
				slog.String("mtype", req.MType),
			)
			http.Error(w, "Invalid metric type. Must be 'gauge' or 'counter'", http.StatusBadRequest)
			return
		}

		var resp *metrics.Metrics

		if req.MType == metrics.Counter {
			if req.Delta == nil {
				log.Error("invalid counter value")
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}

			newValue, err := svc.UpdateCounter(r.Context(), req.ID, *req.Delta)
			if err != nil {
				log.Error("failed to save counter",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
					slog.Int64("value", *req.Delta),
				)
				http.Error(w, "Failed to save counter", http.StatusInternalServerError)
				return
			}
			resp = &metrics.Metrics{ID: req.ID, MType: metrics.Counter, Delta: &newValue}
		}

		if req.MType == metrics.Gauge {
			if req.Value == nil {
				log.Error("invalid gauge value")
				http.Error(w, "Invalid gauge value", http.StatusBadRequest)
				return
			}

			newValue, err := svc.UpdateGauge(r.Context(), req.ID, *req.Value)
			if err != nil {
				log.Error("failed to save gauge",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
					slog.Float64("value", *req.Value),
				)
				http.Error(w, "Failed to save gauge", http.StatusInternalServerError)
				return
			}
			resp = &metrics.Metrics{ID: req.ID, MType: metrics.Gauge, Value: &newValue}
		}

		log.Debug("data saved",
			slog.String("mtype", req.MType),
			slog.String("name", req.ID),
		)
		auditor.NotifyRequest(r, []string{req.ID})

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("cannot encode response JSON body",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
