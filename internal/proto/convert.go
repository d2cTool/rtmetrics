package proto

import "github.com/d2cTool/rtmetrics/internal/model"

// ToModel превращает protobuf-метрики в model.Metrics для хранилища.
func ToModel(in []*Metric) []model.Metrics {
	out := make([]model.Metrics, 0, len(in))
	for _, m := range in {
		if m == nil || m.Id == "" {
			continue
		}
		switch m.Type {
		case Metric_COUNTER:
			delta := m.Delta
			out = append(out, model.Metrics{ID: m.Id, MType: model.Counter, Delta: &delta})
		case Metric_GAUGE:
			value := m.Value
			out = append(out, model.Metrics{ID: m.Id, MType: model.Gauge, Value: &value})
		}
	}
	return out
}

// FromModel превращает model.Metrics в protobuf-метрики для gRPC.
func FromModel(in []model.Metrics) []*Metric {
	out := make([]*Metric, 0, len(in))
	for _, m := range in {
		switch m.MType {
		case model.Counter:
			item := &Metric{Id: m.ID, Type: Metric_COUNTER}
			if m.Delta != nil {
				item.Delta = *m.Delta
			}
			out = append(out, item)
		case model.Gauge:
			item := &Metric{Id: m.ID, Type: Metric_GAUGE}
			if m.Value != nil {
				item.Value = *m.Value
			}
			out = append(out, item)
		}
	}
	return out
}
