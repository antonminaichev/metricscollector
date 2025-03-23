package memstorage

// MetricStorager interface defines metods for MemStorage type.
type MemStorager interface {
	UpdateCounter(name string, value int64) error
	UpdateGauge(name string, value float64) error
	GetCounter() map[string]int64
	GetGauge() map[string]float64
}

// MemStorage is a type struct for counter and gauge metrics. MemStorage shoud implement MemStorager interface.
type MemStorage struct {
	Counter map[string]int64
	Gauge   map[string]float64
}

// UpdateCounter sum the value of {name} counter metric.
func (storage *MemStorage) UpdateCounter(name string, value int64) {
	storage.Counter[name] += value
}

// UpdateGauge change the value of {name} gauge metric.
func (storage *MemStorage) UpdateGauge(name string, value float64) {
	storage.Gauge[name] = value
}

// GetCounter returns map of counter metrics.
func (storage *MemStorage) GetCounter() map[string]int64 {
	return storage.Counter
}

// GetGauge returns map of gauge metrics.
func (storage *MemStorage) GetGauge() map[string]float64 {
	return storage.Gauge
}
