// Package config загружает конфигурацию агента из файла, флагов и окружения.
package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/d2cTool/rtmetrics/internal/config/common"
)

// AgentConfig — флаги, переменные окружения и JSON-файл процесса агента.
type AgentConfig struct {
	Env            string
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
	GRPCAddress    string `env:"GRPC_ADDRESS"`
}

// Load читает JSON-файл (если задан), затем флаги, затем перекрывает их окружением.
func Load() *AgentConfig {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cfg, err := parse(fs, os.Args[1:])
	if err != nil {
		panic(err)
	}
	return cfg
}

// Parse собирает AgentConfig из args и окружения.
// Приоритет: дефолты < JSON-файл < флаги < переменные окружения.
func Parse(args []string) (*AgentConfig, error) {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	return parse(fs, args)
}

func defaults() *AgentConfig {
	return &AgentConfig{Env: common.EnvLocal, Address: "localhost:8080", ReportInterval: 10, PollInterval: 2, RateLimit: 1}
}

func parse(fs *flag.FlagSet, args []string) (*AgentConfig, error) {
	cfg := defaults()

	var (
		address        = cfg.Address
		reportInterval = cfg.ReportInterval
		pollInterval   = cfg.PollInterval
		key            = cfg.Key
		rateLimit      = cfg.RateLimit
		cryptoKey      = cfg.CryptoKey
		grpcAddress    = cfg.GRPCAddress
		configPath     string
	)

	fs.StringVar(&address, "a", address, "server address")
	fs.IntVar(&reportInterval, "r", reportInterval, "report interval")
	fs.IntVar(&pollInterval, "p", pollInterval, "poll interval")
	fs.StringVar(&key, "k", key, "key for request signing (HMAC-SHA256)")
	fs.IntVar(&rateLimit, "l", rateLimit, "max number of simultaneous outgoing requests")
	fs.StringVar(&cryptoKey, "crypto-key", cryptoKey, "path to PEM file with RSA public key")
	fs.StringVar(&grpcAddress, "g", grpcAddress, "gRPC server address")
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
		cfg.Address = address
	}
	if common.FlagPassed(visited, "r") {
		cfg.ReportInterval = reportInterval
	}
	if common.FlagPassed(visited, "p") {
		cfg.PollInterval = pollInterval
	}
	if common.FlagPassed(visited, "k") {
		cfg.Key = key
	}
	if common.FlagPassed(visited, "l") {
		cfg.RateLimit = rateLimit
	}
	if common.FlagPassed(visited, "crypto-key") {
		cfg.CryptoKey = cryptoKey
	}
	if common.FlagPassed(visited, "g") {
		cfg.GRPCAddress = grpcAddress
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
