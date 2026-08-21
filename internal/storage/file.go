package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	metrics "github.com/d2cTool/rtmetrics/internal/model"
)

// Save атомарно пишет counters и gauges в path (через временный файл).
func Save(ctx context.Context, path string, counters map[string]int64, gauges map[string]float64) error {
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

	items := make([]*metrics.Metrics, 0, len(counters)+len(gauges))
	for id, v := range counters {
		items = append(items, metrics.NewCounter(id, v))
	}
	for id, v := range gauges {
		items = append(items, metrics.NewGauge(id, v))
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
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
		_ = os.Remove(path)
		return os.Rename(tmpPath, path)
	}
	return nil
}

// Load читает снимок из path. Пустой или отсутствующий файл даёт пустые карты.
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
	var items []*metrics.Metrics
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil, err
	}
	for _, m := range items {
		if m == nil {
			continue
		}
		switch m.MType {
		case metrics.Counter:
			if m.Delta != nil {
				counters[m.ID] = *m.Delta
			}
		case metrics.Gauge:
			if m.Value != nil {
				gauges[m.ID] = *m.Value
			}
		}
	}
	return counters, gauges, nil
}
