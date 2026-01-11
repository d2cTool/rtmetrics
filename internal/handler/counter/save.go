package counter

import (
	"log/slog"
	"net/http"

	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/go-chi/chi/v5/middleware"
)

func New(log *slog.Logger, saver repository.MetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.save.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		log.Info("request", slog.Any("request", r))

		//saver.
	}
}
