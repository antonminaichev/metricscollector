package handlers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

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

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(rw, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer gzr.Close()
			r.Body = io.NopCloser(gzr)
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			rw.Header().Set("Content-Encoding", "gzip")
			gzw := gzip.NewWriter(rw)
			defer gzw.Close()

			gzrw := gzipResponseWriter{Writer: gzw, ResponseWriter: rw}
			next.ServeHTTP(gzrw, r)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
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

// PostMetricJSON updates metric value via JSON request
func PostMetricJSON(rw http.ResponseWriter, r *http.Request, mu metricUpdater, mg metricGetter) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var metric Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var response Metrics
	response.ID = metric.ID
	response.MType = metric.MType

	if metric.ID == "" || metric.MType == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "counter":
		if metric.Delta == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.UpdateCounter(metric.ID, *metric.Delta)
		if val, ok := mg.GetCounter()[response.ID]; ok {
			response.Delta = &val
		}
	case "gauge":
		if metric.Value == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.UpdateGauge(metric.ID, *metric.Value)
		if val, ok := mg.GetGauge()[response.ID]; ok {
			response.Value = &val
		}
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(response); err != nil {
		http.Error(rw, "Can`t encode response", http.StatusInternalServerError)
		return
	}
}

// GetMetricJSON returns metric value via JSON request
func GetMetricJSON(rw http.ResponseWriter, r *http.Request, mg metricGetter) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var metric Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var response Metrics
	response.ID = metric.ID
	response.MType = metric.MType

	switch metric.MType {
	case "gauge":
		metrics := mg.GetGauge()
		value, ok := metrics[metric.ID]
		if !ok {
			http.Error(rw, "No such gauge metric "+metric.ID, http.StatusNotFound)
			return
		}
		response.Value = &value
	case "counter":
		metrics := mg.GetCounter()
		value, ok := metrics[metric.ID]
		if !ok {
			http.Error(rw, "No such counter metric "+metric.ID, http.StatusNotFound)
		}
		response.Delta = &value
	default:
		http.Error(rw, "No such metric type "+metric.ID, http.StatusNotFound)
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(response); err != nil {
		http.Error(rw, "Can`t encode response", http.StatusInternalServerError)
		return
	}
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
			http.Error(rw, "No such counter metric "+metricName, http.StatusNotFound)
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
		r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
			PostMetricJSON(w, r, ms, ms)
		})
		r.Post("/update/", func(w http.ResponseWriter, r *http.Request) {
			PostMetricJSON(w, r, ms, ms)
		})
		r.Post("/value", func(w http.ResponseWriter, r *http.Request) {
			GetMetricJSON(w, r, ms)
		})
		r.Post("/value/", func(w http.ResponseWriter, r *http.Request) {
			GetMetricJSON(w, r, ms)
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
