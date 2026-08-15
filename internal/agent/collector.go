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

// Gauges собирает runtime-метрики в слайс. Value указывает на поля g:
// структура сбегает в кучу один раз, а не по аллокации на каждую метрику.
func (g GaugeMetrics) Gauges() []m.Metrics {
	out := make([]m.Metrics, 0, 29)
	return append(out,
		m.Metrics{ID: "Alloc", MType: m.Gauge, Value: &g.Alloc},
		m.Metrics{ID: "BuckHashSys", MType: m.Gauge, Value: &g.BuckHashSys},
		m.Metrics{ID: "Frees", MType: m.Gauge, Value: &g.Frees},
		m.Metrics{ID: "GCCPUFraction", MType: m.Gauge, Value: &g.GCCPUFraction},
		m.Metrics{ID: "GCSys", MType: m.Gauge, Value: &g.GCSys},
		m.Metrics{ID: "HeapAlloc", MType: m.Gauge, Value: &g.HeapAlloc},
		m.Metrics{ID: "HeapIdle", MType: m.Gauge, Value: &g.HeapIdle},
		m.Metrics{ID: "HeapInuse", MType: m.Gauge, Value: &g.HeapInuse},
		m.Metrics{ID: "HeapObjects", MType: m.Gauge, Value: &g.HeapObjects},
		m.Metrics{ID: "HeapReleased", MType: m.Gauge, Value: &g.HeapReleased},
		m.Metrics{ID: "HeapSys", MType: m.Gauge, Value: &g.HeapSys},
		m.Metrics{ID: "LastGC", MType: m.Gauge, Value: &g.LastGC},
		m.Metrics{ID: "Lookups", MType: m.Gauge, Value: &g.Lookups},
		m.Metrics{ID: "MCacheInuse", MType: m.Gauge, Value: &g.MCacheInuse},
		m.Metrics{ID: "MCacheSys", MType: m.Gauge, Value: &g.MCacheSys},
		m.Metrics{ID: "MSpanInuse", MType: m.Gauge, Value: &g.MSpanInuse},
		m.Metrics{ID: "MSpanSys", MType: m.Gauge, Value: &g.MSpanSys},
		m.Metrics{ID: "Mallocs", MType: m.Gauge, Value: &g.Mallocs},
		m.Metrics{ID: "NextGC", MType: m.Gauge, Value: &g.NextGC},
		m.Metrics{ID: "NumForcedGC", MType: m.Gauge, Value: &g.NumForcedGC},
		m.Metrics{ID: "NumGC", MType: m.Gauge, Value: &g.NumGC},
		m.Metrics{ID: "OtherSys", MType: m.Gauge, Value: &g.OtherSys},
		m.Metrics{ID: "PauseTotalNs", MType: m.Gauge, Value: &g.PauseTotalNs},
		m.Metrics{ID: "StackInuse", MType: m.Gauge, Value: &g.StackInuse},
		m.Metrics{ID: "StackSys", MType: m.Gauge, Value: &g.StackSys},
		m.Metrics{ID: "Sys", MType: m.Gauge, Value: &g.Sys},
		m.Metrics{ID: "TotalAlloc", MType: m.Gauge, Value: &g.TotalAlloc},
		m.Metrics{ID: "RandomValue", MType: m.Gauge, Value: &g.RandomValue},
	)
}

func BuildBatch(gauges GaugeMetrics, counters CountMetrics) []m.Metrics {
	batch := gauges.Gauges()
	delta := counters.PollCount
	return append(batch, m.Metrics{ID: "PollCount", MType: m.Counter, Delta: &delta})
}
