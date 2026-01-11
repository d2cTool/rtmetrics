package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/d2cTool/rtmetrics/internal/config"
	//"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	//saveCounter "github.com/d2cTool/rtmetrics/internal/handler/counter"
	mwLogger "github.com/d2cTool/rtmetrics/internal/middleware/logger"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	cfg := config.Load()
	fmt.Println("config:", cfg) // TODO: remove only for debug

	log := setupLogger(cfg.Env)
	log.Info("starting server", slog.String("env", cfg.Env))

	//var storage MemStorage

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.URLFormat)

	//router.Post("/counter/{name}/{value}", saveCounter.New(log, storage))
	//router.Post("/gauge/{name}/{value}", saveCounter.New(log, storage))
	//router.Post("/", saveCounter.New(log, storage))
	//
	//router.Get("/value/counter/{name}", saveCounter.New(log, storage))
	//router.Get("/value/gauge/{name}", saveCounter.New(log, storage))
	//router.Get("/", saveCounter.New(log, storage))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}),
		)
	}

	return log
}
