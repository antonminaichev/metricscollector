package handlers

import (
	"net/http"
	"strconv"
	"strings"

	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
)

// PostMetric handler is used for adding new metric to MemStorage
func PostMetric(w http.ResponseWriter, r *http.Request, storage *ms.MemStorage) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")

	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(segments) < 4 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := segments[1]
	metricName := segments[2]
	metricValue := segments[3]

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metricType {
	case "counter":
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.UpdateCounter(metricName, v)
	case "gauge":
		v, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.UpdateGauge(metricName, v)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
