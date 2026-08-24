package config

import (
	"errors"
	"time"

	"github.com/d2cTool/rtmetrics/internal/config/common"
)

// fileConfig — JSON-файл сервера. Указатели отличают «поле задано» от нуля.
type fileConfig struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	Key           *string `json:"key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`

	DatabaseMaxOpenConns    *int    `json:"database_max_open_conns"`
	DatabaseMaxIdleConns    *int    `json:"database_max_idle_conns"`
	DatabaseConnMaxIdleTime *string `json:"database_conn_max_idle_time"`
	DatabaseConnMaxLifetime *string `json:"database_conn_max_lifetime"`
	DatabasePingTimeout     *string `json:"database_ping_timeout"`
}

func applyFile(cfg *ServerConfig, path string) error {
	var file fileConfig
	if err := common.LoadJSON(path, &file); err != nil {
		return err
	}
	return overlayFile(cfg, &file)
}

func overlayFile(cfg *ServerConfig, file *fileConfig) error {
	common.Assign(&cfg.HTTPServer.Address, file.Address)
	common.Assign(&cfg.Restore, file.Restore)
	common.Assign(&cfg.FileStoragePath, file.StoreFile)
	common.Assign(&cfg.DatabaseDSN, file.DatabaseDSN)
	common.Assign(&cfg.CryptoKey, file.CryptoKey)
	common.Assign(&cfg.Key, file.Key)
	common.Assign(&cfg.AuditFile, file.AuditFile)
	common.Assign(&cfg.AuditURL, file.AuditURL)
	common.Assign(&cfg.Database.MaxOpenConns, file.DatabaseMaxOpenConns)
	common.Assign(&cfg.Database.MaxIdleConns, file.DatabaseMaxIdleConns)
	return errors.Join(
		common.AssignFunc(&cfg.StoreInterval, file.StoreInterval, common.ParseIntervalSeconds),
		common.AssignFunc(&cfg.Database.ConnMaxIdleTime, file.DatabaseConnMaxIdleTime, time.ParseDuration),
		common.AssignFunc(&cfg.Database.ConnMaxLifetime, file.DatabaseConnMaxLifetime, time.ParseDuration),
		common.AssignFunc(&cfg.Database.PingTimeout, file.DatabasePingTimeout, time.ParseDuration),
	)
}
