package handlers

import (
	"context"
	"io"
	"net/http"
)

// MetricPrinter интерфейс для вывода метрик
type metricPrinter interface {
	PrintAllMetrics() string
}

type database interface {
	Ping(ctx context.Context) error
	Close()
}

// HealthCheck используется для проверки доступности сервера
func HealthCheck(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status": "ok"}`))
}

// PrintAllMetrics выводит все метрики
func PrintAllMetrics(rw http.ResponseWriter, r *http.Request, mp metricPrinter) {
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)
	io.WriteString(rw, mp.PrintAllMetrics())
}

func PingDB(w http.ResponseWriter, r *http.Request, db database) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := db.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
