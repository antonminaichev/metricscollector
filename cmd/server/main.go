package main

import (
	"log"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server"
	"go.uber.org/zap"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func printBuildInfo() {
	v := buildVersion
	if v == "" {
		v = "N/A"
	}
	d := buildDate
	if d == "" {
		d = "N/A"
	}
	c := buildCommit
	if c == "" {
		c = "N/A"
	}

	log.Printf("Build version: %s\n", v)
	log.Printf("Build date: %s\n", d)
	log.Printf("Build commit: %s\n", c)
}

func run() error {
	printBuildInfo()

	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err = logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	storage, err := server.SetupStorage(cfg.DatabaseConnection, cfg.FileStoragePath, cfg.Restore, cfg.StoreInterval)
	if err != nil {
		return err
	}

	logger.Log.Info("Starting server", zap.String("address", cfg.Address))
	return server.StartServer(cfg.Address, storage, cfg.HashKey)
}
