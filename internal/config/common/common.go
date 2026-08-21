// Package common — общие константы окружения и настройка логгера.
package common

import (
	"fmt"
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

// PrintBuildInfo пишет в stdout данные сборки. Пустые значения заменяются на N/A.
func PrintBuildInfo(version, date, commit string) {
	fmt.Printf("Build version: %s\n", orNA(version))
	fmt.Printf("Build date: %s\n", orNA(date))
	fmt.Printf("Build commit: %s\n", orNA(commit))
}

func orNA(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
