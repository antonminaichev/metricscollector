package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	Address            string `env:"ADDRESS" envDefault:"localhost:8080"`
	LogLevel           string `env:"LOG_LEVEL" envDefault:"INFO"`
	StoreInterval      int    `env:"STORE_INTERVAL" envDefault:"300"`
	FileStoragePath    string `env:"FILE_STORAGE_PATH" envDefault:"./metrics/metrics.json"`
	Restore            bool   `env:"RESTORE" envDefault:"true"`
	DatabaseConnection string `env:"DATABASE_DSN"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Определяем флаги
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	loglevel := flag.String("l", cfg.LogLevel, "Log level for server")
	storeInterval := flag.Int("i", cfg.StoreInterval, "Store interval in seconds")
	filePath := flag.String("f", cfg.FileStoragePath, "File storage path")
	restore := flag.Bool("r", cfg.Restore, "Restore metrics from file")
	databaseConnection := flag.String("d", cfg.DatabaseConnection, "Database connection string")

	flag.Parse()

	// Обновляем конфигурацию значениями из флагов
	cfg.Address = *address
	cfg.LogLevel = *loglevel
	cfg.StoreInterval = *storeInterval
	cfg.FileStoragePath = *filePath
	cfg.Restore = *restore
	cfg.DatabaseConnection = *databaseConnection

	return cfg, nil
}
