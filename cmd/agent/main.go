package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d2cTool/rtmetrics/internal/agent"
	config "github.com/d2cTool/rtmetrics/internal/config/agent"
	common "github.com/d2cTool/rtmetrics/internal/config/common"
)

func main() {
	cfg := config.Load()

	log := common.SetupLogger(cfg.Env)
	log.Info("starting agent",
		slog.String("env", cfg.Env),
		slog.String("server_address", cfg.Address),
		slog.Duration("poll_interval", cfg.PollInterval),
		slog.Duration("report_interval", cfg.ReportInterval),
	)

	client := agent.NewClient("http://"+cfg.Address, log)
	runner := agent.NewRunner(client, log, cfg.PollInterval, cfg.ReportInterval)

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем сбор и отправку метрик
	runner.Start(ctx)

	// Ожидаем сигнал для завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info("shutting down agent...")
	cancel()

	// Даем время горутинам завершиться
	time.Sleep(100 * time.Millisecond)

	log.Info("agent stopped")
}
