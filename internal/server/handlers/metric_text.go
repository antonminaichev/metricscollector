package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"github.com/go-chi/chi"
)

// PostMetric updates single metric value via plaintext request.
func PostMetric(rw http.ResponseWriter, r *http.Request, s storage.MetricWriter) {
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
	case string(storage.Counter):
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.UpdateMetric(r.Context(), metricName, storage.Counter, &v, nil); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	case string(storage.Gauge):
		v, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.UpdateMetric(r.Context(), metricName, storage.Gauge, nil, &v); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// GetMetric returns a metric values via plaintext request.
func GetMetric(rw http.ResponseWriter, r *http.Request, s storage.MetricReader) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "metric")

	var mType storage.MetricType
	switch metricType {
	case string(storage.Counter):
		mType = storage.Counter
	case string(storage.Gauge):
		mType = storage.Gauge
	default:
		http.Error(rw, "No such metric type "+metricName, http.StatusNotFound)
		return
	}

	delta, value, err := s.GetMetric(r.Context(), metricName, mType)
	if err != nil {
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}

	if mType == storage.Counter && delta != nil {
		if _, err := io.WriteString(rw, strconv.FormatInt(*delta, 10)); err != nil {
			http.Error(rw, "Failed to write response", http.StatusInternalServerError)
			return
		}
	} else if mType == storage.Gauge && value != nil {
		if _, err := io.WriteString(rw, strconv.FormatFloat(*value, 'f', -1, 64)); err != nil {
			http.Error(rw, "Failed to write response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(rw, "Metric value is nil", http.StatusNotFound)
	}
}
