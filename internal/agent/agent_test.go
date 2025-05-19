package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCollectMetrics ensures CollectMetrics sends Alloc, PollCount, and RandomValue without race.
func TestCollectMetrics(t *testing.T) {
	pollInterval := 1
	jobs := make(chan Metrics, len(metrics)+5)

	// start collection
	go CollectMetrics(pollInterval, jobs)

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
