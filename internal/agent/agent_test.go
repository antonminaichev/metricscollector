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
		expectedChecks func(t *testing.T, metrics []Metrics)
	}{
		{
			name:         "Сбор всех метрик",
			pollInterval: 2,
			expectedChecks: func(t *testing.T, metrics []Metrics) {
				findMetric := func(name string) *Metrics {
					for i := range metrics {
						if metrics[i].ID == name {
							return &metrics[i]
						}
					}
					return nil
				}

				for _, name := range []string{
					"BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
					"HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
					"HeapSys", "LastGC", "Lookups", "MCacheInuse",
					"MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs",
					"NextGC", "NumForcedGC", "NumGC", "OtherSys",
					"PauseTotalNs", "StackInuse", "StackSys", "Sys",
					"TotalAlloc", "HeapAlloc", "Alloc",
				} {
					metric := findMetric(name)
					require.NotNil(t, metric, "Метрика %s не найдена", name)
					require.NotNil(t, metric.Value, "Значение метрики %s не установлено", name)
				}

				// Проверка PollCount
				pollCount := findMetric("PollCount")
				require.NotNil(t, pollCount)
				require.Equal(t, "counter", pollCount.MType)
				require.NotNil(t, pollCount.Delta)
				require.Greater(t, *pollCount.Delta, int64(0))

				// Проверка RandomValue
				randomValue := findMetric("RandomValue")
				require.NotNil(t, randomValue)
				require.Equal(t, "gauge", randomValue.MType)
				require.NotNil(t, randomValue.Value)
				require.GreaterOrEqual(t, *randomValue.Value, 0.0)
				require.LessOrEqual(t, *randomValue.Value, 1.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс значений перед тестом
			for i := range metrics {
				switch metrics[i].MType {
				case "gauge":
					metrics[i].Value = nil
				case "counter":
					metrics[i].Delta = new(int64)
					*metrics[i].Delta = 0
				}
			}

			done := make(chan bool)
			go func() {
				CollectMetrics(tt.pollInterval)
				done <- true
			}()
			time.Sleep(6 * time.Second)
			tt.expectedChecks(t, metrics)
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
