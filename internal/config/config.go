package config

import (
	"flag"
	"time"
)

type Config struct {
	Env            string `yaml:"env" env:"ENV" env-default:"local"`
	HTTPServer     `yaml:"http_server"`
	ReportInterval time.Duration `yaml:"report_interval" env-default:"10s"`
	PollInterval   time.Duration `yaml:"poll_interval" env-default:"2s"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"10s"`
	IdleTimeout time.Duration `yaml:"idleTimeout" env-default:"10s"`
}

func Load() *Config {
	var srv = HTTPServer{Address: "localhost:8080", Timeout: 10 * time.Second, IdleTimeout: 10 * time.Second}
	var cfg = Config{Env: "local", HTTPServer: srv}

	flag.StringVar(&cfg.HTTPServer.Address, "a", "localhost:8080", "server address")
	flag.DurationVar(&cfg.ReportInterval, "r", 10*time.Second, "report interval")
	flag.DurationVar(&cfg.PollInterval, "p", 2*time.Second, "poll interval")
	flag.Parse()
	return &cfg
}
