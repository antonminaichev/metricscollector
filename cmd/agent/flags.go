package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/caarlos0/env"
)

// Config stores agent setting.
type Config struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	RateLimit      int    `env:"RATE_LIMIT"`
	HashKey        string `env:"KEY"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

func loadJSONConfig(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func pickConfigPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		}
	}
	return ""
}

// NewConfig initialises new agent configuration.
func NewConfig() (*Config, error) {
	cfg := &Config{Address: "localhost:8080", ReportInterval: 2, PollInterval: 2, RateLimit: 30}

	configPath := pickConfigPathFromArgs(os.Args[1:])
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	if configPath != "" {
		if err := loadJSONConfig(configPath, cfg); err != nil {
			return nil, err
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	address := flag.String("a", cfg.Address, "{Host:port} for server")
	reportInterval := flag.Int("r", cfg.ReportInterval, "Report interval, seconds")
	pollInterval := flag.Int("p", cfg.PollInterval, "Poll interval, seconds")
	rateLimit := flag.Int("l", cfg.RateLimit, "Max concurrent requests")
	hashKey := flag.String("k", cfg.HashKey, "Hash key")
	cryptoKey := flag.String("crypto-key", cfg.CryptoKey, "Path to public key")
	_ = flag.String("c", configPath, "Path to config file (JSON)")

	flag.Parse()

	cfg.Address = *address
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval
	cfg.RateLimit = *rateLimit
	cfg.HashKey = *hashKey
	cfg.CryptoKey = *cryptoKey
	return cfg, nil
}
