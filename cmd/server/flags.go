package main

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	Address string `env:"ADDRESS" envDefault:"localhost:8080"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	address := flag.String("a", cfg.Address, "{Host:port} for server")
	flag.Parse()
	cfg.Address = *address

	return cfg, nil
}
