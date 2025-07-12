package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"go.uber.org/zap"
)

// FileStorage realises intreface for metric storage in a file.
type FileStorage struct {
	filePath string
	metrics  struct {
		Counters map[string]int64   `json:"counters"`
		Gauges   map[string]float64 `json:"gauges"`
	}
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewFileStorage creates a new instance of FileStorage.
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

// UpdateMetric updates or creates metric if it doesnt exist.
func (fs *FileStorage) UpdateMetric(ctx context.Context, id string, mType storage.MetricType, delta *int64, value *float64) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

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

// GetMetric returns metric value from a storage.
func (fs *FileStorage) GetMetric(ctx context.Context, id string, mType storage.MetricType) (*int64, *float64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

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

// GetAllMetrics returns all metrics from a storage.
func (fs *FileStorage) GetAllMetrics(ctx context.Context) (map[string]int64, map[string]float64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

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

// Ping checks database availability.
func (fs *FileStorage) Ping(ctx context.Context) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, err := os.Stat(fs.filePath)
	return err
}

// LoadMetrics loads metrics from a file to RAM.
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

// SaveMetrics saves metrics to a file.
func (fs *FileStorage) SaveMetrics() error {
	data, err := json.MarshalIndent(fs.metrics, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
}
