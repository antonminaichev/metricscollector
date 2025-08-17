package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFileStorage(t *testing.T) {
	logger := zap.NewNop()

	t.Run("creates new file storage successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test_metrics.json")

		fs, err := NewFileStorage(filePath, logger)
		assert.NoError(t, err)
		assert.NotNil(t, fs)
		assert.Equal(t, filePath, fs.filePath)
	})

	t.Run("loads existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "existing_metrics.json")

		// Создаем файл с данными
		testData := `{"counters":{"test_counter":5},"gauges":{"test_gauge":3.14}}`
		err := os.WriteFile(filePath, []byte(testData), 0644)
		require.NoError(t, err)

		fs, err := NewFileStorage(filePath, logger)
		assert.NoError(t, err)
		assert.NotNil(t, fs)

		// Проверяем, что данные загружены
		assert.Equal(t, int64(5), fs.metrics.Counters["test_counter"])
		assert.Equal(t, 3.14, fs.metrics.Gauges["test_gauge"])
	})

	t.Run("handles invalid JSON file gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "invalid.json")

		// Создаем файл с невалидным JSON
		err := os.WriteFile(filePath, []byte("invalid json"), 0644)
		require.NoError(t, err)

		// NewFileStorage должен вернуть ошибку при попытке загрузить невалидный JSON
		_, err = NewFileStorage(filePath, logger)
		assert.Error(t, err)
	})
}

func TestFileStorage_UpdateMetric(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_metrics.json")

	fs, err := NewFileStorage(filePath, logger)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("updates counter metric", func(t *testing.T) {
		delta := int64(10)
		err := fs.UpdateMetric(ctx, "test_counter", storage.Counter, &delta, nil)
		assert.NoError(t, err)
		assert.Equal(t, delta, fs.metrics.Counters["test_counter"])

		// Обновляем еще раз
		delta2 := int64(5)
		err = fs.UpdateMetric(ctx, "test_counter", storage.Counter, &delta2, nil)
		assert.NoError(t, err)
		assert.Equal(t, int64(15), fs.metrics.Counters["test_counter"])
	})

	t.Run("updates gauge metric", func(t *testing.T) {
		value := 3.14
		err := fs.UpdateMetric(ctx, "test_gauge", storage.Gauge, nil, &value)
		assert.NoError(t, err)
		assert.Equal(t, value, fs.metrics.Gauges["test_gauge"])

		// Перезаписываем значение
		value2 := 2.71
		err = fs.UpdateMetric(ctx, "test_gauge", storage.Gauge, nil, &value2)
		assert.NoError(t, err)
		assert.Equal(t, value2, fs.metrics.Gauges["test_gauge"])
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		delta := int64(10)
		err := fs.UpdateMetric(ctx, "test", storage.Counter, &delta, nil)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("handles counter with nil delta", func(t *testing.T) {
		err := fs.UpdateMetric(ctx, "nil_counter", storage.Counter, nil, nil)
		assert.NoError(t, err)
		// Метрика не должна быть добавлена
		_, exists := fs.metrics.Counters["nil_counter"]
		assert.False(t, exists)
	})

	t.Run("handles gauge with nil value", func(t *testing.T) {
		err := fs.UpdateMetric(ctx, "nil_gauge", storage.Gauge, nil, nil)
		assert.NoError(t, err)
		// Метрика не должна быть добавлена
		_, exists := fs.metrics.Gauges["nil_gauge"]
		assert.False(t, exists)
	})
}

func TestFileStorage_GetMetric(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_metrics.json")

	fs, err := NewFileStorage(filePath, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Добавляем тестовые данные
	fs.metrics.Counters["test_counter"] = 42
	fs.metrics.Gauges["test_gauge"] = 3.14

	t.Run("gets counter metric", func(t *testing.T) {
		delta, value, err := fs.GetMetric(ctx, "test_counter", storage.Counter)
		assert.NoError(t, err)
		assert.NotNil(t, delta)
		assert.Nil(t, value)
		assert.Equal(t, int64(42), *delta)
	})

	t.Run("gets gauge metric", func(t *testing.T) {
		delta, value, err := fs.GetMetric(ctx, "test_gauge", storage.Gauge)
		assert.NoError(t, err)
		assert.Nil(t, delta)
		assert.NotNil(t, value)
		assert.Equal(t, 3.14, *value)
	})

	t.Run("returns error for non-existent counter", func(t *testing.T) {
		_, _, err := fs.GetMetric(ctx, "non_existent", storage.Counter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric not found")
	})

	t.Run("returns error for non-existent gauge", func(t *testing.T) {
		_, _, err := fs.GetMetric(ctx, "non_existent", storage.Gauge)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric not found")
	})

	t.Run("returns error for unknown metric type", func(t *testing.T) {
		_, _, err := fs.GetMetric(ctx, "test", storage.MetricType("unknown"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metric type")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := fs.GetMetric(ctx, "test_counter", storage.Counter)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestFileStorage_GetAllMetrics(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_metrics.json")

	fs, err := NewFileStorage(filePath, logger)
	require.NoError(t, err)

	ctx := context.Background()

	// Добавляем тестовые данные
	fs.metrics.Counters["counter1"] = 10
	fs.metrics.Counters["counter2"] = 20
	fs.metrics.Gauges["gauge1"] = 1.1
	fs.metrics.Gauges["gauge2"] = 2.2

	t.Run("gets all metrics", func(t *testing.T) {
		counters, gauges, err := fs.GetAllMetrics(ctx)
		assert.NoError(t, err)

		assert.Len(t, counters, 2)
		assert.Equal(t, int64(10), counters["counter1"])
		assert.Equal(t, int64(20), counters["counter2"])

		assert.Len(t, gauges, 2)
		assert.Equal(t, 1.1, gauges["gauge1"])
		assert.Equal(t, 2.2, gauges["gauge2"])
	})

	t.Run("returns empty maps when no metrics", func(t *testing.T) {
		emptyFs, err := NewFileStorage(filepath.Join(tempDir, "empty.json"), logger)
		require.NoError(t, err)

		counters, gauges, err := emptyFs.GetAllMetrics(ctx)
		assert.NoError(t, err)
		assert.Empty(t, counters)
		assert.Empty(t, gauges)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := fs.GetAllMetrics(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestFileStorage_Ping(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_metrics.json")

	fs, err := NewFileStorage(filePath, logger)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("ping returns no error when file exists", func(t *testing.T) {
		// Сохраняем файл
		err := fs.SaveMetrics()
		require.NoError(t, err)

		err = fs.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("ping returns error when file does not exist", func(t *testing.T) {
		nonExistentFs, err := NewFileStorage(filepath.Join(tempDir, "non_existent.json"), logger)
		require.NoError(t, err)

		err = nonExistentFs.Ping(ctx)
		assert.Error(t, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := fs.Ping(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestFileStorage_LoadMetrics(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()

	t.Run("loads valid JSON file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "valid.json")
		testData := `{"counters":{"c1":100},"gauges":{"g1":99.9}}`
		err := os.WriteFile(filePath, []byte(testData), 0644)
		require.NoError(t, err)

		fs, err := NewFileStorage(filePath, logger)
		require.NoError(t, err)

		err = fs.LoadMetrics()
		assert.NoError(t, err)
		assert.Equal(t, int64(100), fs.metrics.Counters["c1"])
		assert.Equal(t, 99.9, fs.metrics.Gauges["g1"])
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "non_existent.json")
		fs, err := NewFileStorage(filePath, logger)
		require.NoError(t, err)

		err = fs.LoadMetrics()
		assert.NoError(t, err) // Должен вернуть nil для несуществующего файла
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "invalid_loadmetrics.json")
		err := os.WriteFile(filePath, []byte("invalid json"), 0644)
		require.NoError(t, err)

		_, err = NewFileStorage(filePath, logger)
		assert.Error(t, err) // NewFileStorage должен вернуть ошибку из-за невалидного JSON в LoadMetrics
	})
}

func TestFileStorage_SaveMetrics(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "save_test.json")

	fs, err := NewFileStorage(filePath, logger)
	require.NoError(t, err)

	t.Run("saves metrics to file", func(t *testing.T) {
		fs.metrics.Counters["save_counter"] = 123
		fs.metrics.Gauges["save_gauge"] = 45.67

		err := fs.SaveMetrics()
		assert.NoError(t, err)

		// Проверяем, что файл создан
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Проверяем содержимое файла
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "save_counter")
		assert.Contains(t, content, "save_gauge")
		assert.Contains(t, content, "123")
		assert.Contains(t, content, "45.67")
	})

	t.Run("handles write error", func(t *testing.T) {
		// Создаем FileStorage с недоступным для записи путем
		invalidPath := "/root/invalid/path/metrics.json"
		invalidFs, err := NewFileStorage(invalidPath, logger)
		require.NoError(t, err)

		err = invalidFs.SaveMetrics()
		assert.Error(t, err)
	})
}

func TestFileStorage_Integration(t *testing.T) {
	logger := zap.NewNop()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "integration_test.json")

	t.Run("full workflow", func(t *testing.T) {
		ctx := context.Background()

		// Создаем хранилище
		fs, err := NewFileStorage(filePath, logger)
		require.NoError(t, err)

		// Добавляем метрики
		delta1 := int64(10)
		err = fs.UpdateMetric(ctx, "requests", storage.Counter, &delta1, nil)
		require.NoError(t, err)

		value1 := 98.6
		err = fs.UpdateMetric(ctx, "temperature", storage.Gauge, nil, &value1)
		require.NoError(t, err)

		// Проверяем, что данные сохранены в файл
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Создаем новое хранилище с тем же файлом
		fs2, err := NewFileStorage(filePath, logger)
		require.NoError(t, err)

		// Проверяем, что данные загружены
		delta, _, err := fs2.GetMetric(ctx, "requests", storage.Counter)
		require.NoError(t, err)
		assert.Equal(t, int64(10), *delta)

		_, value, err := fs2.GetMetric(ctx, "temperature", storage.Gauge)
		require.NoError(t, err)
		assert.Equal(t, 98.6, *value)

		// Получаем все метрики
		counters, gauges, err := fs2.GetAllMetrics(ctx)
		require.NoError(t, err)
		assert.Len(t, counters, 1)
		assert.Len(t, gauges, 1)
	})
}
