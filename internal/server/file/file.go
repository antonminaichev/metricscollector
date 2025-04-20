package file

import (
	"encoding/json"
	"os"

	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
	"go.uber.org/zap"
)

type FileStorage struct {
	storage  *ms.MemStorage
	filePath string
	logger   *zap.Logger
}

type MetricsData struct {
	Gauge   map[string]float64 `json:"gauge"`
	Counter map[string]int64   `json:"counter"`
}

func NewFileStorage(storage *ms.MemStorage, filePath string, logger *zap.Logger) *FileStorage {
	return &FileStorage{
		storage:  storage,
		filePath: filePath,
		logger:   logger,
	}
}

func (fs *FileStorage) SaveMetrics() error {
	data := MetricsData{
		Gauge:   fs.storage.Gauge,
		Counter: fs.storage.Counter,
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

	fs.storage.Gauge = metricsData.Gauge
	fs.storage.Counter = metricsData.Counter

	return nil
}
