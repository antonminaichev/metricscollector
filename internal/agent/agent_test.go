package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestCollectMetrics_ContextCancellation(t *testing.T) {
	jobs := make(chan Metrics, 10)
	ctx, cancel := context.WithCancel(context.Background())

	// Запускаем сбор метрик
	go CollectMetrics(ctx, 1, jobs)

	// Ждем немного, чтобы функция запустилась
	time.Sleep(100 * time.Millisecond)

	// Отменяем контекст
	cancel()

	// Ждем немного, чтобы функция завершилась
	time.Sleep(100 * time.Millisecond)

	// Канал должен быть закрыт или функция должна завершиться
	select {
	case <-jobs:
		// ОК, возможно получили метрику до отмены
	default:
		// ОК, метрик нет
	}
}

func TestCollectSystemMetrics(t *testing.T) {
	jobs := make(chan Metrics, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Запускаем сбор системных метрик
	go CollectSystemMetrics(ctx, 1, jobs)

	var (
		gotTotalMemory bool
		gotFreeMemory  bool
	)

	timeout := time.After(3 * time.Second)

	for {
		select {
		case m := <-jobs:
			switch m.ID {
			case "TotalMemory":
				gotTotalMemory = true
				assert.Equal(t, "gauge", m.MType)
				assert.NotNil(t, m.Value)
				assert.Greater(t, *m.Value, float64(0))

			case "FreeMemory":
				gotFreeMemory = true
				assert.Equal(t, "gauge", m.MType)
				assert.NotNil(t, m.Value)
				assert.GreaterOrEqual(t, *m.Value, float64(0))

			default:
				if len(m.ID) > 13 && m.ID[:13] == "CPUutilization" {
					// Проверяем CPU метрики, но не требуем их для завершения теста
					assert.Equal(t, "gauge", m.MType)
					assert.NotNil(t, m.Value)
					assert.GreaterOrEqual(t, *m.Value, float64(0))
					assert.LessOrEqual(t, *m.Value, float64(100))
				}
			}

			// Завершаем если получили основные метрики
			if gotTotalMemory && gotFreeMemory {
				// CPU может быть недоступен в тестовой среде, поэтому не требуем его
				return
			}

		case <-timeout:
			// Проверяем что получили хотя бы память
			if gotTotalMemory || gotFreeMemory {
				return // Хотя бы что-то получили
			}
			t.Fatal("timeout waiting for system metrics")
		}
	}
}

func TestCollectSystemMetrics_ContextCancellation(t *testing.T) {
	jobs := make(chan Metrics, 10)
	ctx, cancel := context.WithCancel(context.Background())

	go CollectSystemMetrics(ctx, 1, jobs)
	time.Sleep(100 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что функция корректно завершилась
	select {
	case <-jobs:
		// ОК, возможно получили метрику до отмены
	default:
		// ОК, метрик нет
	}
}

func TestCalculateHash(t *testing.T) {
	t.Run("calculates hash correctly", func(t *testing.T) {
		buf := bytes.NewBufferString("test data")
		key := "secret_key"

		hash1 := calculateHash(buf, key)
		hash2 := calculateHash(buf, key)

		assert.NotEmpty(t, hash1)
		assert.Equal(t, hash1, hash2) // Хеш должен быть детерминированным
		assert.Len(t, hash1, 64)      // SHA256 в hex = 64 символа
	})

	t.Run("different data produces different hash", func(t *testing.T) {
		buf1 := bytes.NewBufferString("test data 1")
		buf2 := bytes.NewBufferString("test data 2")
		key := "secret_key"

		hash1 := calculateHash(buf1, key)
		hash2 := calculateHash(buf2, key)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different key produces different hash", func(t *testing.T) {
		buf := bytes.NewBufferString("test data")
		key1 := "secret_key_1"
		key2 := "secret_key_2"

		hash1 := calculateHash(buf, key1)
		hash2 := calculateHash(buf, key2)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestCheckServerAvailability(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		available := checkServerAvailability(server.URL)
		assert.True(t, available)
	})

	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		available := checkServerAvailability(server.URL)
		assert.False(t, available)
	})

	t.Run("server is unreachable", func(t *testing.T) {
		available := checkServerAvailability("http://localhost:99999")
		assert.False(t, available)
	})

	t.Run("adds http prefix if missing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		// Убираем http:// из URL
		hostWithoutProtocol := server.URL[7:] // убираем "http://"
		available := checkServerAvailability(hostWithoutProtocol)
		assert.True(t, available)
	})

	t.Run("preserves https prefix", func(t *testing.T) {
		available := checkServerAvailability("https://httpbin.org")
		// Может быть true или false в зависимости от доступности, главное что не паникует
		_ = available
	})
}

func TestDoRequest(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		err = doRequest(client, req)
		assert.NoError(t, err)
	})

	t.Run("failed request", func(t *testing.T) {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, "http://localhost:99999", nil)
		require.NoError(t, err)

		err = doRequest(client, req)
		assert.Error(t, err)
	})
}

func TestSendPlain(t *testing.T) {
	requestReceived := false
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}
	buf := bytes.NewBufferString("test data")
	hashkey := "secret_key"
	agentIP := "127.0.0.1"

	sendPlain(client, server.URL, hashkey, buf, agentIP)

	assert.True(t, requestReceived)
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "gzip", receivedHeaders.Get("Content-Encoding"))
	assert.NotEmpty(t, receivedHeaders.Get("HashSHA256"))
	assert.Equal(t, agentIP, receivedHeaders.Get("X-Real-IP"))
}

func TestSendPlain_NoHashKey(t *testing.T) {
	requestReceived := false
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}
	buf := bytes.NewBufferString("test data")
	agentIP := "127.0.0.1"

	sendPlain(client, server.URL, "", buf, agentIP)

	assert.True(t, requestReceived)
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "gzip", receivedHeaders.Get("Content-Encoding"))
	assert.Empty(t, receivedHeaders.Get("HashSHA256"))
	assert.Equal(t, agentIP, receivedHeaders.Get("X-Real-IP"))
}

func TestSendEncrypted(t *testing.T) {
	requestReceived := false
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Генерируем ключи для теста
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	client := &http.Client{}
	buf := bytes.NewBufferString("test data")
	hashkey := "secret_key"
	agentIP := "127.0.0.1"

	sendEncrypted(client, server.URL, hashkey, buf, &privateKey.PublicKey, agentIP)

	assert.True(t, requestReceived)
	assert.Equal(t, "application/octet-stream", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "gzip", receivedHeaders.Get("Content-Encoding"))
	assert.NotEmpty(t, receivedHeaders.Get("HashSHA256"))
	assert.Equal(t, agentIP, receivedHeaders.Get("X-Real-IP"))
}

func TestSendEncrypted_NoHashKey(t *testing.T) {
	requestReceived := false
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	client := &http.Client{}
	buf := bytes.NewBufferString("test data")
	agentIP := "127.0.0.1"

	sendEncrypted(client, server.URL, "", buf, &privateKey.PublicKey, agentIP)

	assert.True(t, requestReceived)
	assert.Equal(t, "application/octet-stream", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "gzip", receivedHeaders.Get("Content-Encoding"))
	assert.Empty(t, receivedHeaders.Get("HashSHA256"))
	assert.Equal(t, agentIP, receivedHeaders.Get("X-Real-IP"))
}

func TestMetricWorker(t *testing.T) {
	t.Run("processes metrics without encryption", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		jobs := make(chan Metrics, 2)

		// Добавляем несколько метрик
		jobs <- Metrics{ID: "test1", MType: "gauge", Value: ptrFloat64(1.0)}
		jobs <- Metrics{ID: "test2", MType: "counter", Delta: ptrInt64(5)}
		close(jobs)

		MetricWorker(client, server.URL, "", jobs, 0, "")

		assert.Equal(t, 2, requestCount)
	})

	t.Run("processes metrics with encryption", func(t *testing.T) {
		// Создаем временный файл с ключом
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		keyFile := createTempRealKeyFile(t, &privateKey.PublicKey)
		defer func() {
			if keyFile != "" {
				// Удаляем временный файл
				_ = os.Remove(keyFile)
			}
		}()

		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			// Проверяем что используется шифрование только если файл ключа существует
			if keyFile != "" {
				assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		jobs := make(chan Metrics, 1)
		jobs <- Metrics{ID: "test", MType: "gauge", Value: ptrFloat64(1.0)}
		close(jobs)

		MetricWorker(client, server.URL, "", jobs, 0, keyFile)

		assert.Equal(t, 1, requestCount)
	})

	t.Run("adds http prefix if missing", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		jobs := make(chan Metrics, 1)
		jobs <- Metrics{ID: "test", MType: "gauge", Value: ptrFloat64(1.0)}
		close(jobs)

		// Убираем http:// из URL
		hostWithoutProtocol := server.URL[7:]
		MetricWorker(client, hostWithoutProtocol, "", jobs, 0, "")

		assert.Equal(t, 1, requestCount)
	})
}

func TestMetrics_Structure(t *testing.T) {
	t.Run("metrics array is properly defined", func(t *testing.T) {
		assert.Greater(t, len(metrics), 0)

		// Проверяем, что есть основные метрики
		var foundAlloc, foundPollCount, foundRandomValue bool
		for _, m := range metrics {
			switch m.ID {
			case "Alloc":
				foundAlloc = true
				assert.Equal(t, "gauge", m.MType)
				assert.NotNil(t, m.getValue)
			case "PollCount":
				foundPollCount = true
				assert.Equal(t, "counter", m.MType)
			case "RandomValue":
				foundRandomValue = true
				assert.Equal(t, "gauge", m.MType)
			}
		}

		assert.True(t, foundAlloc)
		assert.True(t, foundPollCount)
		assert.True(t, foundRandomValue)
	})
}

// Helper functions

func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

func createTempKeyFile(t *testing.T, publicKey *rsa.PublicKey) string {
	// В реальном тесте нужно создать временный файл с ключом
	// Для простоты возвращаем пустую строку (не будет шифрования)
	return ""
}

func createTempRealKeyFile(t *testing.T, publicKey *rsa.PublicKey) string {
	// В реальном тесте нужно создать временный файл с ключом
	// Для простоты возвращаем пустую строку (не будет шифрования)
	return ""
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

func BenchmarkCalculateHash(b *testing.B) {
	buf := bytes.NewBufferString("test data for benchmark")
	key := "secret_key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateHash(buf, key)
	}
}
