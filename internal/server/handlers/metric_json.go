package handlers

import (
	"database/sql"
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
		if metric.MType == MetricTypeCounter && metric.Delta != nil {
			query := `
				INSERT INTO metrics (id, type, delta, value)
				VALUES ($1, $2, $3, NULL)
				ON CONFLICT (id, type) DO UPDATE
				SET delta = metrics.delta + EXCLUDED.delta`
			_, err := database.DB.Exec(query, metric.ID, metric.MType, *metric.Delta)
			if err != nil {
				http.Error(rw, "Failed to update counter in database", http.StatusInternalServerError)
				return
			}
		} else {
			err := database.UpdateMetric(metric.ID, metric.MType, metric.Delta, metric.Value)
			if err != nil {
				http.Error(rw, "Failed to update metric in database", http.StatusInternalServerError)
				return
			}
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
			currentValue := mg.GetCounter()[response.ID]
			mu.UpdateCounter(metric.ID, currentValue+*metric.Delta)
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

// PostMetricsJSON обновляет множество метрик через JSON-запрос
func PostMetricsJSON(rw http.ResponseWriter, r *http.Request, mu metricUpdaterJSON, mg metricGetterJSON) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var metrics []Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	for _, metric := range metrics {
		if metric.ID == "" || metric.MType == "" {
			http.Error(rw, "Invalid metric format", http.StatusBadRequest)
			return
		}
	}

	var responses []Metrics

	if database.DB != nil {
		tx, err := database.DB.Begin()
		if err != nil {
			http.Error(rw, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		for _, metric := range metrics {
			if metric.MType == MetricTypeCounter {
				query := `
					INSERT INTO metrics (id, type, delta, value)
					VALUES ($1, $2, $3, NULL)
					ON CONFLICT (id, type) DO UPDATE
					SET delta = metrics.delta + EXCLUDED.delta`
				_, err := tx.Exec(query, metric.ID, metric.MType, *metric.Delta)
				if err != nil {
					http.Error(rw, "Failed to update counter in database", http.StatusInternalServerError)
					return
				}
			} else {
				query := `
					INSERT INTO metrics (id, type, delta, value)
					VALUES ($1, $2, NULL, $3)
					ON CONFLICT (id, type) DO UPDATE
					SET value = EXCLUDED.value`
				_, err := tx.Exec(query, metric.ID, metric.MType, *metric.Value)
				if err != nil {
					http.Error(rw, "Failed to update gauge in database", http.StatusInternalServerError)
					return
				}
			}
		}

		for _, metric := range metrics {
			var delta sql.NullInt64
			var value sql.NullFloat64
			err := tx.QueryRow("SELECT delta, value FROM metrics WHERE id = $1 AND type = $2",
				metric.ID, metric.MType).Scan(&delta, &value)
			if err != nil {
				http.Error(rw, "Failed to get updated metric from database", http.StatusInternalServerError)
				return
			}

			response := Metrics{
				ID:    metric.ID,
				MType: metric.MType,
			}
			if delta.Valid {
				response.Delta = &delta.Int64
			}
			if value.Valid {
				response.Value = &value.Float64
			}
			responses = append(responses, response)
		}

		if err := tx.Commit(); err != nil {
			http.Error(rw, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
	} else {
		for _, metric := range metrics {
			var response Metrics
			response.ID = metric.ID
			response.MType = metric.MType

			switch metric.MType {
			case MetricTypeCounter:
				if metric.Delta == nil {
					http.Error(rw, "Invalid counter metric", http.StatusBadRequest)
					return
				}
				mu.UpdateCounter(metric.ID, *metric.Delta)
				if val, ok := mg.GetCounter()[response.ID]; ok {
					response.Delta = &val
				}
			case MetricTypeGauge:
				if metric.Value == nil {
					http.Error(rw, "Invalid gauge metric", http.StatusBadRequest)
					return
				}
				mu.UpdateGauge(metric.ID, *metric.Value)
				if val, ok := mg.GetGauge()[response.ID]; ok {
					response.Value = &val
				}
			default:
				http.Error(rw, "Invalid metric type", http.StatusBadRequest)
				return
			}
			responses = append(responses, response)
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(responses); err != nil {
		http.Error(rw, "Can`t encode response", http.StatusInternalServerError)
		return
	}
}
