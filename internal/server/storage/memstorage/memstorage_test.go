package storage

import (
	"context"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"github.com/stretchr/testify/assert"
)

func TestMemoryStorage_CounterUpdateAndGet(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	id := "test_counter"
	delta := int64(5)

	err := s.UpdateMetric(ctx, id, storage.Counter, &delta, nil)
	assert.NoError(t, err)

	gotDelta, _, err := s.GetMetric(ctx, id, storage.Counter)
	assert.NoError(t, err)
	assert.Equal(t, delta, *gotDelta)
}

func TestMemoryStorage_GaugeUpdateAndGet(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	id := "test_gauge"
	value := 3.14

	err := s.UpdateMetric(ctx, id, storage.Gauge, nil, &value)
	assert.NoError(t, err)

	_, gotValue, err := s.GetMetric(ctx, id, storage.Gauge)
	assert.NoError(t, err)
	assert.Equal(t, value, *gotValue)
}

func TestMemoryStorage_GetAllMetrics(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	_ = s.UpdateMetric(ctx, "c1", storage.Counter, ptrInt64(1), nil)
	_ = s.UpdateMetric(ctx, "g1", storage.Gauge, nil, ptrFloat64(2.5))

	counters, gauges, err := s.GetAllMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), counters["c1"])
	assert.Equal(t, 2.5, gauges["g1"])
}

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }
