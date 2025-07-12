// Storage package is used for creating and operating different metric storage types.
package storage

import "context"

// MetricType defines metric type.
type MetricType string

// Two main metric types.
const (
	Counter MetricType = "counter"
	Gauge   MetricType = "gauge"
)

// Metric presents single metric type and its value.
type Metric struct {
	ID    string     `json:"id"`              // имя метрики
	MType MetricType `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64     `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64   `json:"value,omitempty"`
}

// Storage defines an interface for metric operations.
type Storage interface {
	// UpdateMetric updates or creates metric in a storage.
	UpdateMetric(ctx context.Context, id string, mType MetricType, delta *int64, value *float64) error

	// GetMetric returns metric values from a storage.
	GetMetric(ctx context.Context, id string, mType MetricType) (*int64, *float64, error)

	// GetAllMetrics returns all metrics from a storage.
	GetAllMetrics(ctx context.Context) (map[string]int64, map[string]float64, error)

	// Ping checks database availability.
	Ping(ctx context.Context) error
}
