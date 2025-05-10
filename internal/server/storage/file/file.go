package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"go.uber.org/zap"
)

// FileStorage реализует интерфейс Storage для хранения метрик в файле
type FileStorage struct {
	filePath string
	metrics  struct {
		Counters map[string]int64   `json:"counters"`
		Gauges   map[string]float64 `json:"gauges"`
	}
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewFileStorage создает новый экземпляр FileStorage
func NewFileStorage(filePath string, logger *zap.Logger) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		logger:   logger,
		metrics: struct {
			Counters map[string]int64   `json:"counters"`
			Gauges   map[string]float64 `json:"gauges"`
		}{
			Counters: make(map[string]int64),
			Gauges:   make(map[string]float64),
		},
	}

	// Загружаем данные из файла, если он существует
	if err := fs.LoadMetrics(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return fs, nil
}

// UpdateMetric обновляет или создает метрику
func (fs *FileStorage) UpdateMetric(id string, mType storage.MetricType, delta *int64, value *float64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	switch mType {
	case storage.Counter:
		if delta != nil {
			fs.metrics.Counters[id] += *delta
		}
	case storage.Gauge:
		if value != nil {
			fs.metrics.Gauges[id] = *value
		}
	}

	if err := fs.SaveMetrics(); err != nil {
		fs.logger.Error("failed to save metrics to file", zap.Error(err))
		return err
	}

	return nil
}

// GetMetric возвращает значение метрики
func (fs *FileStorage) GetMetric(id string, mType storage.MetricType) (*int64, *float64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	switch mType {
	case storage.Counter:
		if value, ok := fs.metrics.Counters[id]; ok {
			return &value, nil, nil
		}
	case storage.Gauge:
		if value, ok := fs.metrics.Gauges[id]; ok {
			return nil, &value, nil
		}
	default:
		return nil, nil, fmt.Errorf("unknown metric type: %s", mType)
	}

	return nil, nil, fmt.Errorf("metric not found")
}

// GetAllMetrics возвращает все метрики
func (fs *FileStorage) GetAllMetrics() (map[string]int64, map[string]float64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	counters := make(map[string]int64, len(fs.metrics.Counters))
	gauges := make(map[string]float64, len(fs.metrics.Gauges))

	for k, v := range fs.metrics.Counters {
		counters[k] = v
	}
	for k, v := range fs.metrics.Gauges {
		gauges[k] = v
	}

	return counters, gauges, nil
}

// Ping проверяет доступность хранилища
func (fs *FileStorage) Ping() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	_, err := os.Stat(fs.filePath)
	return err
}

// loadFromFile загружает метрики из файла
func (fs *FileStorage) LoadMetrics() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &fs.metrics)
}

// saveToFile сохраняет метрики в файл
func (fs *FileStorage) SaveMetrics() error {
	data, err := json.MarshalIndent(fs.metrics, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
}
