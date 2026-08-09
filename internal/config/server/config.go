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
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

type HTTPServerConfig struct {
	Address      string `env:"ADDRESS"`
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() *ServerConfig {
	var httpSrv = HTTPServerConfig{Address: "localhost:8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	var cfg = ServerConfig{Env: common.EnvLocal, HTTPServer: &httpSrv, StoreInterval: 300, FileStoragePath: "./tmp/data", Restore: false}

	flag.StringVar(&cfg.HTTPServer.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "./tmp/data", "file storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "restore")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN (PostgreSQL)")
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	return &cfg
}
