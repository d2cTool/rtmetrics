package agent

import (
	"reflect"
	"testing"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGaugesCoversAllFields(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(GaugeMetrics{})
	fields := make([]string, 0, typ.NumField())
	for i := range typ.NumField() {
		fields = append(fields, typ.Field(i).Name)
	}

	names := make([]string, 0, len(fields))
	for _, metric := range (GaugeMetrics{}).Gauges() {
		names = append(names, metric.ID)
	}

	assert.ElementsMatch(t, fields, names)
}

func TestGaugesValues(t *testing.T) {
	t.Parallel()

	gauges := (GaugeMetrics{Alloc: 1.5, RandomValue: 0.25}).Gauges()

	byName := make(map[string]float64, len(gauges))
	for _, metric := range gauges {
		require.Equal(t, m.Gauge, metric.MType)
		require.NotNil(t, metric.Value, "gauge %s без значения", metric.ID)
		byName[metric.ID] = *metric.Value
	}

	assert.InDelta(t, 1.5, byName["Alloc"], 0.0001)
	assert.InDelta(t, 0.25, byName["RandomValue"], 0.0001)
	assert.Zero(t, byName["Sys"])
}

func TestBuildBatchAppendsPollCount(t *testing.T) {
	t.Parallel()

	batch := BuildBatch(GaugeMetrics{Alloc: 1.5}, CountMetrics{PollCount: 7})

	require.Len(t, batch, reflect.TypeOf(GaugeMetrics{}).NumField()+1)

	last := batch[len(batch)-1]
	assert.Equal(t, "PollCount", last.ID)
	assert.Equal(t, m.Counter, last.MType)
	require.NotNil(t, last.Delta)
	assert.Equal(t, int64(7), *last.Delta)
}
