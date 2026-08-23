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
func Load() *ServerConfig {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cfg, err := parse(fs, os.Args[1:])
	if err != nil {
		panic(err)
	}
	return cfg
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

	var (
		address         = cfg.HTTPServer.Address
		storeInterval   = cfg.StoreInterval
		fileStoragePath = cfg.FileStoragePath
		restore         = cfg.Restore
		databaseDSN     = cfg.DatabaseDSN
		key             = cfg.Key
		cryptoKey       = cfg.CryptoKey
		auditFile       = cfg.AuditFile
		auditURL        = cfg.AuditURL
		trustedSubnet   = cfg.TrustedSubnet
		grpcAddress     = cfg.GRPCAddress
		maxOpenConns    = cfg.Database.MaxOpenConns
		maxIdleConns    = cfg.Database.MaxIdleConns
		connMaxIdleTime = cfg.Database.ConnMaxIdleTime
		connMaxLifetime = cfg.Database.ConnMaxLifetime
		pingTimeout     = cfg.Database.PingTimeout
		configPath      string
	)

	fs.StringVar(&address, "a", address, "server address")
	fs.IntVar(&storeInterval, "i", storeInterval, "store interval")
	fs.StringVar(&fileStoragePath, "f", fileStoragePath, "file storage path")
	fs.BoolVar(&restore, "r", restore, "restore")
	fs.StringVar(&databaseDSN, "d", databaseDSN, "database DSN (PostgreSQL)")
	fs.StringVar(&key, "k", key, "key for request signing (HMAC-SHA256)")
	fs.StringVar(&cryptoKey, "crypto-key", cryptoKey, "path to PEM file with RSA private key")
	fs.StringVar(&auditFile, "audit-file", auditFile, "path to audit log file")
	fs.StringVar(&auditURL, "audit-url", auditURL, "URL to POST audit events")
	fs.StringVar(&trustedSubnet, "t", trustedSubnet, "trusted subnet in CIDR notation")
	fs.StringVar(&grpcAddress, "g", grpcAddress, "gRPC server address")
	fs.IntVar(&maxOpenConns, "db-max-open-conns", maxOpenConns, "database max open connections")
	fs.IntVar(&maxIdleConns, "db-max-idle-conns", maxIdleConns, "database max idle connections")
	fs.DurationVar(&connMaxIdleTime, "db-conn-max-idle-time", connMaxIdleTime, "database connection max idle time")
	fs.DurationVar(&connMaxLifetime, "db-conn-max-lifetime", connMaxLifetime, "database connection max lifetime")
	fs.DurationVar(&pingTimeout, "db-ping-timeout", pingTimeout, "database ping timeout on startup")
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

	visited := common.VisitedFlags(fs)
	if common.FlagPassed(visited, "a") {
		cfg.HTTPServer.Address = address
	}
	if common.FlagPassed(visited, "i") {
		cfg.StoreInterval = storeInterval
	}
	if common.FlagPassed(visited, "f") {
		cfg.FileStoragePath = fileStoragePath
	}
	if common.FlagPassed(visited, "r") {
		cfg.Restore = restore
	}
	if common.FlagPassed(visited, "d") {
		cfg.DatabaseDSN = databaseDSN
	}
	if common.FlagPassed(visited, "k") {
		cfg.Key = key
	}
	if common.FlagPassed(visited, "crypto-key") {
		cfg.CryptoKey = cryptoKey
	}
	if common.FlagPassed(visited, "audit-file") {
		cfg.AuditFile = auditFile
	}
	if common.FlagPassed(visited, "audit-url") {
		cfg.AuditURL = auditURL
	}
	if common.FlagPassed(visited, "t") {
		cfg.TrustedSubnet = trustedSubnet
	}
	if common.FlagPassed(visited, "g") {
		cfg.GRPCAddress = grpcAddress
	}
	if common.FlagPassed(visited, "db-max-open-conns") {
		cfg.Database.MaxOpenConns = maxOpenConns
	}
	if common.FlagPassed(visited, "db-max-idle-conns") {
		cfg.Database.MaxIdleConns = maxIdleConns
	}
	if common.FlagPassed(visited, "db-conn-max-idle-time") {
		cfg.Database.ConnMaxIdleTime = connMaxIdleTime
	}
	if common.FlagPassed(visited, "db-conn-max-lifetime") {
		cfg.Database.ConnMaxLifetime = connMaxLifetime
	}
	if common.FlagPassed(visited, "db-ping-timeout") {
		cfg.Database.PingTimeout = pingTimeout
	}

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
