package value

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, repo repository.MetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.value.new"
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

		if req.MType == metrics.Counter {
			value, err := repo.GetCounter(r.Context(), req.ID)
			if err != nil {
				log.Error("failed to get counter",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
				)
				http.Error(w, "Counter not found", http.StatusNotFound)
				return
			}
			req.Delta = &value
		}

		if req.MType == metrics.Gauge {
			value, err := repo.GetGauge(r.Context(), req.ID)
			if err != nil {
				log.Error("failed to get gauge",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
				)
				http.Error(w, "Gauge not found", http.StatusNotFound)
				return
			}
			req.Value = &value
		}

		log.Info("data retrieved",
			slog.String("mtype", req.MType),
			slog.String("name", req.ID),
		)

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			log.Error("cannot encode response JSON body",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}
