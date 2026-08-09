package ping

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Pinger — минимальный интерфейс проверки доступности хранилища.
// Его реализует *sql.DB, поэтому хендлер не завязан на конкретный драйвер.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// New возвращает обработчик GET /ping: 200 OK при доступной БД,
// 500 Internal Server Error — при ошибке соединения или если БД не настроена.
func New(log *slog.Logger, db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.ping.new"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if db == nil {
			log.Error("database is not configured")
			http.Error(w, "Database is not configured", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Error("database ping failed", slog.String("error", err.Error()))
			http.Error(w, "Database connection failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
