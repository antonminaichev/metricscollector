package file

import (
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

// Storage интерфейс для работы с метриками
type storage interface {
	UpdateCounter(name string, value int64)
	UpdateGauge(name string, value float64)
	GetCounter() map[string]int64
	GetGauge() map[string]float64
	PrintAllMetrics() string
}

type FileStorage struct {
	storage  storage
	filePath string
	logger   *zap.Logger
}

type MetricsData struct {
	Gauge   map[string]float64 `json:"gauge"`
	Counter map[string]int64   `json:"counter"`
}

func NewFileStorage(storage storage, filePath string, logger *zap.Logger) *FileStorage {
	return &FileStorage{
		storage:  storage,
		filePath: filePath,
		logger:   logger,
	}
}

func (fs *FileStorage) SaveMetrics() error {
	data := MetricsData{
		Gauge:   fs.storage.GetGauge(),
		Counter: fs.storage.GetCounter(),
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, file, 0644)
}

func (fs *FileStorage) LoadMetrics() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metricsData MetricsData
	if err := json.Unmarshal(data, &metricsData); err != nil {
		return err
	}

	for name, value := range metricsData.Gauge {
		fs.storage.UpdateGauge(name, value)
	}

	for name, value := range metricsData.Counter {
		fs.storage.UpdateCounter(name, value)
	}

	return nil
}
