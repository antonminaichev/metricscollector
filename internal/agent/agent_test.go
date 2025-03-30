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
		expectedChecks func(t *testing.T, metrics []Metric)
	}{
		{
			name:         "Сбор всех метрик",
			pollInterval: 2,
			expectedChecks: func(t *testing.T, metrics []Metric) {
				findMetric := func(name string) *Metric {
					for i := range metrics {
						if metrics[i].name == name {
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
					require.NotNil(t, metric.value, "Значение метрики %s не установлено", name)
				}

				pollCount := findMetric("PollCount")
				require.NotNil(t, pollCount)
				require.Greater(t, pollCount.value.(counter), counter(0))

				randomValue := findMetric("RandomValue")
				require.NotNil(t, randomValue)
				require.GreaterOrEqual(t, randomValue.value.(gauge), gauge(0))
				require.LessOrEqual(t, randomValue.value.(gauge), gauge(1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем значения метрик перед тестом
			for i := range metrics {
				switch metrics[i].mtype {
				case "gauge":
					metrics[i].value = gauge(0)
				case "counter":
					metrics[i].value = counter(0)
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
