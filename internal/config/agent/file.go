package config

import "github.com/d2cTool/rtmetrics/internal/config/common"

// fileConfig — JSON-файл агента. Указатели отличают «поле задано» от нуля.
type fileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
	GRPCAddress    *string `json:"grpc_address"`
}

func applyFile(cfg *AgentConfig, path string) error {
	var file fileConfig
	if err := common.LoadJSON(path, &file); err != nil {
		return err
	}
	return overlayFile(cfg, &file)
}

func overlayFile(cfg *AgentConfig, file *fileConfig) error {
	if file.Address != nil {
		cfg.Address = *file.Address
	}
	if file.ReportInterval != nil {
		sec, err := common.ParseIntervalSeconds(*file.ReportInterval)
		if err != nil {
			return err
		}
		cfg.ReportInterval = sec
	}
	if file.PollInterval != nil {
		sec, err := common.ParseIntervalSeconds(*file.PollInterval)
		if err != nil {
			return err
		}
		cfg.PollInterval = sec
	}
	if file.CryptoKey != nil {
		cfg.CryptoKey = *file.CryptoKey
	}
	if file.Key != nil {
		cfg.Key = *file.Key
	}
	if file.RateLimit != nil {
		cfg.RateLimit = *file.RateLimit
	}
	if file.GRPCAddress != nil {
		cfg.GRPCAddress = *file.GRPCAddress
	}
	return nil
}
