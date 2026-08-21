// Package config загружает конфигурацию агента из флагов и окружения.
package config

import (
	"flag"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
)

// AgentConfig — флаги и переменные окружения процесса агента.
type AgentConfig struct {
	Env            string
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

// Load читает флаги, затем перекрывает их переменными окружения.
func Load() *AgentConfig {
	var cfg = AgentConfig{Env: common.EnvLocal, Address: "localhost:8080", ReportInterval: 10, PollInterval: 2, RateLimit: 1}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "report interval")
	flag.IntVar(&cfg.PollInterval, "p", 2, "poll interval")
	flag.StringVar(&cfg.Key, "k", "", "key for request signing (HMAC-SHA256)")
	flag.IntVar(&cfg.RateLimit, "l", 1, "max number of simultaneous outgoing requests")
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	return &cfg
}
