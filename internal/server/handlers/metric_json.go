// Handlers package contains different http server handlers.
package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
)

// PostMetricJSON updates single metric value via JSON request.
func PostMetricJSON(rw http.ResponseWriter, r *http.Request, s storage.Storage, trustedCIDR string) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !ipAllowed(r, trustedCIDR) {
		http.Error(rw, "client ip is forbidden", http.StatusForbidden)
		return
	}

	var metric storage.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Базовая валидация
	if metric.ID == "" || metric.MType == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var response storage.Metric
	response.ID = metric.ID
	response.MType = metric.MType

	switch metric.MType {
	case storage.Counter:
		if metric.Delta == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.UpdateMetric(r.Context(), metric.ID, storage.Counter, metric.Delta, nil); err != nil {
			http.Error(rw, "Failed to update counter", http.StatusInternalServerError)
			return
		}
		delta, _, err := s.GetMetric(r.Context(), metric.ID, storage.Counter)
		if err != nil {
			http.Error(rw, "Failed to fetch updated metric", http.StatusInternalServerError)
			return
		}
		response.Delta = delta

	case storage.Gauge:
		if metric.Value == nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := s.UpdateMetric(r.Context(), metric.ID, storage.Gauge, nil, metric.Value); err != nil {
			http.Error(rw, "Failed to update gauge", http.StatusInternalServerError)
			return
		}
		_, value, err := s.GetMetric(r.Context(), metric.ID, storage.Gauge)
		if err != nil {
			http.Error(rw, "Failed to fetch updated metric", http.StatusInternalServerError)
			return
		}
		response.Value = value

	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		http.Error(rw, "Can't encode response", http.StatusInternalServerError)
	}
}

// GetMetricJSON returns a metric value via JSON request.
func GetMetricJSON(rw http.ResponseWriter, r *http.Request, s storage.MetricReader) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var metric storage.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var response storage.Metric
	response.ID = metric.ID
	response.MType = metric.MType

	var mType storage.MetricType
	switch metric.MType {
	case storage.Counter:
		mType = storage.Counter
	case storage.Gauge:
		mType = storage.Gauge
	default:
		http.Error(rw, "No such metric type "+metric.ID, http.StatusNotFound)
		return
	}

	delta, value, err := s.GetMetric(r.Context(), metric.ID, mType)
	if err != nil {
		http.Error(rw, "Metric not found", http.StatusNotFound)
		return
	}

	response.Delta = delta
	response.Value = value

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		http.Error(rw, "Can't encode response", http.StatusInternalServerError)
	}
}

// PostMetricsJSON updates a banch of metric values via JSON request.
func PostMetricsJSON(rw http.ResponseWriter, r *http.Request, s storage.Storage, trustedCIDR string) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !ipAllowed(r, trustedCIDR) {
		http.Error(rw, "client ip is forbidden", http.StatusForbidden)
		return
	}

	var metrics []storage.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	response := make([]storage.Metric, 0, len(metrics))

	for _, metric := range metrics {
		if metric.ID == "" || metric.MType == "" {
			continue
		}

		var mType storage.MetricType
		switch metric.MType {
		case storage.Counter:
			mType = storage.Counter
			if metric.Delta == nil {
				continue
			}
		case storage.Gauge:
			mType = storage.Gauge
			if metric.Value == nil {
				continue
			}
		default:
			continue
		}

		if err := s.UpdateMetric(r.Context(), metric.ID, mType, metric.Delta, metric.Value); err != nil {
			http.Error(rw, "Failed to update metrics", http.StatusInternalServerError)
			return
		}

		delta, value, err := s.GetMetric(r.Context(), metric.ID, mType)
		if err != nil {
			http.Error(rw, "Failed to fetch updated metric", http.StatusInternalServerError)
			return
		}

		response = append(response, storage.Metric{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: delta,
			Value: value,
		})
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		http.Error(rw, "Can't encode response", http.StatusInternalServerError)
	}
}

// ipAllowed проверяет, что X-Real-IP попадает в доверенную подсеть из ENV TRUSTED_SUBNET.
func ipAllowed(r *http.Request, trustedCIDR string) bool {
	xr := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xr == "" {
		return false
	}
	ip := net.ParseIP(xr)
	if ip == nil {
		return false
	}
	_, n, err := net.ParseCIDR(trustedCIDR)
	if err != nil {
		// Неверная подсеть в конфиге — считаем запрос недоверенным
		return false
	}
	return n.Contains(ip)
}
