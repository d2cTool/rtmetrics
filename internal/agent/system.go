package agent

import (
	"context"
	"fmt"

	m "github.com/d2cTool/rtmetrics/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemMetrics struct {
	TotalMemory    float64
	FreeMemory     float64
	CPUUtilization []float64
}

func CollectSystem(ctx context.Context) (SystemMetrics, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return SystemMetrics{}, fmt.Errorf("failed to read virtual memory stats: %w", err)
	}

	utilization, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return SystemMetrics{}, fmt.Errorf("failed to read cpu utilization: %w", err)
	}

	return SystemMetrics{
		TotalMemory:    float64(vm.Total),
		FreeMemory:     float64(vm.Free),
		CPUUtilization: utilization,
	}, nil
}

func (s SystemMetrics) Gauges() []m.Metrics {
	metrics := make([]m.Metrics, 0, len(s.CPUUtilization)+2)
	metrics = append(metrics,
		*m.NewGauge("TotalMemory", s.TotalMemory),
		*m.NewGauge("FreeMemory", s.FreeMemory),
	)

	for i, value := range s.CPUUtilization {
		metrics = append(metrics, *m.NewGauge(fmt.Sprintf("CPUutilization%d", i+1), value))
	}

	return metrics
}
