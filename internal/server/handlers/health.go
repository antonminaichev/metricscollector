package handlers

import (
	"io"
	"net/http"
)

// MetricPrinter интерфейс для вывода метрик
type metricPrinter interface {
	PrintAllMetrics() string
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
