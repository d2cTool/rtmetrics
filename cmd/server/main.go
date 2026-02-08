package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d2cTool/rtmetrics/internal/config/common"
	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/handler/update"
	"github.com/d2cTool/rtmetrics/internal/handler/value"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/html"
	"github.com/d2cTool/rtmetrics/internal/handler/post"
	"github.com/d2cTool/rtmetrics/internal/middleware/compress"
	"github.com/d2cTool/rtmetrics/internal/middleware/logger"
	"github.com/d2cTool/rtmetrics/internal/server"
)

func main() {
	cfg := config.Load()

	log := common.SetupLogger(cfg.Env)
	log.Info("starting server",
		slog.String("env", cfg.Env),
		slog.Duration("store_interval", cfg.StoreInterval),
		slog.String("file_storage_path", cfg.FileStoragePath),
		slog.Bool("restore", cfg.Restore),
	)

	st := storage.New()

	server.RestoreIfNeeded(cfg, st, log)

	var repo repository.MetricsRepository = st
	if cfg.StoreInterval == 0 && cfg.FileStoragePath != "" {
		repo = server.NewSyncSaveRepo(st, cfg, log)
	} else if cfg.StoreInterval > 0 && cfg.FileStoragePath != "" {
		go server.RunPeriodicSave(cfg, st, log)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(compress.New(log))

	router.Post("/update", update.New(log, repo))
	router.Post("/value", value.New(log, repo))

	router.Post("/{mtype}/{name}/{value}", post.New(log, repo))
	router.Post("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	router.Get("/value/{mtype}/{name}", get.New(log, repo))
	router.Get("/", html.New(log, repo))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	srv := &http.Server{
		Addr:         cfg.HttpServer.Address,
		ReadTimeout:  cfg.HttpServer.ReadTimeout,
		WriteTimeout: cfg.HttpServer.WriteTimeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
		Handler:      router,
	}

	serverExited := make(chan error, 1)
	go func() { serverExited <- srv.ListenAndServe() }()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverExited:
		if err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server", slog.String("error", err.Error()))
		}
	case <-sigChan:
		log.Info("shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("server shutdown error", slog.String("error", err.Error()))
		}
		cancel()
		<-serverExited
	}

	server.SaveSnapshot(cfg, st, log)
	log.Info("server stopped")
}
