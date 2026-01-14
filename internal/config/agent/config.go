package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	Env            string
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func Load() *AgentConfig {
	var cfg = AgentConfig{Env: "local", Address: "localhost:8080", ReportInterval: 10 * time.Second, PollInterval: 2 * time.Second}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.DurationVar(&cfg.ReportInterval, "r", 10*time.Second, "report interval")
	flag.DurationVar(&cfg.PollInterval, "p", 2*time.Second, "poll interval")
	flag.Parse()
	return &cfg
}
