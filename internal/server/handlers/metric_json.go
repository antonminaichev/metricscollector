package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/antonminaichev/metricscollector/internal/database"
)

type metricUpdaterJSON interface {
	UpdateCounter(name string, value int64)
	UpdateGauge(name string, value float64)
}

type metricGetterJSON interface {
	GetCounter() map[string]int64
	GetGauge() map[string]float64
}

// PostMetricJSON обновляет значение метрики через JSON-запрос
func PostMetricJSON(rw http.ResponseWriter, r *http.Request, mu metricUpdaterJSON, mg metricGetterJSON) {
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

	if database.DB != nil {
		// Не совсем понятно как использовать тут интерфейс, описывать новый storage type?
		// И приводить сигнатуры методов memstorage к UpdateMetric(...) и GetMetric(...) как сделал в package database?
		err := database.UpdateMetric(metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			http.Error(rw, "Failed to update metric in database", http.StatusInternalServerError)
			return
		}
		delta, value, err := database.GetMetric(metric.ID, metric.MType)
		if err != nil {
			http.Error(rw, "Failed to get updated metric from database", http.StatusInternalServerError)
			return
		}
		response.Delta = delta
		response.Value = value
	} else {
		switch metric.MType {
		case MetricTypeCounter:
			if metric.Delta == nil {
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			mu.UpdateCounter(metric.ID, *metric.Delta)
			if val, ok := mg.GetCounter()[response.ID]; ok {
				response.Delta = &val
			}
		case MetricTypeGauge:
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
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(response); err != nil {
		http.Error(rw, "Can`t encode response", http.StatusInternalServerError)
		return
	}
}

// GetMetricJSON возвращает значение метрики через JSON-запрос
func GetMetricJSON(rw http.ResponseWriter, r *http.Request, mg metricGetterJSON) {
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

	if database.DB != nil {
		delta, value, err := database.GetMetric(metric.ID, metric.MType)
		if err != nil {
			http.Error(rw, "Metric not found", http.StatusNotFound)
			return
		}
		response.Delta = delta
		response.Value = value
	} else {
		switch metric.MType {
		case MetricTypeGauge:
			metrics := mg.GetGauge()
			value, ok := metrics[metric.ID]
			if !ok {
				http.Error(rw, "No such gauge metric "+metric.ID, http.StatusNotFound)
				return
			}
			response.Value = &value
		case MetricTypeCounter:
			metrics := mg.GetCounter()
			value, ok := metrics[metric.ID]
			if !ok {
				http.Error(rw, "No such counter metric "+metric.ID, http.StatusNotFound)
				return
			}
			response.Delta = &value
		default:
			http.Error(rw, "No such metric type "+metric.ID, http.StatusNotFound)
			return
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(response); err != nil {
		http.Error(rw, "Can`t encode response", http.StatusInternalServerError)
		return
	}
}
