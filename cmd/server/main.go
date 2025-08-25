package main

import (
	"log"
	"os"

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
	//Unsure how to pass subnet flag to main.go without force SET ENV
	_ = os.Setenv("TRUSTED_SUBNET", cfg.TrustedSubnet)

	if err = logger.Initialize(cfg.LogLevel); err != nil {
		return err
	}

	storage, err := server.SetupStorage(cfg.DatabaseConnection, cfg.FileStoragePath, cfg.Restore, cfg.StoreInterval)
	if err != nil {
		return err
	}

	if cfg.Mode == "grpc" {
		addr := cfg.GRPCAddress
		if addr == "" {
			addr = cfg.Address
		}
		logger.Log.Info("Starting gRPC server", zap.String("address", addr))
		return server.StartGRPCServer(addr, storage, cfg.HashKey, cfg.CryptoKey, cfg.TrustedSubnet)
	}

	logger.Log.Info("Starting HTTP server", zap.String("address", cfg.Address))
	return server.StartServer(cfg.Address, storage, cfg.HashKey, cfg.CryptoKey, cfg.TrustedSubnet)
}
