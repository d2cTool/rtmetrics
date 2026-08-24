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
}

// Load читает JSON-файл (если задан), затем флаги, затем перекрывает их окружением.
func Load() (*AgentConfig, error) {
	return Parse(os.Args[1:])
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

	var configPath string
	flags := common.NewBindings()
	flags.String(fs, "a", cfg.Address, "server address", &cfg.Address)
	flags.Int(fs, "r", cfg.ReportInterval, "report interval", &cfg.ReportInterval)
	flags.Int(fs, "p", cfg.PollInterval, "poll interval", &cfg.PollInterval)
	flags.String(fs, "k", cfg.Key, "key for request signing (HMAC-SHA256)", &cfg.Key)
	flags.Int(fs, "l", cfg.RateLimit, "max number of simultaneous outgoing requests", &cfg.RateLimit)
	flags.String(fs, "crypto-key", cfg.CryptoKey, "path to PEM file with RSA public key", &cfg.CryptoKey)
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

	return cfg, nil
}
