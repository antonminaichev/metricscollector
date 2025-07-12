package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/antonminaichev/metricscollector/internal/server/storage"
	memstorage "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"
)

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }

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

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Delta: %d\n",
		w.Result().StatusCode, response.ID, response.MType, *response.Delta)

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

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Value: %.2f\n",
		w.Result().StatusCode, response.ID, response.MType, *response.Value)

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

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Delta: %d\n",
		w.Result().StatusCode, response.ID, response.MType, *response.Delta)

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

	var response storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, ID: %s, Type: %s, Value: %.2f\n",
		w.Result().StatusCode, response.ID, response.MType, *response.Value)

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

	var response []storage.Metric
	_ = json.NewDecoder(w.Body).Decode(&response)

	fmt.Printf("Status: %d, count: %d, first: %s, second: %s\n",
		w.Result().StatusCode,
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

	fmt.Printf("Status: %d, Body: %s\n", w.Result().StatusCode, w.Body.String())

	// Output: Status: 200, Body: {"status": "ok"}
}
