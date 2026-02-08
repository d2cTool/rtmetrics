package update

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, repo repository.MetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.update.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		resp := ""

		if req.MType == metrics.Counter {
			if req.Delta == nil {
				log.Error("invalid counter value")
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}

			newValue, err := repo.SaveCounter(r.Context(), req.ID, *req.Delta)
			if err != nil {
				log.Error("failed to save counter",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
					slog.Int64("value", *req.Delta),
				)
				http.Error(w, "Failed to save counter", http.StatusInternalServerError)
				return
			}
			resp = strconv.FormatInt(newValue, 10)
		}

		if req.MType == metrics.Gauge {
			if req.Value == nil {
				log.Error("invalid gauge value")
				http.Error(w, "Invalid gauge value", http.StatusBadRequest)
				return
			}

			newValue, err := repo.SaveGauge(r.Context(), req.ID, *req.Value)
			if err != nil {
				log.Error("failed to save gauge",
					slog.String("error", err.Error()),
					slog.String("name", req.ID),
					slog.Float64("value", *req.Value),
				)
				http.Error(w, "Failed to save gauge", http.StatusInternalServerError)
				return
			}
			resp = strconv.FormatFloat(newValue, 'f', -1, 64)
		}

		log.Info("data saved",
			slog.String("mtype", req.MType),
			slog.String("name", req.ID),
			slog.String("new_value", resp),
		)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}
}
