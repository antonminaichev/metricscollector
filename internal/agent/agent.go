package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strings"
	"time"
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
	resp, err := http.Get(host + "/health")
	if err != nil {
		log.Printf("Something went wrong: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
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

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error sending request for %s: %v", m.ID, err)
				continue
			}
			resp.Body.Close()
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

			req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(jsonBody)))
			if err != nil {
				log.Printf("Error creating request for %s: %v", m.ID, err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error sending request for %s: %v", m.ID, err)
				continue
			}
			resp.Body.Close()
		}

		reportCount++
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)

	}
}
