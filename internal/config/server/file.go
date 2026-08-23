package config

import (
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
	if file.Address != nil {
		cfg.HTTPServer.Address = *file.Address
	}
	if file.Restore != nil {
		cfg.Restore = *file.Restore
	}
	if file.StoreInterval != nil {
		sec, err := common.ParseIntervalSeconds(*file.StoreInterval)
		if err != nil {
			return err
		}
		cfg.StoreInterval = sec
	}
	if file.StoreFile != nil {
		cfg.FileStoragePath = *file.StoreFile
	}
	if file.DatabaseDSN != nil {
		cfg.DatabaseDSN = *file.DatabaseDSN
	}
	if file.CryptoKey != nil {
		cfg.CryptoKey = *file.CryptoKey
	}
	if file.Key != nil {
		cfg.Key = *file.Key
	}
	if file.AuditFile != nil {
		cfg.AuditFile = *file.AuditFile
	}
	if file.AuditURL != nil {
		cfg.AuditURL = *file.AuditURL
	}
	if file.DatabaseMaxOpenConns != nil {
		cfg.Database.MaxOpenConns = *file.DatabaseMaxOpenConns
	}
	if file.DatabaseMaxIdleConns != nil {
		cfg.Database.MaxIdleConns = *file.DatabaseMaxIdleConns
	}
	if file.DatabaseConnMaxIdleTime != nil {
		d, err := time.ParseDuration(*file.DatabaseConnMaxIdleTime)
		if err != nil {
			return err
		}
		cfg.Database.ConnMaxIdleTime = d
	}
	if file.DatabaseConnMaxLifetime != nil {
		d, err := time.ParseDuration(*file.DatabaseConnMaxLifetime)
		if err != nil {
			return err
		}
		cfg.Database.ConnMaxLifetime = d
	}
	if file.DatabasePingTimeout != nil {
		d, err := time.ParseDuration(*file.DatabasePingTimeout)
		if err != nil {
			return err
		}
		cfg.Database.PingTimeout = d
	}
	return nil
}
