package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
)

type AgentConfig struct {
	Env            string
	Address        string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

func Load() *AgentConfig {
	var cfg = AgentConfig{Env: common.EnvLocal, Address: "localhost:8080", ReportInterval: 10 * time.Second, PollInterval: 2 * time.Second}

	err := env.Parse(&cfg)
	if err == nil && cfg.Address != "" {
		return &cfg
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.DurationVar(&cfg.ReportInterval, "r", 10*time.Second, "report interval")
	flag.DurationVar(&cfg.PollInterval, "p", 2*time.Second, "poll interval")
	flag.Parse()
	return &cfg
}
