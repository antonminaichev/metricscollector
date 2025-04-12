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

type metricPrinter interface {
	PrintAllMetrics() string
}

// PostMetric updates metric value
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
	case "counter":
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.UpdateCounter(metricName, v)
	case "gauge":
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

// GetMetric returns metric value
func GetMetric(rw http.ResponseWriter, r *http.Request, mg metricGetter) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "metric")
	switch metricType {
	case "gauge":
		metrics := mg.GetGauge()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such gauge metric "+metricName, http.StatusNotFound)
			return
		}
		io.WriteString(rw, strconv.FormatFloat(value, 'f', -1, 64))
	case "counter":
		metrics := mg.GetCounter()
		value, ok := metrics[metricName]
		if !ok {
			http.Error(rw, "No such countermetric "+metricName, http.StatusNotFound)
		}
		io.WriteString(rw, strconv.FormatInt(value, 10))
	default:
		http.Error(rw, "No such metric type "+metricName, http.StatusNotFound)
	}
}

// PrintAllMetrics prints all metrics
func PrintAllMetrics(rw http.ResponseWriter, r *http.Request, mp metricPrinter) {
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)
	io.WriteString(rw, mp.PrintAllMetrics())
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

// MetricRouter is a composite interface for all operations
type metricStorage interface {
	metricUpdater
	metricGetter
	metricPrinter
}

func MetricRouter(ms metricStorage) chi.Router {
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			PrintAllMetrics(w, r, ms)
		})
		r.Get("/health", HealthCheck)
		r.Get("/value/{type}/{metric}", func(w http.ResponseWriter, r *http.Request) {
			GetMetric(w, r, ms)
		})
		r.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, r *http.Request) {
			PostMetric(w, r, ms)
		})
	})
	return r
}
