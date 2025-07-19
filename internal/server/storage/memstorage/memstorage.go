package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
)

// MemoryStorage realises Storage interface for RAM storage.
type MemoryStorage struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
}

// NewMemoryStorage creates new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

// UpdateMetric updates or creates a metric in a in-memory storage.
func (s *MemoryStorage) UpdateMetric(ctx context.Context, id string, mType storage.MetricType, delta *int64, value *float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

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

// GetMetric returns a single metric from in-memory storage.
func (s *MemoryStorage) GetMetric(ctx context.Context, id string, mType storage.MetricType) (*int64, *float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

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

// GetAllMetrics returns all metrics from a in-memory storage.
func (s *MemoryStorage) GetAllMetrics(ctx context.Context) (map[string]int64, map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

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

func (s *MemoryStorage) Ping(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}
