// Package config загружает конфигурацию HTTP-сервера из флагов и окружения.
package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
	"github.com/d2cTool/rtmetrics/internal/database"
)

// ServerConfig — флаги и переменные окружения процесса сервера.
type ServerConfig struct {
	Env             string
	HTTPServer      *HTTPServerConfig
	Database        *database.Config
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
}

// HTTPServerConfig — адрес и таймауты http.Server.
type HTTPServerConfig struct {
	Address      string `env:"ADDRESS"`
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Load читает флаги, затем перекрывает их переменными окружения.
func Load() *ServerConfig {
	var httpSrv = HTTPServerConfig{Address: "localhost:8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	var dbCfg = database.DefaultConfig()
	var cfg = ServerConfig{Env: common.EnvLocal, HTTPServer: &httpSrv, Database: dbCfg, StoreInterval: 300, FileStoragePath: "./tmp/data", Restore: false}

	flag.StringVar(&cfg.HTTPServer.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&cfg.StoreInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "./tmp/data", "file storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "restore")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN (PostgreSQL)")
	flag.StringVar(&cfg.Key, "k", "", "key for request signing (HMAC-SHA256)")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to PEM file with RSA private key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "URL to POST audit events")
	flag.IntVar(&cfg.Database.MaxOpenConns, "db-max-open-conns", dbCfg.MaxOpenConns, "database max open connections")
	flag.IntVar(&cfg.Database.MaxIdleConns, "db-max-idle-conns", dbCfg.MaxIdleConns, "database max idle connections")
	flag.DurationVar(&cfg.Database.ConnMaxIdleTime, "db-conn-max-idle-time", dbCfg.ConnMaxIdleTime, "database connection max idle time")
	flag.DurationVar(&cfg.Database.ConnMaxLifetime, "db-conn-max-lifetime", dbCfg.ConnMaxLifetime, "database connection max lifetime")
	flag.DurationVar(&cfg.Database.PingTimeout, "db-ping-timeout", dbCfg.PingTimeout, "database ping timeout on startup")
	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	return &cfg
}
