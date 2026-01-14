package repository

type MetricsRepository interface {
	SaveCounter(name string, value int64) (int64, error)
	SaveGauge(name string, value float64) (float64, error)

	GetCounter(name string) (int64, error)
	GetGauge(name string) (float64, error)

	GetAllCounters() (map[string]int64, error)
	GetAllGauges() (map[string]float64, error)
}
