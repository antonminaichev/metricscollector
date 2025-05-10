package storage

// MetricType определяет тип метрики
type MetricType string

const (
	Counter MetricType = "counter"
	Gauge   MetricType = "gauge"
)

// Metric представляет собой метрику с её значением
type Metric struct {
	ID    string     `json:"id"`              // имя метрики
	MType MetricType `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64     `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64   `json:"value,omitempty"`
}

// Storage определяет интерфейс для хранения метрик
type Storage interface {
	// UpdateMetric обновляет или создает метрику
	UpdateMetric(id string, mType MetricType, delta *int64, value *float64) error

	// GetMetric возвращает значение метрики
	GetMetric(id string, mType MetricType) (*int64, *float64, error)

	// GetAllMetrics возвращает все метрики
	GetAllMetrics() (map[string]int64, map[string]float64, error)

	// Ping проверяет доступность хранилища
	Ping() error
}
