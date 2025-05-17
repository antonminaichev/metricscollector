package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/antonminaichev/metricscollector/internal/retry"
)

type Metrics struct {
	ID       string   `json:"id"`              // имя метрики
	MType    string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta    *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value    *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	getValue func(*runtime.MemStats) float64
}

var metrics = []Metrics{
	{"Alloc", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.Alloc) }},
	{"BuckHashSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.BuckHashSys) }},
	{"Frees", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.Frees) }},
	{"GCCPUFraction", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return m.GCCPUFraction }},
	{"GCSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.GCSys) }},
	{"HeapAlloc", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapAlloc) }},
	{"HeapIdle", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapIdle) }},
	{"HeapInuse", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) }},
	{"HeapObjects", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapObjects) }},
	{"HeapReleased", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapReleased) }},
	{"HeapSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.HeapSys) }},
	{"LastGC", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.LastGC) }},
	{"Lookups", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.Lookups) }},
	{"MCacheInuse", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.MCacheInuse) }},
	{"MCacheSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.MCacheSys) }},
	{"MSpanInuse", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.MSpanInuse) }},
	{"MSpanSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.MSpanSys) }},
	{"Mallocs", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.Mallocs) }},
	{"NextGC", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.NextGC) }},
	{"NumForcedGC", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.NumForcedGC) }},
	{"NumGC", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.NumGC) }},
	{"OtherSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.OtherSys) }},
	{"PauseTotalNs", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) }},
	{"StackInuse", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.StackInuse) }},
	{"StackSys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.StackSys) }},
	{"Sys", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.Sys) }},
	{"TotalAlloc", "gauge", nil, nil, func(m *runtime.MemStats) float64 { return float64(m.TotalAlloc) }},
	{"PollCount", "counter", new(int64), nil, nil},
	{"RandomValue", "gauge", nil, nil, nil},
}

// checkServerAvailability is used for checking server availability
func checkServerAvailability(host string) bool {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}

	err := retry.Do(retry.DefaultRetryConfig(), func() error {
		resp, err := http.Get(host + "/health")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned status code %d", resp.StatusCode)
		}
		return nil
	})

	if err != nil {
		log.Printf("Server availability check failed: %v", err)
		return false
	}
	return true
}

// CollectMetrics is used metric collection
func CollectMetrics(pollInterval int) {
	log.Printf("Poll interval: %d sec", pollInterval)

	var runtimeMetrics runtime.MemStats
	for {
		runtime.ReadMemStats(&runtimeMetrics)

		for i := range metrics {
			m := &metrics[i]
			switch m.MType {
			case "gauge":
				if m.getValue != nil {
					val := m.getValue(&runtimeMetrics)
					m.Value = &val
				} else if m.ID == "RandomValue" {
					val := rand.Float64()
					m.Value = &val
				}
			case "counter":
				if m.ID == "PollCount" && m.Delta != nil {
					*m.Delta++
				}
			}
		}
		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
}

// PostMetric is used for sending metrics to server
func PostMetric(client *http.Client, reportInterval int, host string) {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	log.Printf("Report Interval: %d sec", reportInterval)
	log.Printf("Host: %s", host)

	for !checkServerAvailability(host) {
		log.Printf("Server unreachable, retry in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	log.Printf("Server %s is reachable", host)

	reportCount := 0

	for {
		for _, m := range metrics {
			var valueStr string
			switch m.MType {
			case "gauge":
				if m.Value == nil {
					continue
				}
				valueStr = fmt.Sprintf("%f", *m.Value)
			case "counter":
				if m.Delta == nil {
					continue
				}
				valueStr = fmt.Sprintf("%d", *m.Delta)
			default:
				continue
			}

			url := fmt.Sprintf("%s/update/%s/%s/%s", host, m.MType, m.ID, valueStr)
			req, err := http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				log.Printf("Error creating request for %s: %v", m.ID, err)
				continue
			}
			req.Header.Set("Content-Type", "text/plain")

			err = retry.Do(retry.DefaultRetryConfig(), func() error {
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				return nil
			})
			if err != nil {
				log.Printf("Error sending request for %s after retries: %v", m.ID, err)
				continue
			}
		}
		reportCount++
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}

// PostMetricJSON is used for sending metrics to server via JSON request
func PostMetricJSON(client *http.Client, reportInterval int, host string) {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	log.Printf("Report Interval: %d sec", reportInterval)
	log.Printf("Host: %s", host)

	for !checkServerAvailability(host) {
		log.Printf("Server unreachable, retry in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	log.Printf("Server %s is reachable", host)

	reportCount := 0

	for {
		for _, m := range metrics {
			url := fmt.Sprintf("%s/update", host)
			jsonBody, err := json.Marshal(m)
			if err != nil {
				log.Printf("Error encoding JSON for %s: %v", m.ID, err)
				continue
			}
			buf := bytes.NewBuffer(nil)
			zb, err := gzip.NewWriterLevel(buf, gzip.BestSpeed)
			if err != nil {
				log.Printf("Unable to create gzip writer")
				continue
			}
			_, err = zb.Write(jsonBody)
			if err != nil {
				log.Printf("Unable to zip data")
				continue
			}
			err = zb.Close()
			if err != nil {
				log.Printf("Unable to close zip writer")
				continue
			}
			req, err := http.NewRequest(http.MethodPost, url, buf)
			if err != nil {
				log.Printf("Error creating request for %s: %v", m.ID, err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Content-Encoding", "gzip")

			err = retry.Do(retry.DefaultRetryConfig(), func() error {
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				return nil
			})
			if err != nil {
				log.Printf("Error sending request for %s after retries: %v", m.ID, err)
				continue
			}
		}

		reportCount++
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}

// PostMetricsBatch is used for sending metrics to server via JSON request by batches
func PostMetricsBatch(client *http.Client, reportInterval int, host string) {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	log.Printf("Report Interval: %d sec", reportInterval)
	log.Printf("Host: %s", host)

	for !checkServerAvailability(host) {
		log.Printf("Server unreachable, retry in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	log.Printf("Server %s is reachable", host)

	reportCount := 0

	for {
		var metricsBatch []Metrics
		for _, m := range metrics {
			metric := Metrics{
				ID:    m.ID,
				MType: m.MType,
			}
			if m.MType == "counter" && m.Delta != nil {
				metric.Delta = m.Delta
			} else if m.MType == "gauge" && m.Value != nil {
				metric.Value = m.Value
			}
			metricsBatch = append(metricsBatch, metric)
		}

		if len(metricsBatch) == 0 {
			log.Printf("No metrics to send, skipping batch")
			time.Sleep(time.Duration(reportInterval) * time.Second)
			continue
		}

		url := fmt.Sprintf("%s/updates/", host)
		jsonBody, err := json.Marshal(metricsBatch)
		if err != nil {
			log.Printf("Error encoding JSON batch: %v", err)
			continue
		}

		buf := bytes.NewBuffer(nil)
		zb, err := gzip.NewWriterLevel(buf, gzip.BestSpeed)
		if err != nil {
			log.Printf("Unable to create gzip writer: %v", err)
			continue
		}
		_, err = zb.Write(jsonBody)
		if err != nil {
			log.Printf("Unable to zip data: %v", err)
			continue
		}
		err = zb.Close()
		if err != nil {
			log.Printf("Unable to close zip writer: %v", err)
			continue
		}

		req, err := http.NewRequest(http.MethodPost, url, buf)
		if err != nil {
			log.Printf("Error creating batch request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		err = retry.Do(retry.DefaultRetryConfig(), func() error {
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			return nil
		})
		if err != nil {
			log.Printf("Error sending batch request after retries: %v", err)
			continue
		}

		reportCount++
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}
