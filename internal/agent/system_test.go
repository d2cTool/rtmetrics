package agent

import (
	"context"
	"runtime"
	"testing"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemGaugesNaming(t *testing.T) {
	t.Parallel()

	system := SystemMetrics{
		TotalMemory:    1024,
		FreeMemory:     512,
		CPUUtilization: []float64{1.5, 2.5, 3.5},
	}

	byName := make(map[string]float64)
	for _, metric := range system.Gauges() {
		require.Equal(t, m.Gauge, metric.MType)
		require.NotNil(t, metric.Value, "gauge %s без значения", metric.ID)
		byName[metric.ID] = *metric.Value
	}

	require.Len(t, byName, 5)
	assert.InDelta(t, 1024, byName["TotalMemory"], 0.0001)
	assert.InDelta(t, 512, byName["FreeMemory"], 0.0001)
	assert.InDelta(t, 1.5, byName["CPUutilization1"], 0.0001)
	assert.InDelta(t, 2.5, byName["CPUutilization2"], 0.0001)
	assert.InDelta(t, 3.5, byName["CPUutilization3"], 0.0001)
}

func TestCollectSystemReturnsPerCPUMetrics(t *testing.T) {
	system, err := CollectSystem(context.Background())
	require.NoError(t, err)

	assert.Positive(t, system.TotalMemory)
	assert.NotEmpty(t, system.CPUUtilization, "ожидается хотя бы одно ядро при %d CPU", runtime.NumCPU())
}
