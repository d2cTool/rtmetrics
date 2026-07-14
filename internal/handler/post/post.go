package post

import (
	"log/slog"
	"net/http"
	"strconv"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, svc service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.post.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		mtype := chi.URLParam(r, "mtype")
		name := chi.URLParam(r, "name")
		valueStr := chi.URLParam(r, "value")

		if mtype != metrics.Counter && mtype != metrics.Gauge {
			log.Error("invalid metric type",
				slog.String("mtype", mtype),
			)
			http.Error(w, "Invalid metric type. Must be 'gauge' or 'counter'", http.StatusBadRequest)
			return
		}

		resp := ""

		if mtype == metrics.Counter {

			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				log.Error("failed to counter value",
					slog.String("error", err.Error()),
					slog.String("value", valueStr),
				)
				http.Error(w, "Invalid value parameter", http.StatusBadRequest)
				return
			}

			newValue, err := svc.UpdateCounter(r.Context(), name, value)
			if err != nil {
				log.Error("failed to save counter",
					slog.String("error", err.Error()),
					slog.String("name", name),
					slog.Int64("value", value),
				)
				http.Error(w, "Failed to save counter", http.StatusInternalServerError)
				return
			}
			resp = strconv.FormatInt(newValue, 10)
		}

		if mtype == metrics.Gauge {

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				log.Error("failed to parse value",
					slog.String("error", err.Error()),
					slog.String("value", valueStr),
				)
				http.Error(w, "Invalid gauge parameter", http.StatusBadRequest)
				return
			}

			newValue, err := svc.UpdateGauge(r.Context(), name, value)
			if err != nil {
				log.Error("failed to save gauge",
					slog.String("error", err.Error()),
					slog.String("name", name),
					slog.Float64("value", value),
				)
				http.Error(w, "Failed to save gauge", http.StatusInternalServerError)
				return
			}
			resp = strconv.FormatFloat(newValue, 'f', -1, 64)
		}

		log.Info("data saved",
			slog.String("mtype", mtype),
			slog.String("name", name),
			slog.String("value", valueStr),
			slog.String("new_value", resp),
		)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp))
	}
}
