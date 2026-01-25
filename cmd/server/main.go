package main

import (
	"log/slog"
	"net/http"

	common "github.com/d2cTool/rtmetrics/internal/config/common"
	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/html"
	"github.com/d2cTool/rtmetrics/internal/handler/post"
	mwLogger "github.com/d2cTool/rtmetrics/internal/middleware/logger"
)

func main() {
	cfg := config.Load()

	log := common.SetupLogger(cfg.Env)
	log.Info("starting server", slog.String("env", cfg.Env))

	storage := storage.New()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))

	router.Post("/{mtype}/{name}/{value}", post.New(log, storage))
	router.Post("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	router.Get("/value/{mtype}/{name}", get.New(log, storage))
	router.Get("/", html.New(log, storage))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	srv := &http.Server{
		Addr:         cfg.HttpServer.Address,
		ReadTimeout:  cfg.HttpServer.ReadTimeout,
		WriteTimeout: cfg.HttpServer.WriteTimeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
		Handler:      router,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", slog.String("error", err.Error()))
	}

	log.Error("server stopped")
}
