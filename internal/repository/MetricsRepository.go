package repository

type MetricsRepository interface {
	SaveCounter(name string, value float64) (float64, error)
	SaveGauge(name string, value int64) (int64, error)

	GetCounter(name string) (float64, error)
	GetGauge(name string) (float64, error)

	GetAllCounters() (map[string]float64, error)
	GetAllGauges() (map[string]int64, error)
}
