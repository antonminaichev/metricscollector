package main

import (
	"net/http"

	h "github.com/antonminaichev/metricscollector/internal/server/handlers"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.HealthCheck)
	mux.HandleFunc("/update/", func(w http.ResponseWriter, r *http.Request) {
		h.PostMetric(w, r, storage)
	})
	return http.ListenAndServe(`:8080`, mux)
}
