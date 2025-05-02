package handlers

import (
	"io"
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/database"
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

func PingDatabase(w http.ResponseWriter, r *http.Request) {
	err := database.PingDB()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}
