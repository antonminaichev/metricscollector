package server

import (
	"net/http"
	"time"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/middleware"
	"github.com/antonminaichev/metricscollector/internal/server/router"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	db "github.com/antonminaichev/metricscollector/internal/server/storage/database"
	fs "github.com/antonminaichev/metricscollector/internal/server/storage/file"
	ms "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"

	"go.uber.org/zap"
)

func StartServer(addr string, storage storage.Storage, hashKey string) error {
	server := &http.Server{
		Addr: addr,
		Handler: logger.WithLogging(
			middleware.HashHandler(
				middleware.GzipHandler(
					router.NewRouter(storage),
				),
				hashKey,
			),
		),
	}
	return server.ListenAndServe()
}

func SetupStorage(DSN string, fspath string, restore bool, storeInterval int) (storage.Storage, error) {
	if DSN != "" {
		logger.Log.Info("Connecting to database", zap.String("dsn", DSN))
		return db.NewPostgresStorage(DSN)
	}

	if fspath != "" {
		fs, err := fs.NewFileStorage(fspath, logger.Log)
		if err != nil {
			return nil, err
		}
		logger.Log.Info("Using file storage", zap.String("path", fspath))

		if restore {
			if err := fs.LoadMetrics(); err != nil {
				logger.Log.Warn("Failed to restore metrics from file", zap.Error(err))
			}
		}

		go startPeriodicSave(fs, storeInterval)
		return fs, nil
	}

	logger.Log.Info("Using in-memory storage")
	return ms.NewMemoryStorage(), nil
}

func startPeriodicSave(fs *fs.FileStorage, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := fs.SaveMetrics(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
	}
}
