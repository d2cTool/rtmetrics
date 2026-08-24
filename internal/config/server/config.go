// Package config загружает конфигурацию HTTP-сервера из файла, флагов и окружения.
package config

import (
	"flag"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
	"github.com/d2cTool/rtmetrics/internal/database"
)

// ServerConfig — флаги, переменные окружения и JSON-файл процесса сервера.
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
	TrustedSubnet   string `env:"TRUSTED_SUBNET"`
	GRPCAddress     string `env:"GRPC_ADDRESS"`
}

// HTTPServerConfig — адрес и таймауты http.Server.
type HTTPServerConfig struct {
	Address      string `env:"ADDRESS"`
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Load читает JSON-файл (если задан), затем флаги, затем перекрывает их окружением.
func Load() (*ServerConfig, error) {
	return Parse(os.Args[1:])
}

// Parse собирает ServerConfig из args и окружения.
// Приоритет: дефолты < JSON-файл < флаги < переменные окружения.
func Parse(args []string) (*ServerConfig, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	return parse(fs, args)
}

func defaults() *ServerConfig {
	httpSrv := HTTPServerConfig{Address: "localhost:8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	return &ServerConfig{
		Env:             common.EnvLocal,
		HTTPServer:      &httpSrv,
		Database:        database.DefaultConfig(),
		StoreInterval:   300,
		FileStoragePath: "./tmp/data",
		Restore:         false,
	}
}

func parse(fs *flag.FlagSet, args []string) (*ServerConfig, error) {
	cfg := defaults()

	var configPath string
	flags := common.NewBindings()
	flags.String(fs, "a", cfg.HTTPServer.Address, "server address", &cfg.HTTPServer.Address)
	flags.Int(fs, "i", cfg.StoreInterval, "store interval", &cfg.StoreInterval)
	flags.String(fs, "f", cfg.FileStoragePath, "file storage path", &cfg.FileStoragePath)
	flags.Bool(fs, "r", cfg.Restore, "restore", &cfg.Restore)
	flags.String(fs, "d", cfg.DatabaseDSN, "database DSN (PostgreSQL)", &cfg.DatabaseDSN)
	flags.String(fs, "k", cfg.Key, "key for request signing (HMAC-SHA256)", &cfg.Key)
	flags.String(fs, "crypto-key", cfg.CryptoKey, "path to PEM file with RSA private key", &cfg.CryptoKey)
	flags.String(fs, "audit-file", cfg.AuditFile, "path to audit log file", &cfg.AuditFile)
	flags.String(fs, "audit-url", cfg.AuditURL, "URL to POST audit events", &cfg.AuditURL)
	flags.String(fs, "t", cfg.TrustedSubnet, "trusted subnet in CIDR notation", &cfg.TrustedSubnet)
	flags.String(fs, "g", cfg.GRPCAddress, "gRPC server address", &cfg.GRPCAddress)
	flags.Int(fs, "db-max-open-conns", cfg.Database.MaxOpenConns, "database max open connections", &cfg.Database.MaxOpenConns)
	flags.Int(fs, "db-max-idle-conns", cfg.Database.MaxIdleConns, "database max idle connections", &cfg.Database.MaxIdleConns)
	flags.Duration(fs, "db-conn-max-idle-time", cfg.Database.ConnMaxIdleTime, "database connection max idle time", &cfg.Database.ConnMaxIdleTime)
	flags.Duration(fs, "db-conn-max-lifetime", cfg.Database.ConnMaxLifetime, "database connection max lifetime", &cfg.Database.ConnMaxLifetime)
	flags.Duration(fs, "db-ping-timeout", cfg.Database.PingTimeout, "database ping timeout on startup", &cfg.Database.PingTimeout)
	fs.StringVar(&configPath, "c", "", "path to JSON config file")
	fs.StringVar(&configPath, "config", "", "path to JSON config file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if path := common.ResolveConfigPath(configPath); path != "" {
		if err := applyFile(cfg, path); err != nil {
			return nil, err
		}
	}

	flags.Apply(fs)

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	applyStoreFileEnv(cfg)

	return cfg, nil
}

// applyStoreFileEnv принимает STORE_FILE, если FILE_STORAGE_PATH не задан.
func applyStoreFileEnv(cfg *ServerConfig) {
	if _, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		return
	}
	if v, ok := os.LookupEnv("STORE_FILE"); ok {
		cfg.FileStoragePath = v
	}
}
