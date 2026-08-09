package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/pressly/goose/v3"
)

// Проверка, что *Storage реализует repository.MetricsRepository.
var _ repository.MetricsRepository = (*Storage)(nil)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Storage — реализация repository.MetricsRepository поверх PostgreSQL.
type Storage struct {
	db *sql.DB
}

// New создаёт хранилище и применяет миграции схемы БД.
func New(ctx context.Context, db *sql.DB) (*Storage, error) {
	if err := migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return &Storage{db: db}, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, "migrations")
}

func (s *Storage) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	const query = `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype)
		DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
		RETURNING delta`

	var result int64
	if err := s.db.QueryRowContext(ctx, query, name, metrics.Counter, value).Scan(&result); err != nil {
		return 0, fmt.Errorf("save counter %q: %w", name, err)
	}
	return result, nil
}

func (s *Storage) SaveGauge(ctx context.Context, name string, value float64) (float64, error) {
	const query = `
		INSERT INTO metrics (id, mtype, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype)
		DO UPDATE SET value = EXCLUDED.value
		RETURNING value`

	var result float64
	if err := s.db.QueryRowContext(ctx, query, name, metrics.Gauge, value).Scan(&result); err != nil {
		return 0, fmt.Errorf("save gauge %q: %w", name, err)
	}
	return result, nil
}

func (s *Storage) SaveBatch(ctx context.Context, batch []metrics.Metrics) error {
	const counterQuery = `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype)
		DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`

	const gaugeQuery = `
		INSERT INTO metrics (id, mtype, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype)
		DO UPDATE SET value = EXCLUDED.value`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	counterStmt, err := tx.PrepareContext(ctx, counterQuery)
	if err != nil {
		return fmt.Errorf("prepare counter stmt: %w", err)
	}
	defer counterStmt.Close()

	gaugeStmt, err := tx.PrepareContext(ctx, gaugeQuery)
	if err != nil {
		return fmt.Errorf("prepare gauge stmt: %w", err)
	}
	defer gaugeStmt.Close()

	for _, m := range batch {
		switch m.MType {
		case metrics.Counter:
			if m.Delta == nil {
				continue
			}
			if _, err := counterStmt.ExecContext(ctx, m.ID, metrics.Counter, *m.Delta); err != nil {
				return fmt.Errorf("save counter %q in batch: %w", m.ID, err)
			}
		case metrics.Gauge:
			if m.Value == nil {
				continue
			}
			if _, err := gaugeStmt.ExecContext(ctx, m.ID, metrics.Gauge, *m.Value); err != nil {
				return fmt.Errorf("save gauge %q in batch: %w", m.ID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *Storage) GetCounter(ctx context.Context, name string) (int64, error) {
	const query = `SELECT delta FROM metrics WHERE id = $1 AND mtype = $2`

	var value int64
	err := s.db.QueryRowContext(ctx, query, name, metrics.Counter).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("counter %q not found: %w", name, err)
	}
	if err != nil {
		return 0, fmt.Errorf("get counter %q: %w", name, err)
	}
	return value, nil
}

func (s *Storage) GetGauge(ctx context.Context, name string) (float64, error) {
	const query = `SELECT value FROM metrics WHERE id = $1 AND mtype = $2`

	var value float64
	err := s.db.QueryRowContext(ctx, query, name, metrics.Gauge).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("gauge %q not found: %w", name, err)
	}
	if err != nil {
		return 0, fmt.Errorf("get gauge %q: %w", name, err)
	}
	return value, nil
}

func (s *Storage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	const query = `SELECT id, delta FROM metrics WHERE mtype = $1`

	rows, err := s.db.QueryContext(ctx, query, metrics.Counter)
	if err != nil {
		return nil, fmt.Errorf("get all counters: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var (
			id    string
			delta int64
		)
		if err := rows.Scan(&id, &delta); err != nil {
			return nil, fmt.Errorf("scan counter row: %w", err)
		}
		result[id] = delta
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counter rows: %w", err)
	}
	return result, nil
}

func (s *Storage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	const query = `SELECT id, value FROM metrics WHERE mtype = $1`

	rows, err := s.db.QueryContext(ctx, query, metrics.Gauge)
	if err != nil {
		return nil, fmt.Errorf("get all gauges: %w", err)
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var (
			id    string
			value float64
		)
		if err := rows.Scan(&id, &value); err != nil {
			return nil, fmt.Errorf("scan gauge row: %w", err)
		}
		result[id] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gauge rows: %w", err)
	}
	return result, nil
}
