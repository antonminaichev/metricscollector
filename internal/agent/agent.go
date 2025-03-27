package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

type gauge float64
type counter int64

type Metrics struct {
	Alloc         gauge
	BuckHashSys   gauge
	Frees         gauge
	GCCPUFraction gauge
	GCSys         gauge
	HeapAlloc     gauge
	HeapIdle      gauge
	HeapInuse     gauge
	HeapObjects   gauge
	HeapReleased  gauge
	HeapSys       gauge
	LastGC        gauge
	Lookups       gauge
	MCacheInuse   gauge
	MCacheSys     gauge
	MSpanInuse    gauge
	MSpanSys      gauge
	Mallocs       gauge
	NextGC        gauge
	NumForcedGC   gauge
	NumGC         gauge
	OtherSys      gauge
	PauseTotalNs  gauge
	StackInuse    gauge
	StackSys      gauge
	Sys           gauge
	TotalAlloc    gauge
	PollCount     counter
	RandomValue   gauge
}

var actualMetrics Metrics

// checkServerAvailability is used for checking server availability
func checkServerAvailability(host string) bool {
	resp, err := http.Get(host + "/health")
	if err != nil {
		log.Printf("Something went wrong: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// CollectMetrics is used for collecting metrics from runtime package
func CollectMetrics(pollinterval int) {
	log.Printf("Poll interval: %d sec", pollinterval)

	var runtimeMetrics runtime.MemStats
	for {
		runtime.ReadMemStats(&runtimeMetrics)

		actualMetrics.Alloc = gauge(runtimeMetrics.Alloc)
		actualMetrics.BuckHashSys = gauge(runtimeMetrics.BuckHashSys)
		actualMetrics.Frees = gauge(runtimeMetrics.Frees)
		actualMetrics.GCCPUFraction = gauge(runtimeMetrics.GCCPUFraction)
		actualMetrics.GCSys = gauge(runtimeMetrics.GCSys)
		actualMetrics.HeapAlloc = gauge(runtimeMetrics.HeapAlloc)
		actualMetrics.HeapIdle = gauge(runtimeMetrics.HeapIdle)
		actualMetrics.HeapInuse = gauge(runtimeMetrics.HeapInuse)
		actualMetrics.HeapObjects = gauge(runtimeMetrics.HeapObjects)
		actualMetrics.HeapReleased = gauge(runtimeMetrics.HeapReleased)
		actualMetrics.HeapSys = gauge(runtimeMetrics.HeapSys)
		actualMetrics.LastGC = gauge(runtimeMetrics.LastGC)
		actualMetrics.Lookups = gauge(runtimeMetrics.Lookups)
		actualMetrics.MCacheInuse = gauge(runtimeMetrics.MCacheInuse)
		actualMetrics.MCacheSys = gauge(runtimeMetrics.MCacheSys)
		actualMetrics.MSpanInuse = gauge(runtimeMetrics.MSpanInuse)
		actualMetrics.MSpanSys = gauge(runtimeMetrics.MSpanSys)
		actualMetrics.Mallocs = gauge(runtimeMetrics.Mallocs)
		actualMetrics.NextGC = gauge(runtimeMetrics.NextGC)
		actualMetrics.NumForcedGC = gauge(runtimeMetrics.NumForcedGC)
		actualMetrics.NumGC = gauge(runtimeMetrics.NumGC)
		actualMetrics.OtherSys = gauge(runtimeMetrics.OtherSys)
		actualMetrics.PauseTotalNs = gauge(runtimeMetrics.PauseTotalNs)
		actualMetrics.StackInuse = gauge(runtimeMetrics.StackInuse)
		actualMetrics.StackSys = gauge(runtimeMetrics.StackSys)
		actualMetrics.Sys = gauge(runtimeMetrics.Sys)
		actualMetrics.TotalAlloc = gauge(runtimeMetrics.TotalAlloc)
		actualMetrics.PollCount++
		actualMetrics.RandomValue = gauge(rand.Float64())

		log.Printf("Current Poll count: %d", actualMetrics.PollCount)
		time.Sleep(time.Duration(pollinterval) * time.Second)
	}
}

// PostMetric is used for sending metrics to server
func PostMetric(client *http.Client, reportInterval int, host string) {
	log.Printf("Report Interval: %d sec", reportInterval)
	log.Printf("Host: %s", host)

	for !checkServerAvailability(host) {
		log.Printf("Server unreachable, retry in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	log.Printf("Server %s is reachable", host)

	reportCount := 0

	for {
		v := reflect.ValueOf(actualMetrics)
		t := reflect.TypeOf(actualMetrics)

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i).Interface()

			var metricType string
			switch reflect.TypeOf(value).Kind() {
			case reflect.Float64:
				metricType = "gauge"
			case reflect.Int64:
				metricType = "counter"
			default:
				continue
			}

			url := fmt.Sprintf("%s/update/%s/%s/%v", host, metricType, field.Name, value)

			req, err := http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				log.Printf("Error creating request for %s: %v", field.Name, err)
				continue
			}
			req.Header.Set("Content-Type", "text/plain")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Error sending request for %s: %v", field.Name, err)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				reportCount++
			}
		}
		log.Printf("Current report count: %d", reportCount)
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}
