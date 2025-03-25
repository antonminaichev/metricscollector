package handlers

import (
	"io"
	"net/http"
	"strconv"

	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
	"github.com/go-chi/chi"
)

// PostMetric handler is used for adding new metric to MemStorage
func PostMetric(rw http.ResponseWriter, r *http.Request, storage *ms.MemStorage) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "metric")
	metricValue := chi.URLParam(r, "value")

	if metricType == "" || metricName == "" || metricValue == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	switch metricType {
	case "counter":
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.UpdateCounter(metricName, v)
	case "gauge":
		v, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.UpdateGauge(metricName, v)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func GetMetric(rw http.ResponseWriter, r *http.Request, storage *ms.MemStorage) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "metric")
	switch metricType {
	case "gauge":
		metrics := storage.GetGauge()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such gauge metric "+metricName, http.StatusNotFound)
		}
		io.WriteString(rw, strconv.FormatFloat(value, 'f', 4, 64))
	case "counter":
		metrics := storage.GetCounter()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such countermetric "+metricName, http.StatusNotFound)
		}
		io.WriteString(rw, strconv.FormatInt(value, 10))
	default:
		http.Error(rw, "No such metric type "+metricName, http.StatusNotFound)
	}
}

func PrintAllMetrics(rw http.ResponseWriter, r *http.Request, storage *ms.MemStorage) {
	rw.Header().Set("Content-Type", "text/html")
	io.WriteString(rw, storage.PrintAllMetrics())
}

// Health Check is used for checking server availability
func HealthCheck(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status": "ok"}`))
}

func MetricRouter(storage *ms.MemStorage) chi.Router {
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			PrintAllMetrics(w, r, storage)
		})
		r.Get("/health", HealthCheck)
		r.Get("/value/{type}/{metric}", func(w http.ResponseWriter, r *http.Request) {
			GetMetric(w, r, storage)
		})
		r.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, r *http.Request) {
			PostMetric(w, r, storage)
		})
	})
	return r
}
