package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Snapshot — снимок метрик для сохранения на диск.
type Snapshot struct {
	Counters map[string]int64   `json:"counters"`
	Gauges   map[string]float64 `json:"gauges"`
}

// Save записывает снимок в файл. Использует временный файл и переименование для атомарности.
func Save(path string, counters map[string]int64, gauges map[string]float64) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(Snapshot{Counters: counters, Gauges: gauges}); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		// На Windows Rename не перезаписывает существующий файл
		_ = os.Remove(path)
		return os.Rename(tmpPath, path)
	}
	return nil
}

// Load читает снимок из файла. Если файл не существует, возвращает nil maps без ошибки.
func Load(path string) (counters map[string]int64, gauges map[string]float64, err error) {
	counters = make(map[string]int64)
	gauges = make(map[string]float64)

	if path == "" {
		return counters, gauges, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return counters, gauges, nil
		}
		return nil, nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, nil, err
	}
	if snap.Counters != nil {
		counters = snap.Counters
	}
	if snap.Gauges != nil {
		gauges = snap.Gauges
	}
	return counters, gauges, nil
}
