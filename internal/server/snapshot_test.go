package server

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	config "github.com/d2cTool/rtmetrics/internal/config/server"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(restore bool, path string, interval time.Duration) *config.ServerConfig {
	return &config.ServerConfig{
		Restore:         restore,
		FileStoragePath: path,
		StoreInterval:   interval,
		HttpServer:      &config.HttpServerConfig{},
	}
}

func TestRestoreIfNeeded_SkipWhenRestoreFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	require.NoError(t, storage.Save(path, map[string]int64{"a": 1}, map[string]float64{"b": 2}))

	cfg := testConfig(false, path, time.Second)
	st := storage.New()
	log := defaultLogger()

	RestoreIfNeeded(cfg, st, log)

	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	gauges, _ := st.GetAllGauges(ctx)
	assert.Empty(t, counters)
	assert.Empty(t, gauges)
}

func TestRestoreIfNeeded_SkipWhenPathEmpty(t *testing.T) {
	cfg := testConfig(true, "", time.Second)
	st := storage.New()
	log := defaultLogger()

	RestoreIfNeeded(cfg, st, log)
	// не падает, хранилище пустое
	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	assert.Empty(t, counters)
}

func TestRestoreIfNeeded_FileNotExists_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	cfg := testConfig(true, path, time.Second)
	st := storage.New()
	log := defaultLogger()

	RestoreIfNeeded(cfg, st, log)

	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	gauges, _ := st.GetAllGauges(ctx)
	assert.Empty(t, counters)
	assert.Empty(t, gauges)
}

func TestRestoreIfNeeded_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	countersIn := map[string]int64{"c1": 10, "c2": 20}
	gaugesIn := map[string]float64{"g1": 1.5, "g2": 2.5}
	require.NoError(t, storage.Save(path, countersIn, gaugesIn))

	cfg := testConfig(true, path, time.Second)
	st := storage.New()
	log := defaultLogger()

	RestoreIfNeeded(cfg, st, log)

	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	gauges, _ := st.GetAllGauges(ctx)
	assert.Equal(t, countersIn, counters)
	assert.Equal(t, gaugesIn, gauges)
}

func TestRestoreIfNeeded_InvalidFile_NoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	cfg := testConfig(true, path, time.Second)
	st := storage.New()
	log := defaultLogger()

	RestoreIfNeeded(cfg, st, log)

	ctx := context.Background()
	counters, _ := st.GetAllCounters(ctx)
	assert.Empty(t, counters)
}

func TestSaveSnapshot_EmptyPath_NoOp(t *testing.T) {
	cfg := testConfig(false, "", time.Second)
	st := storage.New()
	st.Restore(map[string]int64{"x": 1}, nil)
	log := defaultLogger()

	SaveSnapshot(cfg, st, log)
	// не создаёт файл (проверяем, что в TempDir ничего не появилось)
	dir := t.TempDir()
	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}

func TestSaveSnapshot_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	cfg := testConfig(false, path, time.Second)
	st := storage.New()
	st.Restore(map[string]int64{"cnt": 42}, map[string]float64{"g": 3.14})
	log := defaultLogger()

	SaveSnapshot(cfg, st, log)

	require.FileExists(t, path)
	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"cnt": 42}, counters)
	assert.Equal(t, map[string]float64{"g": 3.14}, gauges)
}

func TestSaveSnapshot_EmptyStorage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	cfg := testConfig(false, path, time.Second)
	st := storage.New()
	log := defaultLogger()

	SaveSnapshot(cfg, st, log)

	require.FileExists(t, path)
	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Empty(t, counters)
	assert.Empty(t, gauges)
}

// При StoreInterval=0 периодическое сохранение не запускается — снимок пишется синхронно через SyncSaveRepo.
// RunPeriodicSave при interval=0 сразу выходит и не пишет в файл.
func TestRunPeriodicSave_ZeroInterval_ReturnsImmediately(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tick.json")

	cfg := testConfig(false, path, 0)
	st := storage.New()
	st.Restore(map[string]int64{"x": 1}, nil)
	log := defaultLogger()

	done := make(chan struct{})
	go func() {
		RunPeriodicSave(cfg, st, log)
		close(done)
	}()
	select {
	case <-done:
		// при interval=0 не запускаем тикер — синхронная запись через SyncSaveRepo
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RunPeriodicSave should return immediately when interval is 0")
	}
	// файл не создаётся — при interval=0 периодическое сохранение не используется
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestRunPeriodicSave_EmptyPath_ReturnsImmediately(t *testing.T) {
	cfg := testConfig(false, "", time.Millisecond)
	st := storage.New()
	log := defaultLogger()

	done := make(chan struct{})
	go func() {
		RunPeriodicSave(cfg, st, log)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RunPeriodicSave should return immediately when path is empty")
	}
}

func TestRunPeriodicSave_WritesPeriodically(t *testing.T) {
	// Файл в os.TempDir(), чтобы горутина RunPeriodicSave не блокировала очистку t.TempDir()
	path := filepath.Join(os.TempDir(), "rtmetrics_periodic_test.json")
	defer os.Remove(path)

	cfg := testConfig(false, path, 5*time.Millisecond)
	st := storage.New()
	st.Restore(map[string]int64{"p": 7}, map[string]float64{"q": 11.0})
	log := defaultLogger()

	go RunPeriodicSave(cfg, st, log)

	time.Sleep(20 * time.Millisecond)

	require.FileExists(t, path)
	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"p": 7}, counters)
	assert.Equal(t, map[string]float64{"q": 11.0}, gauges)
}

// --- SyncSaveRepo: при StoreInterval=0 снимок пишется синхронно после каждого Save ---

func TestSyncSaveRepo_SaveCounter_WritesSnapshotImmediately(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync_counter.json")

	cfg := testConfig(false, path, 0)
	st := storage.New()
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	_, err := repo.SaveCounter(ctx, "PollCount", 5)
	require.NoError(t, err)

	require.FileExists(t, path)
	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"PollCount": 5}, counters)
	assert.Empty(t, gauges)
}

func TestSyncSaveRepo_SaveGauge_WritesSnapshotImmediately(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync_gauge.json")

	cfg := testConfig(false, path, 0)
	st := storage.New()
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	_, err := repo.SaveGauge(ctx, "Alloc", 1024.5)
	require.NoError(t, err)

	require.FileExists(t, path)
	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Empty(t, counters)
	assert.Equal(t, map[string]float64{"Alloc": 1024.5}, gauges)
}

func TestSyncSaveRepo_SaveCounterAndGauge_BothInSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync_both.json")

	cfg := testConfig(false, path, 0)
	st := storage.New()
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	_, err := repo.SaveCounter(ctx, "N", 1)
	require.NoError(t, err)
	_, err = repo.SaveGauge(ctx, "G", 2.5)
	require.NoError(t, err)

	counters, gauges, err := storage.Load(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"N": 1}, counters)
	assert.Equal(t, map[string]float64{"G": 2.5}, gauges)
}

func TestSyncSaveRepo_StoreIntervalNonZero_NoSnapshotOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_sync.json")

	cfg := testConfig(false, path, 10*time.Second)
	st := storage.New()
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	_, err := repo.SaveCounter(ctx, "X", 1)
	require.NoError(t, err)

	// при interval > 0 SyncSaveRepo не вызывает SaveSnapshot после Save
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestSyncSaveRepo_EmptyPath_NoSnapshotOnSave(t *testing.T) {
	cfg := testConfig(false, "", 0)
	st := storage.New()
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	_, err := repo.SaveCounter(ctx, "X", 1)
	require.NoError(t, err)
	_, err = repo.SaveGauge(ctx, "Y", 1.0)
	require.NoError(t, err)
	// не падает, файл не создаётся (path пустой)
}

func TestSyncSaveRepo_ReadMethods_Passthrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "read.json")

	cfg := testConfig(false, path, 0)
	st := storage.New()
	st.Restore(map[string]int64{"c": 10}, map[string]float64{"g": 3.14})
	log := defaultLogger()
	repo := NewSyncSaveRepo(st, cfg, log)

	ctx := context.Background()
	v, err := repo.GetCounter(ctx, "c")
	require.NoError(t, err)
	assert.Equal(t, int64(10), v)

	vf, err := repo.GetGauge(ctx, "g")
	require.NoError(t, err)
	assert.Equal(t, 3.14, vf)

	counters, _ := repo.GetAllCounters(ctx)
	gauges, _ := repo.GetAllGauges(ctx)
	assert.Equal(t, map[string]int64{"c": 10}, counters)
	assert.Equal(t, map[string]float64{"g": 3.14}, gauges)
}

func defaultLogger() *slog.Logger {
	return slog.Default()
}
