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
	pollInterval := 1
	cycles := 2

	// Reset global metrics state
	for i := range metrics {
		switch metrics[i].MType {
		case "gauge":
			metrics[i].Value = nil
		case "counter":
			metrics[i].Delta = new(int64)
			*metrics[i].Delta = 0
		}
	}

	jobs := make(chan Metrics, len(metrics)*cycles)
	defer close(jobs)
	go CollectMetrics(pollInterval, jobs)

	// Allow some metric collection
	time.Sleep(time.Duration(pollInterval*cycles) * time.Second)

	// Drain jobs into map
	collected := make(map[string]Metrics)
	for i := 0; i < len(metrics)*cycles; i++ {
		select {
		case m := <-jobs:
			collected[m.ID] = m
		default:
			// no more metrics
		}
	}

	// Check runtime gauges
	for _, name := range []string{"Alloc", "HeapAlloc", "TotalAlloc", "Sys"} {
		m, ok := collected[name]
		require.True(t, ok, "metric %s should be collected", name)
		require.Equal(t, "gauge", m.MType)
		require.NotNil(t, m.Value, "metric %s should have a value", name)
		require.Greater(t, *m.Value, float64(0), "metric %s should be > 0", name)
	}

	// Check PollCount counter
	poll, ok := collected["PollCount"]
	require.True(t, ok, "PollCount should be collected")
	require.Equal(t, "counter", poll.MType)
	require.NotNil(t, poll.Delta, "PollCount should have a delta")
	require.Greater(t, *poll.Delta, int64(0), "PollCount should be incremented")

	// Check RandomValue
	randVal, ok := collected["RandomValue"]
	require.True(t, ok, "RandomValue should be collected")
	require.Equal(t, "gauge", randVal.MType)
	require.NotNil(t, randVal.Value, "RandomValue should have a value")
	require.GreaterOrEqual(t, *randVal.Value, float64(0), "RandomValue >= 0")
	require.LessOrEqual(t, *randVal.Value, float64(1), "RandomValue <= 1")
}

func TestPostMetric(t *testing.T) {
	reportInterval := 1

	reqCh := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/update/") {
			reqCh <- r
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}

	// Start PostMetric in background
	go PostMetric(client, reportInterval, server.URL)

	select {
	case req := <-reqCh:
		require.Equal(t, http.MethodPost, req.Method)
		parts := strings.Split(req.URL.Path, "/")
		require.Len(t, parts, 5, "expected path /update/{type}/{id}/{value}")
		require.Equal(t, "update", parts[1])
		require.Contains(t, []string{"gauge", "counter"}, parts[2], "metric type should be gauge or counter")
		require.NotEmpty(t, parts[3], "metric ID should not be empty")
		require.NotEmpty(t, parts[4], "metric value should not be empty")
	case <-time.After(time.Duration(reportInterval+2) * time.Second):
		t.Fatalf("timeout waiting for PostMetric request")
	}
}
