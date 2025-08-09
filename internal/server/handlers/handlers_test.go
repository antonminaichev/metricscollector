package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	memstorage "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }

// Test for PostMetricJSON
func TestPostMetricJSON(t *testing.T) {
	store := memstorage.NewMemoryStorage()

	t.Run("successful counter update", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_counter",
			MType: storage.Counter,
			Delta: ptrInt64(42),
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response storage.Metric
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test_counter", response.ID)
		assert.Equal(t, storage.Counter, response.MType)
		assert.Equal(t, int64(42), *response.Delta)
	})

	t.Run("successful gauge update", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_gauge",
			MType: storage.Gauge,
			Value: ptrFloat64(3.14),
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)

		var response storage.Metric
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test_gauge", response.ID)
		assert.Equal(t, storage.Gauge, response.MType)
		assert.Equal(t, 3.14, *response.Value)
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/update/", nil)
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing ID", func(t *testing.T) {
		metric := storage.Metric{
			MType: storage.Counter,
			Delta: ptrInt64(42),
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing MType", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test",
			Delta: ptrInt64(42),
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("counter without delta", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_counter",
			MType: storage.Counter,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("gauge without value", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_gauge",
			MType: storage.Gauge,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown metric type", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test",
			MType: storage.MetricType("unknown"),
			Delta: ptrInt64(42),
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test for GetMetricJSON
func TestGetMetricJSON(t *testing.T) {
	store := memstorage.NewMemoryStorage()
	ctx := context.Background()

	// Подготавливаем данные
	_ = store.UpdateMetric(ctx, "test_counter", storage.Counter, ptrInt64(100), nil)
	_ = store.UpdateMetric(ctx, "test_gauge", storage.Gauge, nil, ptrFloat64(99.9))

	t.Run("get counter", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_counter",
			MType: storage.Counter,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		GetMetricJSON(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)

		var response storage.Metric
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test_counter", response.ID)
		assert.Equal(t, storage.Counter, response.MType)
		assert.Equal(t, int64(100), *response.Delta)
	})

	t.Run("get gauge", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "test_gauge",
			MType: storage.Gauge,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		GetMetricJSON(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)

		var response storage.Metric
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test_gauge", response.ID)
		assert.Equal(t, storage.Gauge, response.MType)
		assert.Equal(t, 99.9, *response.Value)
	})

	t.Run("metric not found", func(t *testing.T) {
		metric := storage.Metric{
			ID:    "nonexistent",
			MType: storage.Counter,
		}
		body, _ := json.Marshal(metric)

		req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		GetMetricJSON(w, req, store)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader("invalid"))
		w := httptest.NewRecorder()
		GetMetricJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test for PostMetricsJSON
func TestPostMetricsJSON(t *testing.T) {
	store := memstorage.NewMemoryStorage()

	t.Run("successful batch update", func(t *testing.T) {
		metrics := []storage.Metric{
			{ID: "counter1", MType: storage.Counter, Delta: ptrInt64(10)},
			{ID: "gauge1", MType: storage.Gauge, Value: ptrFloat64(1.5)},
		}
		body, _ := json.Marshal(metrics)

		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		PostMetricsJSON(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []storage.Metric
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader("invalid"))
		w := httptest.NewRecorder()
		PostMetricsJSON(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test for PostMetric (text handler)
func TestPostMetric(t *testing.T) {
	store := memstorage.NewMemoryStorage()

	t.Run("successful counter update", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/test_counter/42", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("metric", "test_counter")
		rctx.URLParams.Add("value", "42")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	})

	t.Run("successful gauge update", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/test_gauge/3.14", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "gauge")
		rctx.URLParams.Add("metric", "test_gauge")
		rctx.URLParams.Add("value", "3.14")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/update/counter/test/42", nil)
		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("missing parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/test/42", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "")
		rctx.URLParams.Add("metric", "test")
		rctx.URLParams.Add("value", "42")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid counter value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/test/invalid", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("metric", "test")
		rctx.URLParams.Add("value", "invalid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid gauge value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/invalid", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "gauge")
		rctx.URLParams.Add("metric", "test")
		rctx.URLParams.Add("value", "invalid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown metric type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/unknown/test/42", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "unknown")
		rctx.URLParams.Add("metric", "test")
		rctx.URLParams.Add("value", "42")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		PostMetric(w, req, store)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// Test for GetMetric (text handler)
func TestGetMetric(t *testing.T) {
	store := memstorage.NewMemoryStorage()
	ctx := context.Background()

	// Подготавливаем данные
	_ = store.UpdateMetric(ctx, "test_counter", storage.Counter, ptrInt64(100), nil)
	_ = store.UpdateMetric(ctx, "test_gauge", storage.Gauge, nil, ptrFloat64(99.9))

	t.Run("get counter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/counter/test_counter", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("metric", "test_counter")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		GetMetric(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "100", w.Body.String())
	})

	t.Run("get gauge", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/gauge/test_gauge", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "gauge")
		rctx.URLParams.Add("metric", "test_gauge")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		GetMetric(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "99.9", w.Body.String())
	})

	t.Run("unknown metric type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/unknown/test", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "unknown")
		rctx.URLParams.Add("metric", "test")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		GetMetric(w, req, store)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("metric not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/counter/nonexistent", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "counter")
		rctx.URLParams.Add("metric", "nonexistent")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		GetMetric(w, req, store)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// Test for HealthCheck
func TestHealthCheck(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		HealthCheck(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, `{"status": "ok"}`, w.Body.String())
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/ping", nil)
		w := httptest.NewRecorder()

		HealthCheck(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// Test for PrintAllMetrics
func TestPrintAllMetrics(t *testing.T) {
	store := memstorage.NewMemoryStorage()
	ctx := context.Background()

	t.Run("prints all metrics", func(t *testing.T) {
		_ = store.UpdateMetric(ctx, "counter1", storage.Counter, ptrInt64(10), nil)
		_ = store.UpdateMetric(ctx, "gauge1", storage.Gauge, nil, ptrFloat64(1.5))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		PrintAllMetrics(w, req, store)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

		body := w.Body.String()
		assert.Contains(t, body, "counter1")
		assert.Contains(t, body, "gauge1")
		assert.Contains(t, body, "10")
		assert.Contains(t, body, "1.5")
		assert.Contains(t, body, "<html>")
		assert.Contains(t, body, "</html>")
	})

	t.Run("empty metrics", func(t *testing.T) {
		emptyStore := memstorage.NewMemoryStorage()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		PrintAllMetrics(w, req, emptyStore)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "<html>")
		assert.Contains(t, body, "Metrics")
	})
}

// Examples (keeping existing ones)

func ExamplePostMetricJSON() {
	s := memstorage.NewMemoryStorage()

	m := storage.Metric{
		ID:    "myCounter",
		MType: storage.Counter,
		Delta: ptrInt64(42),
	}
	body, _ := json.Marshal(m)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	PostMetricJSON(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Delta: %d\n",
		resp.StatusCode, response.ID, response.MType, *response.Delta)

	// Output: Status: 200, ID: myCounter, Type: counter, Delta: 42
}

func ExampleGetMetricJSON() {
	s := memstorage.NewMemoryStorage()
	ctx := context.Background()
	_ = s.UpdateMetric(ctx, "myGauge", storage.Gauge, nil, ptrFloat64(3.14))

	m := storage.Metric{
		ID:    "myGauge",
		MType: storage.Gauge,
	}
	body, _ := json.Marshal(m)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	GetMetricJSON(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Value: %.2f\n",
		resp.StatusCode, response.ID, response.MType, *response.Value)

	// Output: Status: 200, ID: myGauge, Type: gauge, Value: 3.14
}

func ExamplePostMetric() {
	s := memstorage.NewMemoryStorage()

	m := storage.Metric{
		ID:    "myCounter",
		MType: storage.Counter,
		Delta: ptrInt64(42),
	}
	body, _ := json.Marshal(m)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	PostMetricJSON(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Delta: %d\n",
		resp.StatusCode, response.ID, response.MType, *response.Delta)

	// Output: Status: 200, ID: myCounter, Type: counter, Delta: 42
}

func ExampleGetMetric() {
	s := memstorage.NewMemoryStorage()
	ctx := context.Background()
	_ = s.UpdateMetric(ctx, "myGauge", storage.Gauge, nil, ptrFloat64(3.14))

	m := storage.Metric{
		ID:    "myGauge",
		MType: storage.Gauge,
	}
	body, _ := json.Marshal(m)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	GetMetricJSON(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Value: %.2f\n",
		resp.StatusCode, response.ID, response.MType, *response.Value)

	// Output: Status: 200, ID: myGauge, Type: gauge, Value: 3.14
}

func ExamplePostMetricsJSON() {
	s := memstorage.NewMemoryStorage()

	metrics := []storage.Metric{
		{ID: "cpu", MType: storage.Gauge, Value: ptrFloat64(0.9)},
		{ID: "reqs", MType: storage.Counter, Delta: ptrInt64(7)},
	}
	body, _ := json.Marshal(metrics)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	PostMetricsJSON(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	var response []storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, count: %d, first: %s, second: %s\n",
		resp.StatusCode,
		len(response),
		response[0].ID,
		response[1].ID)

	// Output: Status: 200, count: 2, first: cpu, second: reqs
}

func ExamplePrintAllMetrics() {
	s := memstorage.NewMemoryStorage()
	ctx := context.Background()
	_ = s.UpdateMetric(ctx, "disk_free", storage.Gauge, nil, ptrFloat64(128.5))
	_ = s.UpdateMetric(ctx, "http_hits", storage.Counter, ptrInt64(1024), nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	PrintAllMetrics(w, req, s)
	resp := w.Result()
	defer resp.Body.Close()

	body := w.Body.String()
	fmt.Println("Contains disk_free:", strings.Contains(body, "disk_free"))
	fmt.Println("Contains http_hits:", strings.Contains(body, "http_hits"))

	// Output:
	// Contains disk_free: true
	// Contains http_hits: true
}

func ExampleHealthCheck() {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	HealthCheck(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	fmt.Printf("Status: %d, Body: %s\n", resp.StatusCode, w.Body.String())

	// Output: Status: 200, Body: {"status": "ok"}
}
