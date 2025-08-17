package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Additional tests for better coverage

func TestMemoryStorage_UpdateMetric_Errors(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	t.Run("counter without delta", func(t *testing.T) {
		err := s.UpdateMetric(ctx, "test", storage.Counter, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delta value is required")
	})

	t.Run("gauge without value", func(t *testing.T) {
		err := s.UpdateMetric(ctx, "test", storage.Gauge, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})

	t.Run("unknown metric type", func(t *testing.T) {
		err := s.UpdateMetric(ctx, "test", storage.MetricType("unknown"), ptrInt64(1), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metric type")
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := s.UpdateMetric(cancelCtx, "test", storage.Counter, ptrInt64(1), nil)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestMemoryStorage_GetMetric_Errors(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	t.Run("counter not found", func(t *testing.T) {
		_, _, err := s.GetMetric(ctx, "nonexistent", storage.Counter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric not found")
	})

	t.Run("gauge not found", func(t *testing.T) {
		_, _, err := s.GetMetric(ctx, "nonexistent", storage.Gauge)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric not found")
	})

	t.Run("unknown metric type", func(t *testing.T) {
		_, _, err := s.GetMetric(ctx, "test", storage.MetricType("unknown"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metric type")
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := s.GetMetric(cancelCtx, "test", storage.Counter)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestMemoryStorage_GetAllMetrics_Context(t *testing.T) {
	s := NewMemoryStorage()

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := s.GetAllMetrics(cancelCtx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("empty storage", func(t *testing.T) {
		counters, gauges, err := s.GetAllMetrics(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, counters)
		assert.Empty(t, gauges)
	})
}

func TestMemoryStorage_Ping(t *testing.T) {
	s := NewMemoryStorage()

	t.Run("successful ping", func(t *testing.T) {
		err := s.Ping(context.Background())
		assert.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := s.Ping(cancelCtx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestMemoryStorage_CounterAccumulation(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	id := "accumulator"

	// Добавляем несколько значений
	err := s.UpdateMetric(ctx, id, storage.Counter, ptrInt64(10), nil)
	require.NoError(t, err)

	err = s.UpdateMetric(ctx, id, storage.Counter, ptrInt64(5), nil)
	require.NoError(t, err)

	err = s.UpdateMetric(ctx, id, storage.Counter, ptrInt64(3), nil)
	require.NoError(t, err)

	// Проверяем накопление
	delta, _, err := s.GetMetric(ctx, id, storage.Counter)
	require.NoError(t, err)
	assert.Equal(t, int64(18), *delta) // 10 + 5 + 3
}

func TestMemoryStorage_GaugeOverwrite(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	id := "temperature"

	// Устанавливаем начальное значение
	err := s.UpdateMetric(ctx, id, storage.Gauge, nil, ptrFloat64(20.5))
	require.NoError(t, err)

	// Перезаписываем значение
	err = s.UpdateMetric(ctx, id, storage.Gauge, nil, ptrFloat64(25.3))
	require.NoError(t, err)

	// Проверяем, что значение перезаписано
	_, value, err := s.GetMetric(ctx, id, storage.Gauge)
	require.NoError(t, err)
	assert.Equal(t, 25.3, *value)
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	// Запускаем несколько горутин для конкурентного доступа
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()

			// Каждая горутина обновляет свой счетчик
			err := s.UpdateMetric(ctx, fmt.Sprintf("counter_%d", i), storage.Counter, ptrInt64(1), nil)
			if err != nil {
				t.Errorf("UpdateMetric failed: %v", err)
				return
			}

			// И свой gauge
			err = s.UpdateMetric(ctx, fmt.Sprintf("gauge_%d", i), storage.Gauge, nil, ptrFloat64(float64(i)))
			if err != nil {
				t.Errorf("UpdateMetric failed: %v", err)
				return
			}
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что все данные записались корректно
	counters, gauges, err := s.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Len(t, counters, 10)
	assert.Len(t, gauges, 10)

	for i := 0; i < 10; i++ {
		assert.Equal(t, int64(1), counters[fmt.Sprintf("counter_%d", i)])
		assert.Equal(t, float64(i), gauges[fmt.Sprintf("gauge_%d", i)])
	}
}

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }
