package config

import (
	"flag"
)

type ServerConfig struct {
	Env     string
	Address string
}

func Load() *ServerConfig {
	var cfg = ServerConfig{Env: "local", Address: "localhost:8080"}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.Parse()
	return &cfg
}
