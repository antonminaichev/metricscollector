package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {
	tests := []struct {
		name           string
		pollInterval   int
		expectedChecks func(t *testing.T, metrics Metrics)
	}{
		{
			name:         "Сбор всех метрик",
			pollInterval: 2,
			expectedChecks: func(t *testing.T, metrics Metrics) {
				require.NotNil(t, metrics.BuckHashSys, gauge(0))
				require.NotNil(t, metrics.Frees, gauge(0))
				require.NotNil(t, metrics.GCCPUFraction)
				require.NotNil(t, metrics.GCSys, gauge(0))
				require.NotNil(t, metrics.HeapIdle, gauge(0))
				require.NotNil(t, metrics.HeapInuse, gauge(0))
				require.NotNil(t, metrics.HeapObjects, gauge(0))
				require.NotNil(t, metrics.HeapReleased, gauge(0))
				require.NotNil(t, metrics.HeapSys, gauge(0))
				require.NotNil(t, metrics.LastGC, gauge(0))
				require.NotNil(t, metrics.Lookups, gauge(0))
				require.NotNil(t, metrics.MCacheInuse, gauge(0))
				require.NotNil(t, metrics.MCacheSys, gauge(0))
				require.NotNil(t, metrics.MSpanInuse, gauge(0))
				require.NotNil(t, metrics.MSpanSys, gauge(0))
				require.NotNil(t, metrics.Mallocs, gauge(0))
				require.NotNil(t, metrics.NextGC, gauge(0))
				require.NotNil(t, metrics.NumForcedGC, gauge(0))
				require.NotNil(t, metrics.NumGC, gauge(0))
				require.NotNil(t, metrics.OtherSys, gauge(0))
				require.NotNil(t, metrics.PauseTotalNs, gauge(0))
				require.NotNil(t, metrics.StackInuse, gauge(0))
				require.NotNil(t, metrics.StackSys, gauge(0))
				require.NotNil(t, metrics.Sys, gauge(0))
				require.NotNil(t, metrics.TotalAlloc, gauge(0))
				require.Greater(t, metrics.PollCount, counter(0))
				require.Greater(t, metrics.HeapAlloc, gauge(0))
				require.Greater(t, metrics.Alloc, gauge(0))
				require.GreaterOrEqual(t, metrics.RandomValue, gauge(0))
				require.LessOrEqual(t, metrics.RandomValue, gauge(1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMetrics = Metrics{}
			done := make(chan bool)
			go func() {
				CollectMetrics(tt.pollInterval)
				done <- true
			}()
			time.Sleep(6 * time.Second)
			tt.expectedChecks(t, actualMetrics)
		})
	}
}

func TestPostMetric(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		reportInterval int
		expectedChecks func(t *testing.T, request *http.Request)
	}{
		{
			name: "Succeful send metrics",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			reportInterval: 1,
			expectedChecks: func(t *testing.T, request *http.Request) {
				require.Equal(t, http.MethodPost, request.Method)
				require.Contains(t, request.URL.Path, "/update/")
				require.Contains(t, []string{"gauge", "counter"}, strings.Split(request.URL.Path, "/")[2])
				require.NotEmpty(t, strings.Split(request.URL.Path, "/")[3])
				require.NotEmpty(t, strings.Split(request.URL.Path, "/")[4])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &http.Client{}

			// Запускаем отправку метрик
			done := make(chan bool)
			go func() {
				PostMetric(client, tt.reportInterval, server.URL)
				done <- true
			}()
			time.Sleep(3 * time.Second)
		})
	}
}
