package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/antonminaichev/metricscollector/internal/retry"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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

func calculateHash(buf *bytes.Buffer, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(buf.Bytes())
	return hex.EncodeToString(mac.Sum(nil))
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
func CollectMetrics(ctx context.Context, pollInterval int, jobs chan<- Metrics) {
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()
	var rt runtime.MemStats
	var pc int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.ReadMemStats(&rt)

			// send gauges
			for _, mDef := range metrics {
				if mDef.MType != "gauge" {
					continue
				}
				if mDef.getValue != nil {
					val := mDef.getValue(&rt)
					jobs <- Metrics{ID: mDef.ID, MType: mDef.MType, Value: &val}
				} else if mDef.ID == "RandomValue" {
					val := rand.Float64()
					jobs <- Metrics{ID: mDef.ID, MType: mDef.MType, Value: &val}
				}
			}

			// send counter
			pc++
			delta := pc
			jobs <- Metrics{ID: "PollCount", MType: "counter", Delta: &delta}
		}
	}
}

func CollectSystemMetrics(ctx context.Context, pollInterval int, jobs chan<- Metrics) {
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()
	cpuCount, _ := cpu.Counts(true)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Memory metrics
			if vm, err := mem.VirtualMemory(); err == nil {
				tot := float64(vm.Total)
				free := float64(vm.Free)
				jobs <- Metrics{ID: "TotalMemory", MType: "gauge", Value: &tot}
				jobs <- Metrics{ID: "FreeMemory", MType: "gauge", Value: &free}
			}

			// CPU utilization per core
			if pct, err := cpu.Percent(0, true); err == nil {
				for i := 0; i < cpuCount && i < len(pct); i++ {
					v := pct[i]
					jobs <- Metrics{ID: fmt.Sprintf("CPUutilization%d", i), MType: "gauge", Value: &v}
				}
			}
		}
	}
}

func MetricWorker(client *http.Client, host, hashkey string, jobs <-chan Metrics, reportInterval int) {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	for m := range jobs {
		buf := bytes.NewBuffer(nil)
		gw, _ := gzip.NewWriterLevel(buf, gzip.BestSpeed)
		data, _ := json.Marshal(m)
		gw.Write(data)
		gw.Close()

		url := fmt.Sprintf("%s/update", host)
		req, _ := http.NewRequest(http.MethodPost, url, buf)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		if hashkey != "" {
			req.Header.Set("HashSHA256", calculateHash(buf, hashkey))
		}

		retry.Do(retry.DefaultRetryConfig(), func() error {
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
			return err
		})
		time.Sleep(time.Duration(reportInterval) * time.Second)
	}
}
