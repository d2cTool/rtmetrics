package models

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func NewCounter(id string, delta int64) *Metrics {
	return &Metrics{
		ID:    id,
		MType: Counter,
		Delta: &delta,
	}
}

func NewGauge(id string, value float64) *Metrics {
	return &Metrics{
		ID:    id,
		MType: Gauge,
		Value: &value,
	}
}
