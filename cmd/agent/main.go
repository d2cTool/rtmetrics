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
	"github.com/d2cTool/rtmetrics/internal/rsaenc"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	common.PrintBuildInfo(buildVersion, buildDate, buildCommit)

	if err := run(); err != nil {
		slog.Error("agent failed", slog.String("error", err.Error()))
	}
}

func run() error {
	cfg := config.Load()

	log := common.SetupLogger(cfg.Env)
	log.Info("starting agent",
		slog.String("env", cfg.Env),
		slog.String("server_address", cfg.Address),
		slog.Int("poll_interval", cfg.PollInterval),
		slog.Int("report_interval", cfg.ReportInterval),
		slog.Int("rate_limit", cfg.RateLimit),
		slog.Bool("signing_enabled", cfg.Key != ""),
		slog.String("crypto_key", cfg.CryptoKey),
	)

	client := agent.NewClient("http://"+cfg.Address, cfg.Key, log)
	if cfg.CryptoKey != "" {
		pub, err := rsaenc.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			return err
		}
		client.WithPublicKey(pub)
		log.Info("request encryption enabled")
	}
	runner, err := agent.NewRunner(
		client,
		log,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
		cfg.RateLimit,
	)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner.Run(ctx)

	log.Info("agent stopped")
	return nil
}
