package main

import (
	"log"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	storage, err := server.SetupStorage(cfg.DatabaseConnection, cfg.FileStoragePath, cfg.Restore, cfg.StoreInterval)
	if err != nil {
		return err
	}

	logger.Log.Info("Starting server", zap.String("address", cfg.Address))
	return server.StartServer(cfg.Address, storage, cfg.HashKey)
}
