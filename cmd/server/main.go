package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/d2cTool/rtmetrics/internal/audit"
	"github.com/d2cTool/rtmetrics/internal/config/common"
	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/database"
	"github.com/d2cTool/rtmetrics/internal/handler/ping"
	"github.com/d2cTool/rtmetrics/internal/handler/update"
	"github.com/d2cTool/rtmetrics/internal/handler/updates"
	"github.com/d2cTool/rtmetrics/internal/handler/value"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/d2cTool/rtmetrics/internal/storage/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/html"
	"github.com/d2cTool/rtmetrics/internal/handler/post"
	"github.com/d2cTool/rtmetrics/internal/middleware/compress"
	"github.com/d2cTool/rtmetrics/internal/middleware/decrypt"
	"github.com/d2cTool/rtmetrics/internal/middleware/logger"
	"github.com/d2cTool/rtmetrics/internal/middleware/sign"
	"github.com/d2cTool/rtmetrics/internal/rsaenc"
	"github.com/d2cTool/rtmetrics/internal/server"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	common.PrintBuildInfo(buildVersion, buildDate, buildCommit)

	if err := run(); err != nil {
		slog.Error("server failed", slog.String("error", err.Error()))
		exit(1)
	}
}

func exit(code int) {
	os.Exit(code)
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := common.SetupLogger(cfg.Env)
	log.Info("starting server",
		slog.String("env", cfg.Env),
		slog.String("address", cfg.HTTPServer.Address),
		slog.Int("store_interval", cfg.StoreInterval),
		slog.String("file_storage_path", cfg.FileStoragePath),
		slog.Bool("restore", cfg.Restore),
		slog.String("database", cfg.DatabaseDSN),
		slog.Int("db_max_open_conns", cfg.Database.MaxOpenConns),
		slog.Int("db_max_idle_conns", cfg.Database.MaxIdleConns),
		slog.Duration("db_conn_max_idle_time", cfg.Database.ConnMaxIdleTime),
		slog.Duration("db_conn_max_lifetime", cfg.Database.ConnMaxLifetime),
		slog.Bool("signing_enabled", cfg.Key != ""),
		slog.String("crypto_key", cfg.CryptoKey),
		slog.String("audit_file", cfg.AuditFile),
		slog.String("audit_url", cfg.AuditURL),
	)

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		var err error
		db, err = database.New(context.Background(), cfg.DatabaseDSN, cfg.Database)
		if err != nil {
			log.Error("failed to connect to database", slog.String("error", err.Error()))
		} else {
			log.Info("connected to database")
			defer db.Close()
		}
	}

	// Выбор хранилища по приоритету: PostgreSQL -> файл -> память.
	// memSt != nil означает файловый/in-memory режим (нужен для снапшота при завершении).
	var (
		repo  repository.MetricsRepository
		memSt *storage.MemStorage
	)

	if db != nil {
		pg, err := postgres.New(context.Background(), db)
		if err != nil {
			log.Error("failed to initialize postgres storage, falling back", slog.String("error", err.Error()))
		} else {
			repo = pg
			log.Info("using postgres storage")
		}
	}

	saveCtx, stopSave := context.WithCancel(context.Background())
	defer stopSave()
	var saveWG sync.WaitGroup

	if repo == nil {
		memSt = storage.New()
		server.RestoreIfNeeded(cfg, memSt, log)

		switch {
		case cfg.FileStoragePath != "" && cfg.StoreInterval == 0:
			repo = server.NewSyncSaveRepo(memSt, cfg, log)
			log.Info("using in-memory storage with synchronous file persistence")
		case cfg.FileStoragePath != "":
			saveWG.Add(1)
			go func() {
				defer saveWG.Done()
				server.RunPeriodicSave(saveCtx, cfg, memSt, log)
			}()
			repo = memSt
			log.Info("using in-memory storage with periodic file persistence")
		default:
			repo = memSt
			log.Info("using in-memory storage")
		}
	}

	svc := service.New(repo)

	var pinger ping.Pinger
	if db != nil {
		pinger = db
	}

	auditor := newAuditor(log, cfg.AuditFile, cfg.AuditURL)
	defer auditor.Close()

	var priv *rsa.PrivateKey
	if cfg.CryptoKey != "" {
		var err error
		priv, err = rsaenc.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			return err
		}
		log.Info("request decryption enabled")
	}

	router := createRouter(log, svc, pinger, cfg.Key, priv, auditor)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		ReadTimeout:  cfg.HTTPServer.ReadTimeout,
		WriteTimeout: cfg.HTTPServer.WriteTimeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
		Handler:      router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), common.ShutdownSignals()...)
	defer stop()

	serverExited := make(chan error, 1)
	go func() { serverExited <- srv.ListenAndServe() }()

	select {
	case err := <-serverExited:
		if err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server", slog.String("error", err.Error()))
		}
	case <-ctx.Done():
		stop()
		log.Info("shutdown signal received")
		timeout := cfg.HTTPServer.WriteTimeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		shutCtx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Error("server shutdown error", slog.String("error", err.Error()))
		}
		cancel()
		<-serverExited
	}

	stopSave()
	saveWG.Wait()
	if memSt != nil {
		server.SaveSnapshot(cfg, memSt, log)
	}
	log.Info("server stopped")
	return nil
}

func newAuditor(log *slog.Logger, file, url string) *audit.Subject {
	if file == "" && url == "" {
		return nil
	}

	subject := audit.NewSubject()
	n := 0
	if file != "" {
		obs, err := audit.NewFileObserver(file, log)
		if err != nil {
			log.Error("failed to open audit file", slog.String("path", file), slog.String("error", err.Error()))
		} else {
			subject.Subscribe(obs)
			log.Info("audit file observer enabled", slog.String("path", file))
			n++
		}
	}
	if url != "" {
		subject.Subscribe(audit.NewHTTPObserver(url, log))
		log.Info("audit http observer enabled", slog.String("url", url))
		n++
	}
	if n == 0 {
		subject.Close()
		return nil
	}
	return subject
}

func createRouter(log *slog.Logger, svc service.MetricsService, pinger ping.Pinger, key string, priv *rsa.PrivateKey, auditor *audit.Subject) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	if priv != nil {
		router.Use(decrypt.New(log, priv))
	}
	router.Use(compress.New(log))
	if key != "" {
		router.Use(sign.New(log, key))
	}

	router.Mount("/debug", middleware.Profiler())

	router.Get("/ping", ping.New(log, pinger))

	router.Post("/update", update.NewWithAudit(log, svc, auditor))
	router.Post("/update/", update.NewWithAudit(log, svc, auditor))
	router.Post("/updates", updates.NewWithAudit(log, svc, auditor))
	router.Post("/updates/", updates.NewWithAudit(log, svc, auditor))
	router.Post("/value", value.New(log, svc))
	router.Post("/value/", value.New(log, svc))

	router.Post("/update/{mtype}/{name}/{value}", post.NewWithAudit(log, svc, auditor))
	router.Post("/{mtype}/{name}/{value}", post.NewWithAudit(log, svc, auditor))
	router.Post("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	router.Get("/value/{mtype}/{name}", get.New(log, svc))
	router.Get("/", html.New(log, svc))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	return router
}
