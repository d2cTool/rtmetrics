package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
)

type ServerConfig struct {
	Env             string
	HTTPServer      *HTTPServerConfig
	StoreInterval   time.Duration `env:"STORE_INTERVAL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	Restore         bool          `env:"RESTORE"`
}

type HTTPServerConfig struct {
	Address      string `env:"SERVER_ADDRESS"`
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() *ServerConfig {
	var httpSrv = HTTPServerConfig{Address: "localhost:8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	var cfg = ServerConfig{Env: common.EnvLocal, HTTPServer: &httpSrv, StoreInterval: 1 * time.Second, FileStoragePath: "./data", Restore: false}

	err := env.Parse(&cfg)
	if err == nil && cfg.HTTPServer.Address != "" {
		return &cfg
	}

	flag.StringVar(&cfg.HTTPServer.Address, "a", "localhost:8080", "server address")
	flag.DurationVar(&cfg.StoreInterval, "i", 300*time.Second, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "./data", "file storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "restore")
	flag.Parse()
	return &cfg
}
