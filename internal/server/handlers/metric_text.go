package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

type metricUpdater interface {
	UpdateCounter(name string, value int64)
	UpdateGauge(name string, value float64)
}

type metricGetter interface {
	GetCounter() map[string]int64
	GetGauge() map[string]float64
}

// PostMetric обновляет значение метрики
func PostMetric(rw http.ResponseWriter, r *http.Request, mu metricUpdater) {
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
	case MetricTypeCounter:
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.UpdateCounter(metricName, v)
	case MetricTypeGauge:
		v, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.UpdateGauge(metricName, v)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// GetMetric возвращает значение метрики
func GetMetric(rw http.ResponseWriter, r *http.Request, mg metricGetter) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "metric")
	switch metricType {
	case MetricTypeGauge:
		metrics := mg.GetGauge()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such gauge metric "+metricName, http.StatusNotFound)
			return
		}
		io.WriteString(rw, strconv.FormatFloat(value, 'f', -1, 64))
	case MetricTypeCounter:
		metrics := mg.GetCounter()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such counter metric "+metricName, http.StatusNotFound)
		}
		io.WriteString(rw, strconv.FormatInt(value, 10))
	default:
		http.Error(rw, "No such metric type "+metricName, http.StatusNotFound)
	}
}
