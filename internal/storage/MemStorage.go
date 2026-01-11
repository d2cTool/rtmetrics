package storage

import "errors"

type MemStorage struct {
	Counters map[string]float64
	Gauges   map[string]int64
}

func (s MemStorage) SaveCounter(name string, value float64) (float64, error) {
	s.Counters[name] = value
	return value, nil
}

func (s MemStorage) SaveGauge(name string, value int64) (int64, error) {
	v, e := s.Gauges[name]
	if e {
		s.Gauges[name] = v + value
	} else {
		s.Gauges[name] = value
	}

	return s.Gauges[name], nil
}

func (s MemStorage) GetCounter(name string) (float64, error) {
	v, e := s.Counters[name]
	if e {
		return v, nil
	}

	return 0.0, errors.New("can't find counter " + name)
}

func (s MemStorage) GetGauge(name string) (int64, error) {
	v, e := s.Gauges[name]
	if e {
		return v, nil
	}

	return 0, errors.New("can't find gauge " + name)
}

func (s MemStorage) GetAllCounters() (map[string]float64, error) {
	return s.Counters, nil
}

func (s MemStorage) GetAllGauges() (map[string]int64, error) {
	return s.Gauges, nil
}
