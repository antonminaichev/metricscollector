package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/retry"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
)

// HealthCheck checks server availability.
func HealthCheck(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status": "ok"}`))
}

// PrintAllMetrics prints all metrics.
func PrintAllMetrics(rw http.ResponseWriter, r *http.Request, s storage.Storage) {
	counters, gauges, err := s.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(rw, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)

	io.WriteString(rw, "<html><body>")
	io.WriteString(rw, "<h1>Metrics</h1>")

	io.WriteString(rw, "<h2>Counters</h2>")
	io.WriteString(rw, "<ul>")
	for name, value := range counters {
		io.WriteString(rw, fmt.Sprintf("<li>%s: %d</li>", name, value))
	}
	io.WriteString(rw, "</ul>")

	io.WriteString(rw, "<h2>Gauges</h2>")
	io.WriteString(rw, "<ul>")
	for name, value := range gauges {
		io.WriteString(rw, fmt.Sprintf("<li>%s: %f</li>", name, value))
	}
	io.WriteString(rw, "</ul>")

	io.WriteString(rw, "</body></html>")
}

// PingDatabase checks database availability.
func PingDatabase(w http.ResponseWriter, r *http.Request, s storage.Storage) {
	err := retry.Do(retry.DefaultRetryConfig(), func() error {
		return s.Ping(r.Context())
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}
