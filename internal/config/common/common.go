package common

import (
	"log/slog"
	"os"
)

const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case EnvProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	}

	return log
}
