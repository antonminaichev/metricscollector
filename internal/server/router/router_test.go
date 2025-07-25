package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/logger"
	"github.com/antonminaichev/metricscollector/internal/server/middleware"
	st "github.com/antonminaichev/metricscollector/internal/server/storage"
	fs "github.com/antonminaichev/metricscollector/internal/server/storage/file"
	ms "github.com/antonminaichev/metricscollector/internal/server/storage/memstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestHealthCheck(t *testing.T) {
	storage := ms.NewMemoryStorage()
	ts := httptest.NewServer(NewRouter(storage))
	defer ts.Close()
	var testTable = []struct {
		url    string
		want   string
		status int
	}{
		{"/health", `{"status": "ok"}`, http.StatusOK},
		{"/health1", "404 page not found\n", http.StatusNotFound},
	}
	for _, v := range testTable {
		resp, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, resp.StatusCode)
		assert.Equal(t, v.want, get)
		resp.Body.Close()
	}
}

func TestPostMetric(t *testing.T) {
	storage := ms.NewMemoryStorage()
	ts := httptest.NewServer(NewRouter(storage))
	defer ts.Close()

	testTable := []struct {
		name       string
		url        string
		want       int
		metricType string
		value      interface{}
	}{
		{
			name:       "Positive counter",
			url:        "/update/counter/testCounter/100",
			want:       http.StatusOK,
			metricType: "counter",
			value:      int64(100),
		},
		{
			name:       "Positive gauge",
			url:        "/update/gauge/testGauge/123.45",
			want:       http.StatusOK,
			metricType: "gauge",
			value:      float64(123.45),
		},
		{
			name: "Incorrect metric type",
			url:  "/update/unknown/testMetric/100",
			want: http.StatusBadRequest,
		},
		{
			name: "Incorrect value counter",
			url:  "/update/counter/testCounter/abc",
			want: http.StatusBadRequest,
		},
		{
			name: "Incorrect value gauge",
			url:  "/update/gauge/testGauge/abc",
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, http.MethodPost, tt.url)
			assert.Equal(t, tt.want, resp.StatusCode)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				switch tt.metricType {
				case "counter":
					delta, _, _ := storage.GetMetric(context.Background(), "testCounter", st.Counter)
					assert.Equal(t, tt.value, *delta)
				case "gauge":
					_, value, _ := storage.GetMetric(context.Background(), "testGauge", st.Gauge)
					assert.Equal(t, tt.value, *value)
				}
			}
		})
	}
}

func TestGetMetric(t *testing.T) {
	storage := ms.NewMemoryStorage()
	ts := httptest.NewServer(NewRouter(storage))
	defer ts.Close()

	delta := int64(100)
	value := float64(123.45)
	storage.UpdateMetric(context.Background(), "testCounter", st.Counter, &delta, nil)
	storage.UpdateMetric(context.Background(), "testGauge", st.Gauge, nil, &value)

	testTable := []struct {
		name       string
		url        string
		want       int
		metricType string
		value      interface{}
	}{
		{
			name:       "Positive counter",
			url:        "/value/counter/testCounter",
			want:       http.StatusOK,
			metricType: "counter",
			value:      int64(100),
		},
		{
			name:       "Positive gauge",
			url:        "/value/gauge/testGauge",
			want:       http.StatusOK,
			metricType: "gauge",
			value:      float64(123.45),
		},
		{
			name: "Incorrect metric type",
			url:  "/value/unknown/testMetric",
			want: http.StatusNotFound,
		},
		{
			name: "Counter metric not found",
			url:  "/value/counter/nonexistent",
			want: http.StatusNotFound,
		},
		{
			name: "Gauge metric not found",
			url:  "/value/gauge/nonexistent",
			want: http.StatusNotFound,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, http.MethodGet, tt.url)
			assert.Equal(t, tt.want, resp.StatusCode)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				switch tt.metricType {
				case "counter":
					value, _ := strconv.ParseInt(body, 10, 64)
					assert.Equal(t, tt.value, value)
				case "gauge":
					value, _ := strconv.ParseFloat(body, 64)
					assert.Equal(t, tt.value, value)
				}
			}
		})
	}
}

func TestPrintAllMetrics(t *testing.T) {
	storage := ms.NewMemoryStorage()
	ts := httptest.NewServer(NewRouter(storage))
	defer ts.Close()

	delta := int64(52)
	value := float64(5432.21234)
	storage.UpdateMetric(context.Background(), "testCounter", st.Counter, &delta, nil)
	storage.UpdateMetric(context.Background(), "testGauge", st.Gauge, nil, &value)

	resp, body := testRequest(t, ts, http.MethodGet, "/")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	assert.Contains(t, body, "testCounter")
	assert.Contains(t, body, "52")
	assert.Contains(t, body, "testGauge")
	assert.Contains(t, body, "5432.21234")
}

func BenchmarkServer_FileStorageUpdate(b *testing.B) {
	// создаем временный файл для хранения метрик
	tmpFile := filepath.Join(os.TempDir(), "metrics_benchmark.json")
	_ = os.Remove(tmpFile)

	_ = logger.Initialize("ERROR")
	store, err := fs.NewFileStorage(tmpFile, logger.Log)
	if err != nil {
		b.Fatalf("failed to create file storage: %v", err)
	}

	handler := middleware.GzipHandler(NewRouter(store))

	metric := st.Metric{
		ID:    "BenchmarkGauge",
		MType: st.Gauge,
		Value: newFloat64(999.999),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body, _ := json.Marshal(metric)

		var compressed bytes.Buffer
		gzw := gzip.NewWriter(&compressed)
		gzw.Write(body)
		gzw.Close()

		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(compressed.Bytes()))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
	_ = os.Remove(tmpFile)
}

func newFloat64(v float64) *float64 {
	return &v
}
