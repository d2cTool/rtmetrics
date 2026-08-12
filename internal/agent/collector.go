package agent

import (
	"math/rand"
	"runtime"

	m "github.com/d2cTool/rtmetrics/internal/model"
)

type CountMetrics struct {
	PollCount int64
}

type GaugeMetrics struct {
	Alloc         float64
	BuckHashSys   float64
	Frees         float64
	GCCPUFraction float64
	GCSys         float64
	HeapAlloc     float64
	HeapIdle      float64
	HeapInuse     float64
	HeapObjects   float64
	HeapReleased  float64
	HeapSys       float64
	LastGC        float64
	Lookups       float64
	MCacheInuse   float64
	MCacheSys     float64
	MSpanInuse    float64
	MSpanSys      float64
	Mallocs       float64
	NextGC        float64
	NumForcedGC   float64
	NumGC         float64
	OtherSys      float64
	PauseTotalNs  float64
	StackInuse    float64
	StackSys      float64
	Sys           float64
	TotalAlloc    float64
	RandomValue   float64
}

func Collect() GaugeMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return GaugeMetrics{
		Alloc:         float64(m.Alloc),
		BuckHashSys:   float64(m.BuckHashSys),
		Frees:         float64(m.Frees),
		GCCPUFraction: float64(m.GCCPUFraction),
		GCSys:         float64(m.GCSys),
		HeapAlloc:     float64(m.HeapAlloc),
		HeapIdle:      float64(m.HeapIdle),
		HeapInuse:     float64(m.HeapInuse),
		HeapObjects:   float64(m.HeapObjects),
		HeapReleased:  float64(m.HeapReleased),
		HeapSys:       float64(m.HeapSys),
		LastGC:        float64(m.LastGC),
		Lookups:       float64(m.Lookups),
		MCacheInuse:   float64(m.MCacheInuse),
		MCacheSys:     float64(m.MCacheSys),
		MSpanInuse:    float64(m.MSpanInuse),
		MSpanSys:      float64(m.MSpanSys),
		Mallocs:       float64(m.Mallocs),
		NextGC:        float64(m.NextGC),
		NumForcedGC:   float64(m.NumForcedGC),
		NumGC:         float64(m.NumGC),
		OtherSys:      float64(m.OtherSys),
		PauseTotalNs:  float64(m.PauseTotalNs),
		StackInuse:    float64(m.StackInuse),
		StackSys:      float64(m.StackSys),
		Sys:           float64(m.Sys),
		TotalAlloc:    float64(m.TotalAlloc),
		RandomValue:   rand.Float64(),
	}
}

func (g GaugeMetrics) Gauges() []m.Metrics {
	return []m.Metrics{
		*m.NewGauge("Alloc", g.Alloc),
		*m.NewGauge("BuckHashSys", g.BuckHashSys),
		*m.NewGauge("Frees", g.Frees),
		*m.NewGauge("GCCPUFraction", g.GCCPUFraction),
		*m.NewGauge("GCSys", g.GCSys),
		*m.NewGauge("HeapAlloc", g.HeapAlloc),
		*m.NewGauge("HeapIdle", g.HeapIdle),
		*m.NewGauge("HeapInuse", g.HeapInuse),
		*m.NewGauge("HeapObjects", g.HeapObjects),
		*m.NewGauge("HeapReleased", g.HeapReleased),
		*m.NewGauge("HeapSys", g.HeapSys),
		*m.NewGauge("LastGC", g.LastGC),
		*m.NewGauge("Lookups", g.Lookups),
		*m.NewGauge("MCacheInuse", g.MCacheInuse),
		*m.NewGauge("MCacheSys", g.MCacheSys),
		*m.NewGauge("MSpanInuse", g.MSpanInuse),
		*m.NewGauge("MSpanSys", g.MSpanSys),
		*m.NewGauge("Mallocs", g.Mallocs),
		*m.NewGauge("NextGC", g.NextGC),
		*m.NewGauge("NumForcedGC", g.NumForcedGC),
		*m.NewGauge("NumGC", g.NumGC),
		*m.NewGauge("OtherSys", g.OtherSys),
		*m.NewGauge("PauseTotalNs", g.PauseTotalNs),
		*m.NewGauge("StackInuse", g.StackInuse),
		*m.NewGauge("StackSys", g.StackSys),
		*m.NewGauge("Sys", g.Sys),
		*m.NewGauge("TotalAlloc", g.TotalAlloc),
		*m.NewGauge("RandomValue", g.RandomValue),
	}
}

func BuildBatch(gauges GaugeMetrics, counters CountMetrics) []m.Metrics {
	batch := gauges.Gauges()
	return append(batch, *m.NewCounter("PollCount", counters.PollCount))
}
