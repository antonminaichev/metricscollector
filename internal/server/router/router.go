// Router package is used for creating http server handlers layout.
package router

import (
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/server/handlers"
	"github.com/antonminaichev/metricscollector/internal/server/storage"
	"github.com/go-chi/chi"
)

// NewRouter creates a router with a handlers layout.
func NewRouter(s storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handlers.PrintAllMetrics(w, r, s)
		})
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			handlers.PingDatabase(w, r, s)
		})
		r.Post("/update", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetricJSON(w, r, s)
		})
		r.Post("/update/", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetricJSON(w, r, s)
		})
		r.Post("/updates/", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetricsJSON(w, r, s)
		})
		r.Post("/value", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetricJSON(w, r, s)
		})
		r.Post("/value/", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetricJSON(w, r, s)
		})
		r.Get("/health", handlers.HealthCheck)
		r.Get("/value/{type}/{metric}", func(w http.ResponseWriter, r *http.Request) {
			handlers.GetMetric(w, r, s)
		})
		r.Post("/update/{type}/{metric}/{value}", func(w http.ResponseWriter, r *http.Request) {
			handlers.PostMetric(w, r, s)
		})
	})
	return r
}
