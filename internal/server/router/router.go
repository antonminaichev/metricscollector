package router

import (
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/server/handlers"
	"github.com/go-chi/chi"
)

// MetricRouter является составным интерфейсом для всех операций
type metricStorage interface {
	UpdateCounter(name string, value int64)
	UpdateGauge(name string, value float64)
	GetCounter() map[string]int64
	GetGauge() map[string]float64
	PrintAllMetrics() string
}

func NewRouter(ms metricStorage) chi.Router {
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handlers.PrintAllMetrics(w, r, ms)
		})
		r.Get("/ping", handlers.PingDatabase)
		r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetricJSON(w, r, ms, ms)
		})
		r.Post("/update/", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetricJSON(w, r, ms, ms)
		})
		r.Post("/value", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetricJSON(w, r, ms)
		})
		r.Post("/value/", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetricJSON(w, r, ms)
		})
		r.Get("/health", handlers.HealthCheck)
		r.Get("/value/{type}/{metric}", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetric(w, r, ms)
		})
		r.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetric(w, r, ms)
		})
	})
	return r
}
