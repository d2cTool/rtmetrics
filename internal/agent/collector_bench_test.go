package agent

import (
	"testing"
)

func BenchmarkCollect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Collect()
	}
}

func BenchmarkGauges(b *testing.B) {
	gauges := Collect()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = gauges.Gauges()
	}
}

func BenchmarkBuildBatch(b *testing.B) {
	gauges := Collect()
	counters := CountMetrics{PollCount: 42}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = BuildBatch(gauges, counters)
	}
}
