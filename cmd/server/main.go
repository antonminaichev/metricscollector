package main

import (
	"log"
	"net/http"
	"time"

	"github.com/antonminaichev/metricscollector/internal/database"
	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/file"
	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
	"github.com/antonminaichev/metricscollector/internal/server/middleware"
	"github.com/antonminaichev/metricscollector/internal/server/router"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// Run defines MemStorage for metrics and launch http server
func run() error {
	storage := &ms.MemStorage{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	// Database env or flag is not empty
	if cfg.DatabaseConnection != "" {
		logger.Log.Info("Connecting to DB", zap.String("DSN", cfg.DatabaseConnection))
		err = database.InitDB(cfg.DatabaseConnection)
		if err == nil {
			if err := database.InitMetricsTable(); err != nil {
				logger.Log.Error("Failed to initialize metrics table", zap.Error(err))
				cfg.DatabaseConnection = ""
			} else {
				logger.Log.Info("Using database storage")
				server := &http.Server{
					Addr:    cfg.Address,
					Handler: logger.WithLogging(middleware.GzipHandler(router.NewRouter(storage))),
				}
				return server.ListenAndServe()
			}
		}
		logger.Log.Info("Error connecting to DB, falling back to file storage", zap.Error(err))
	}

	// File storage env or flag is not empty
	if cfg.FileStoragePath != "" {
		fileStorage := file.NewFileStorage(storage, cfg.FileStoragePath, logger.Log)

		if cfg.Restore {
			if err := fileStorage.LoadMetrics(); err != nil {
				logger.Log.Error("Failed to load metrics from file", zap.Error(err))
			}
		}

		server := &http.Server{
			Addr:    cfg.Address,
			Handler: logger.WithLogging(middleware.GzipHandler(router.NewRouter(storage))),
		}

		go func() {
			logger.Log.Info("Running server with file storage", zap.String("address", cfg.Address))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Log.Error("Server error", zap.Error(err))
			}
		}()

		for {
			if err := fileStorage.SaveMetrics(); err != nil {
				logger.Log.Error("Failed to save metrics to file", zap.Error(err))
			}
			time.Sleep(time.Duration(cfg.StoreInterval) * time.Second)
		}
	}

	// DSN and file storage env/flags are empty
	logger.Log.Info("Using RAM storage")
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: logger.WithLogging(middleware.GzipHandler(router.NewRouter(storage))),
	}

	logger.Log.Info("Running server with RAM storage", zap.String("address", cfg.Address))
	return server.ListenAndServe()
}
