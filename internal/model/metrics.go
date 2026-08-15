// Package model описывает метрики, которыми обмениваются агент и сервер.
package model

const (
	// Counter — накопительный тип метрики. Повторная запись прибавляет delta.
	Counter = "counter"
	// Gauge — мгновенное значение. Повторная запись заменяет предыдущее.
	Gauge = "gauge"
)

// Metrics — JSON-представление одной метрики в API /update, /updates и /value.
//
// Delta и Value — указатели, чтобы отличить ноль от «поле не задано»
// и не кодировать пустые поля в JSON.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// NewCounter собирает counter с именем id и приращением delta.
func NewCounter(id string, delta int64) *Metrics {
	return &Metrics{
		ID:    id,
		MType: Counter,
		Delta: &delta,
	}
}

// NewGauge собирает gauge с именем id и значением value.
func NewGauge(id string, value float64) *Metrics {
	return &Metrics{
		ID:    id,
		MType: Gauge,
		Value: &value,
	}
}
