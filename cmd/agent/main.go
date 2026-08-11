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
		slog.Int("poll_interval", cfg.PollInterval),
		slog.Int("report_interval", cfg.ReportInterval),
		slog.Int("rate_limit", cfg.RateLimit),
		slog.Bool("signing_enabled", cfg.Key != ""),
	)

	client := agent.NewClient("http://"+cfg.Address, cfg.Key, log)
	runner := agent.NewRunner(
		client,
		log,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
		cfg.RateLimit,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner.Run(ctx)

	log.Info("agent stopped")
}
