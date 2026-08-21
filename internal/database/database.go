// Package database открывает пул PostgreSQL (pgx stdlib) и проверяет его ping-ом.
package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config задаёт параметры пула соединений и таймаут проверки доступности БД.
type Config struct {
	MaxOpenConns    int           `env:"DATABASE_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `env:"DATABASE_MAX_IDLE_CONNS"`
	ConnMaxIdleTime time.Duration `env:"DATABASE_CONN_MAX_IDLE_TIME"`
	ConnMaxLifetime time.Duration `env:"DATABASE_CONN_MAX_LIFETIME"`
	PingTimeout     time.Duration `env:"DATABASE_PING_TIMEOUT"`
}

// DefaultConfig возвращает размеры пула и таймауты по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxIdleTime: time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
		PingTimeout:     5 * time.Second,
	}
}

func (c *Config) normalized() Config {
	def := DefaultConfig()
	if c == nil {
		return *def
	}

	cfg := *c
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = def.MaxOpenConns
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = def.MaxIdleConns
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
	}
	if cfg.ConnMaxIdleTime < 0 {
		cfg.ConnMaxIdleTime = 0
	}
	if cfg.ConnMaxLifetime < 0 {
		cfg.ConnMaxLifetime = 0
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = def.PingTimeout
	}

	return cfg
}

// New открывает пул по dsn, применяет cfg и проверяет соединение ping-ом.
func New(ctx context.Context, dsn string, cfg *Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	poolCfg := cfg.normalized()
	db.SetMaxOpenConns(poolCfg.MaxOpenConns)
	db.SetMaxIdleConns(poolCfg.MaxIdleConns)
	db.SetConnMaxIdleTime(poolCfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, poolCfg.PingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
