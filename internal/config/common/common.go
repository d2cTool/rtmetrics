// Package common — общие константы окружения и настройка логгера.
package common

import (
	"log/slog"
	"os"
)

const (
	// EnvLocal — текстовые логи уровня Info (стенд, разработка).
	EnvLocal = "local"
	// EnvProd — JSON-логи уровня Info.
	EnvProd = "prod"
)

// SetupLogger возвращает slog.Logger для env (local или prod).
func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case EnvProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
