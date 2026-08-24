package config

import (
	"errors"

	"github.com/d2cTool/rtmetrics/internal/config/common"
)

// fileConfig — JSON-файл агента. Указатели отличают «поле задано» от нуля.
type fileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
}

func applyFile(cfg *AgentConfig, path string) error {
	var file fileConfig
	if err := common.LoadJSON(path, &file); err != nil {
		return err
	}
	return overlayFile(cfg, &file)
}

func overlayFile(cfg *AgentConfig, file *fileConfig) error {
	common.Assign(&cfg.Address, file.Address)
	common.Assign(&cfg.CryptoKey, file.CryptoKey)
	common.Assign(&cfg.Key, file.Key)
	common.Assign(&cfg.RateLimit, file.RateLimit)
	return errors.Join(
		common.AssignFunc(&cfg.ReportInterval, file.ReportInterval, common.ParseIntervalSeconds),
		common.AssignFunc(&cfg.PollInterval, file.PollInterval, common.ParseIntervalSeconds),
	)
}
