// Package postgres хранит метрики в PostgreSQL и накатывает embed-миграции.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/d2cTool/rtmetrics/internal/repository"
	"github.com/d2cTool/rtmetrics/internal/retry"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
)

var _ repository.MetricsRepository = (*Storage)(nil)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Storage — PostgreSQL-реализация repository.MetricsRepository.
type Storage struct {
	db *sql.DB
}

// New применяет миграции и возвращает хранилище над db.
func New(ctx context.Context, db *sql.DB) (*Storage, error) {
	if err := migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	return &Storage{db: db}, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply up migrations: %w", err)
	}
	return nil
}

func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func (s *Storage) SaveCounter(ctx context.Context, name string, value int64) (int64, error) {
	const query = `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, $2, $3)
		ON CONFLICT (id, mtype)
		DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
		RETURNING delta`

	var result int64
	err := retry.Do(ctx, isRetriable, func() error {
		return s.db.QueryRowContext(ctx, query, name, metrics.Counter, value).Scan(&result)
	})
	if err != nil {
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
	err := retry.Do(ctx, isRetriable, func() error {
		return s.db.QueryRowContext(ctx, query, name, metrics.Gauge, value).Scan(&result)
	})
	if err != nil {
		return 0, fmt.Errorf("save gauge %q: %w", name, err)
	}
	return result, nil
}

func (s *Storage) SaveBatch(ctx context.Context, batch []metrics.Metrics) error {
	return retry.Do(ctx, isRetriable, func() error {
		return s.saveBatch(ctx, batch)
	})
}

func (s *Storage) saveBatch(ctx context.Context, batch []metrics.Metrics) error {
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
	err := retry.Do(ctx, isRetriable, func() error {
		return s.db.QueryRowContext(ctx, query, name, metrics.Counter).Scan(&value)
	})
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
	err := retry.Do(ctx, isRetriable, func() error {
		return s.db.QueryRowContext(ctx, query, name, metrics.Gauge).Scan(&value)
	})
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

	var result map[string]int64
	err := retry.Do(ctx, isRetriable, func() error {
		rows, err := s.db.QueryContext(ctx, query, metrics.Counter)
		if err != nil {
			return err
		}
		defer rows.Close()

		counters := make(map[string]int64)
		for rows.Next() {
			var (
				id    string
				delta int64
			)
			if err := rows.Scan(&id, &delta); err != nil {
				return err
			}
			counters[id] = delta
		}
		if err := rows.Err(); err != nil {
			return err
		}

		result = counters
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get all counters: %w", err)
	}
	return result, nil
}

func (s *Storage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	const query = `SELECT id, value FROM metrics WHERE mtype = $1`

	var result map[string]float64
	err := retry.Do(ctx, isRetriable, func() error {
		rows, err := s.db.QueryContext(ctx, query, metrics.Gauge)
		if err != nil {
			return err
		}
		defer rows.Close()

		gauges := make(map[string]float64)
		for rows.Next() {
			var (
				id    string
				value float64
			)
			if err := rows.Scan(&id, &value); err != nil {
				return err
			}
			gauges[id] = value
		}
		if err := rows.Err(); err != nil {
			return err
		}

		result = gauges
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get all gauges: %w", err)
	}
	return result, nil
}
