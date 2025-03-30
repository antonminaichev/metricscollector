package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type gauge float64
type counter int64

type Metric struct {
	name     string
	mtype    string
	value    interface{}
	getValue func(*runtime.MemStats) float64
}

var metrics = []Metric{
	{"Alloc", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.Alloc) }},
	{"BuckHashSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.BuckHashSys) }},
	{"Frees", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.Frees) }},
	{"GCCPUFraction", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return m.GCCPUFraction }},
	{"GCSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.GCSys) }},
	{"HeapAlloc", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapAlloc) }},
	{"HeapIdle", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapIdle) }},
	{"HeapInuse", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapInuse) }},
	{"HeapObjects", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapObjects) }},
	{"HeapReleased", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapReleased) }},
	{"HeapSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.HeapSys) }},
	{"LastGC", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.LastGC) }},
	{"Lookups", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.Lookups) }},
	{"MCacheInuse", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.MCacheInuse) }},
	{"MCacheSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.MCacheSys) }},
	{"MSpanInuse", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.MSpanInuse) }},
	{"MSpanSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.MSpanSys) }},
	{"Mallocs", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.Mallocs) }},
	{"NextGC", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.NextGC) }},
	{"NumForcedGC", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.NumForcedGC) }},
	{"NumGC", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.NumGC) }},
	{"OtherSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.OtherSys) }},
	{"PauseTotalNs", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.PauseTotalNs) }},
	{"StackInuse", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.StackInuse) }},
	{"StackSys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.StackSys) }},
	{"Sys", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.Sys) }},
	{"TotalAlloc", "gauge", gauge(0), func(m *runtime.MemStats) float64 { return float64(m.TotalAlloc) }},
	{"PollCount", "counter", counter(0), nil},
	{"RandomValue", "gauge", gauge(0), nil},
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
			if metrics[i].getValue != nil {
				metrics[i].value = gauge(metrics[i].getValue(&runtimeMetrics))
			} else {
				switch metrics[i].name {
				case "PollCount":
					metrics[i].value = metrics[i].value.(counter) + 1
					log.Printf("Current Poll count: %d", metrics[i].value)
				case "RandomValue":
					metrics[i].value = gauge(rand.Float64())
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
			url := fmt.Sprintf("%s/update/%s/%s/%v", host, m.mtype, m.name, m.value)

			req, err := http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				log.Printf("Error creating request for %s: %v", m.name, err)
				continue
			}
			req.Header.Set("Content-Type", "text/plain")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error sending request for %s: %v", m.name, err)
				continue
			}
			resp.Body.Close()
		}
		reportCount++
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)

	}
}
