package config

import (
	"flag"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
)

type AgentConfig struct {
	Env            string
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func Load() *AgentConfig {
	var cfg = AgentConfig{Env: common.EnvLocal, Address: "localhost:8080", ReportInterval: 10, PollInterval: 2}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	return &cfg
}
