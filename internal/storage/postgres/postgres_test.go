package postgres

import (
	"context"
	"io/fs"
	"testing"

	"errors"

	"github.com/DATA-DOG/go-sqlmock"
	metrics "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetriable(t *testing.T) {
	assert.False(t, isRetriable(nil))
	assert.False(t, isRetriable(errors.New("plain error")))

	// Класс 08 — Connection Exception → retriable.
	connErr := &pgconn.PgError{Code: pgerrcode.ConnectionException}
	assert.True(t, isRetriable(connErr))
	connFailure := &pgconn.PgError{Code: pgerrcode.ConnectionFailure}
	assert.True(t, isRetriable(connFailure))

	// Прочие SQLSTATE (например, unique_violation) → не retriable.
	uniqueErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	assert.False(t, isRetriable(uniqueErr))
}

func TestMigrationsAreDiscovered(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	require.NoError(t, err)
	assert.NotEmpty(t, provider.ListSources())
}

func newTestStorage(t *testing.T) (*Storage, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return &Storage{db: db}, mock
}

func TestStorage_SaveCounter(t *testing.T) {
	s, mock := newTestStorage(t)

	mock.ExpectQuery("INSERT INTO metrics").
		WithArgs("PollCount", "counter", int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"delta"}).AddRow(int64(15)))

	got, err := s.SaveCounter(context.Background(), "PollCount", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(15), got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorage_SaveGauge(t *testing.T) {
	s, mock := newTestStorage(t)

	mock.ExpectQuery("INSERT INTO metrics").
		WithArgs("Alloc", "gauge", 3.14).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(3.14))

	got, err := s.SaveGauge(context.Background(), "Alloc", 3.14)
	require.NoError(t, err)
	assert.InDelta(t, 3.14, got, 0.0001)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorage_GetCounter(t *testing.T) {
	s, mock := newTestStorage(t)

	mock.ExpectQuery("SELECT delta FROM metrics").
		WithArgs("PollCount", "counter").
		WillReturnRows(sqlmock.NewRows([]string{"delta"}).AddRow(int64(42)))

	got, err := s.GetCounter(context.Background(), "PollCount")
	require.NoError(t, err)
	assert.Equal(t, int64(42), got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorage_GetCounter_NotFound(t *testing.T) {
	s, mock := newTestStorage(t)

	mock.ExpectQuery("SELECT delta FROM metrics").
		WithArgs("missing", "counter").
		WillReturnError(sqlmock.ErrCancelled) // любая ошибка запроса

	_, err := s.GetCounter(context.Background(), "missing")
	require.Error(t, err)
}

func TestStorage_SaveBatch(t *testing.T) {
	s, mock := newTestStorage(t)

	delta := int64(5)
	value := 3.14
	batch := []metrics.Metrics{
		{ID: "PollCount", MType: metrics.Counter, Delta: &delta},
		{ID: "Alloc", MType: metrics.Gauge, Value: &value},
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO metrics")
	mock.ExpectPrepare("INSERT INTO metrics")
	mock.ExpectExec("INSERT INTO metrics").
		WithArgs("PollCount", "counter", delta).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO metrics").
		WithArgs("Alloc", "gauge", value).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.SaveBatch(context.Background(), batch)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorage_GetAllGauges(t *testing.T) {
	s, mock := newTestStorage(t)

	rows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow("Alloc", 1.5).
		AddRow("Sys", 2.5)
	mock.ExpectQuery("SELECT id, value FROM metrics").
		WithArgs("gauge").
		WillReturnRows(rows)

	got, err := s.GetAllGauges(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]float64{"Alloc": 1.5, "Sys": 2.5}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStorage_GetAllGauges_RetryDropsPartialRows(t *testing.T) {
	s, mock := newTestStorage(t)

	failing := sqlmock.NewRows([]string{"id", "value"}).
		AddRow("Stale", 9.9).
		AddRow("Alloc", 1.5).
		RowError(1, &pgconn.PgError{Code: pgerrcode.ConnectionFailure})
	mock.ExpectQuery("SELECT id, value FROM metrics").
		WithArgs("gauge").
		WillReturnRows(failing)

	mock.ExpectQuery("SELECT id, value FROM metrics").
		WithArgs("gauge").
		WillReturnRows(sqlmock.NewRows([]string{"id", "value"}).AddRow("Alloc", 1.5))

	got, err := s.GetAllGauges(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]float64{"Alloc": 1.5}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}
