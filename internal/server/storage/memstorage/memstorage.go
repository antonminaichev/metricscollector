package storage

import (
	"fmt"
	"sync"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
)

// MemoryStorage реализует интерфейс Storage для хранения в памяти
type MemoryStorage struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
}

// NewMemoryStorage создает новое in-memory хранилище
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (s *MemoryStorage) UpdateMetric(id string, mType storage.MetricType, delta *int64, value *float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch mType {
	case storage.Counter:
		if delta == nil {
			return fmt.Errorf("delta value is required for counter metric")
		}
		s.counters[id] += *delta
	case storage.Gauge:
		if value == nil {
			return fmt.Errorf("value is required for gauge metric")
		}
		s.gauges[id] = *value
	default:
		return fmt.Errorf("unknown metric type: %s", mType)
	}

	return nil
}

func (s *MemoryStorage) GetMetric(id string, mType storage.MetricType) (*int64, *float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch mType {
	case storage.Counter:
		if value, ok := s.counters[id]; ok {
			return &value, nil, nil
		}
	case storage.Gauge:
		if value, ok := s.gauges[id]; ok {
			return nil, &value, nil
		}
	default:
		return nil, nil, fmt.Errorf("unknown metric type: %s", mType)
	}

	return nil, nil, fmt.Errorf("metric not found")
}

func (s *MemoryStorage) GetAllMetrics() (map[string]int64, map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counters := make(map[string]int64, len(s.counters))
	gauges := make(map[string]float64, len(s.gauges))

	for k, v := range s.counters {
		counters[k] = v
	}
	for k, v := range s.gauges {
		gauges[k] = v
	}

	return counters, gauges, nil
}

func (s *MemoryStorage) Ping() error {
	return nil
}
