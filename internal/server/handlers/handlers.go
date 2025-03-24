package handlers

import (
	"net/http"
	"strconv"
	"strings"

	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
)

// PostMetric handler is used for adding new metric to MemStorage
func PostMetric(rw http.ResponseWriter, r *http.Request, storage *ms.MemStorage) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	rw.Header().Set("Content-Type", "text/plain")

	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(segments) < 4 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := segments[1]
	metricName := segments[2]
	metricValue := segments[3]

	if metricName == "" {
		rw.WriteHeader(http.StatusNotFound)
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
