package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
