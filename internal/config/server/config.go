package config

import (
	"flag"
	"time"

	"github.com/d2cTool/rtmetrics/internal/config/common"
)

type ServerConfig struct {
	Env        string
	HttpServer *HttpServerConfig
}

type HttpServerConfig struct {
	Address       string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdIdleTimeout time.Duration
}

func Load() *ServerConfig {
	var httpSrv = HttpServerConfig{Address: "localhost:8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdIdleTimeout: 60 * time.Second}
	var cfg = ServerConfig{Env: common.EnvLocal, HttpServer: &httpSrv}

	flag.StringVar(&cfg.HttpServer.Address, "a", "localhost:8080", "server address")
	flag.Parse()
	return &cfg
}
