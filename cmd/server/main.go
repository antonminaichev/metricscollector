package main

import (
	"log"
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/server/handlers"
	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
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
	return http.ListenAndServe(cfg.Address, handlers.MetricRouter(storage))
}
