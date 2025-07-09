package main

import (
	"log"
	"net/http"
	"time"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/middleware"
	"github.com/antonminaichev/metricscollector/internal/server/router"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	pg "github.com/antonminaichev/metricscollector/internal/server/storage/database"
	fs "github.com/antonminaichev/metricscollector/internal/server/storage/file"
	ms "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// Run defines storage for metrics and launch http server
func run() error {
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	var s storage.Storage
	// Database env or flag is not empty
	if cfg.DatabaseConnection != "" {
		logger.Log.Info("Connecting to DB", zap.String("DSN", cfg.DatabaseConnection))
		pgStorage, err := pg.NewPostgresStorage(cfg.DatabaseConnection)
		if err != nil {
			logger.Log.Error("Failed to connect to database", zap.Error(err))
			return err
		}
		logger.Log.Info("Using database storage")
		s = pgStorage
		server := &http.Server{
			Addr:    cfg.Address,
			Handler: logger.WithLogging(middleware.HashHandler(middleware.GzipHandler(router.NewRouter(s)), cfg.HashKey)),
		}
		return server.ListenAndServe()
	}

	// File storage env or flag is not empty
	if cfg.FileStoragePath != "" {
		fileStorage, err := fs.NewFileStorage(cfg.FileStoragePath, logger.Log)
		if err != nil {
			logger.Log.Error("Failed to initialize file storage", zap.Error(err))
			return err
		}
		logger.Log.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
		s = fileStorage
		if cfg.Restore {
			if err := fileStorage.LoadMetrics(); err != nil {
				logger.Log.Error("Failed to load metrics from file", zap.Error(err))
			}
		}

		server := &http.Server{
			Addr:    cfg.Address,
			Handler: logger.WithLogging(middleware.HashHandler(middleware.GzipHandler(router.NewRouter(s)), cfg.HashKey)),
		}

		go func() {
			logger.Log.Info("Running server with file storage", zap.String("address", cfg.Address))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Log.Error("Server error", zap.Error(err))
			}
		}()

		// Периодически сохраняем метрики
		for {
			if err := fileStorage.SaveMetrics(); err != nil {
				logger.Log.Error("Failed to save metrics to file", zap.Error(err))
			}
			time.Sleep(time.Duration(cfg.StoreInterval) * time.Second)
		}
	}

	// Если не указано ни файловое хранилище, ни база данных, используем RAM
	logger.Log.Info("Using RAM storage")
	s = ms.NewMemoryStorage()
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: logger.WithLogging(middleware.HashHandler(middleware.GzipHandler(router.NewRouter(s)), cfg.HashKey)),
	}

	logger.Log.Info("Running server with RAM storage", zap.String("address", cfg.Address))
	return server.ListenAndServe()
}
