package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/database"
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

	fileStorage := file.NewFileStorage(storage, cfg.FileStoragePath, logger.Log)

	var dbConnection *database.DB
	if cfg.DatabaseDSN != "" {
		dbConnection, err = database.NewDBConnection(cfg.DatabaseDSN, logger.Log)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := dbConnection.Ping(ctx); err != nil {
			return fmt.Errorf("failed to ping database: %w", err)
		}
		defer dbConnection.Close()
	}

	if cfg.Restore {
		if err := fileStorage.LoadMetrics(); err != nil {
			logger.Log.Error("Failed to load metrics from file", zap.Error(err))
		}
	}

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: logger.WithLogging(middleware.GzipHandler(router.NewRouter(storage, dbConnection))),
	}

	// Создаем канал для получения сигналов завершения
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Log.Info("Running server", zap.String("address", cfg.Address))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Server error", zap.Error(err))
		}
	}()

	// Запускаем сохранение метрик в отдельной горутине
	go func() {
		for {
			if err := fileStorage.SaveMetrics(); err != nil {
				logger.Log.Error("Failed to save metrics to file", zap.Error(err))
			}
			time.Sleep(time.Duration(cfg.StoreInterval) * time.Second)
		}
	}()

	// Ждем сигнала завершения
	<-done
	logger.Log.Info("Server is shutting down...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}

	return nil
}
