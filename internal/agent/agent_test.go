package agent

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCollectMetrics ensures CollectMetrics sends Alloc, PollCount, and RandomValue without race.
func TestCollectMetrics(t *testing.T) {
	pollInterval := 1
	jobs := make(chan Metrics, len(metrics)+5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start collection
	go CollectMetrics(ctx, pollInterval, jobs)

	// wait for required metrics
	var (
		gotAlloc     bool
		gotPollCount bool
		gotRandom    bool
	)
	// timeout guard
	timeout := time.After(5 * time.Second)

	for {
		select {
		case m := <-jobs:
			switch m.ID {
			case "Alloc":
				gotAlloc = true
				require.Equal(t, "gauge", m.MType)
				require.NotNil(t, m.Value)
				require.Greater(t, *m.Value, float64(0))

			case "PollCount":
				gotPollCount = true
				require.Equal(t, "counter", m.MType)
				require.NotNil(t, m.Delta)
				require.Greater(t, *m.Delta, int64(0))

			case "RandomValue":
				gotRandom = true
				require.Equal(t, "gauge", m.MType)
				require.NotNil(t, m.Value)
				require.GreaterOrEqual(t, *m.Value, float64(0.0))
				require.LessOrEqual(t, *m.Value, float64(1.0))
			}
			// break when all seen
			if gotAlloc && gotPollCount && gotRandom {
				return
			}

		case <-timeout:
			t.Fatal("timeout waiting for metrics from CollectMetrics")
		}
	}
}

func BenchmarkCollectMetricsLoop(b *testing.B) {
	jobs := make(chan Metrics, 1000)
	defer close(jobs)

	go func() {
		for range jobs {
			// эмуляция отправки
		}
	}()
	//Функция CollectMetrics, но без ticker-а
	var rt runtime.MemStats
	for i := 0; i < b.N; i++ {
		runtime.ReadMemStats(&rt)
		for _, mDef := range metrics {
			if mDef.MType != "gauge" && mDef.ID != "RandomValue" {
				continue
			}
			if mDef.getValue != nil {
				val := mDef.getValue(&rt)
				jobs <- Metrics{ID: mDef.ID, MType: mDef.MType, Value: &val}
			} else if mDef.ID == "RandomValue" {
				val := float64(i)
				jobs <- Metrics{ID: mDef.ID, MType: mDef.MType, Value: &val}
			}
		}
		delta := int64(i + 1)
		jobs <- Metrics{ID: "PollCount", MType: "counter", Delta: &delta}
	}
}
