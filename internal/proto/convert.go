package proto

import "github.com/d2cTool/rtmetrics/internal/model"

// ToModel превращает protobuf-метрики в model.Metrics для хранилища.
func ToModel(in []*Metric) []model.Metrics {
	out := make([]model.Metrics, 0, len(in))
	for _, m := range in {
		if m == nil || m.GetId() == "" {
			continue
		}
		switch m.GetType() {
		case Metric_COUNTER:
			delta := m.GetDelta()
			out = append(out, model.Metrics{ID: m.GetId(), MType: model.Counter, Delta: &delta})
		case Metric_GAUGE:
			value := m.GetValue()
			out = append(out, model.Metrics{ID: m.GetId(), MType: model.Gauge, Value: &value})
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
			item := new(Metric)
			item.SetId(m.ID)
			item.SetType(Metric_COUNTER)
			if m.Delta != nil {
				item.SetDelta(*m.Delta)
			}
			out = append(out, item)
		case model.Gauge:
			item := new(Metric)
			item.SetId(m.ID)
			item.SetType(Metric_GAUGE)
			if m.Value != nil {
				item.SetValue(*m.Value)
			}
			out = append(out, item)
		}
	}
	return out
}
