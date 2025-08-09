package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antonminaichev/metricscollector/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipResponseWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	defer gzw.Close()

	recorder := httptest.NewRecorder()
	grw := gzipResponseWriter{
		ResponseWriter: recorder,
		Writer:         gzw,
	}

	testData := []byte("Hello, World!")
	n, err := grw.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	gzw.Close()
	// Проверяем, что данные сжаты
	assert.NotEqual(t, testData, buf.Bytes())
}

func TestGzipHandler(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	t.Run("handles gzipped request", func(t *testing.T) {
		// Создаем gzip-сжатые данные
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		gzw.Write([]byte("gzipped request body"))
		gzw.Close()

		req := httptest.NewRequest(http.MethodPost, "/test", &buf)
		req.Header.Set("Content-Encoding", "gzip")

		recorder := httptest.NewRecorder()
		handler := GzipHandler(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test response", recorder.Body.String())
	})

	t.Run("compresses response when client accepts gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		recorder := httptest.NewRecorder()
		handler := GzipHandler(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
		// Ответ должен быть сжат
		assert.NotEqual(t, "test response", recorder.Body.String())
	})

	t.Run("does not compress when client does not accept gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		recorder := httptest.NewRecorder()
		handler := GzipHandler(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, recorder.Header().Get("Content-Encoding"))
		assert.Equal(t, "test response", recorder.Body.String())
	})

	t.Run("handles invalid gzip request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("not gzipped"))
		req.Header.Set("Content-Encoding", "gzip")

		recorder := httptest.NewRecorder()
		handler := GzipHandler(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("handles multiple accept-encoding values", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "deflate, gzip, br")

		recorder := httptest.NewRecorder()
		handler := GzipHandler(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))
	})
}

func TestHashHandler(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	t.Run("skips validation when key is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
		req.Header.Set("HashSHA256", "invalid_hash")

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, "")
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test response", recorder.Body.String())
	})

	t.Run("validates hash correctly", func(t *testing.T) {
		key := "secret_key"
		body := "test body"

		// Создаем правильный хеш
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(body))
		expectedHash := hex.EncodeToString(mac.Sum(nil))

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req.Header.Set("HashSHA256", expectedHash)

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test response", recorder.Body.String())
		assert.NotEmpty(t, recorder.Header().Get("HashSHA256"))
	})

	t.Run("rejects invalid hash", func(t *testing.T) {
		key := "secret_key"
		body := "test body"

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req.Header.Set("HashSHA256", "invalid_hash")

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("rejects malformed hash", func(t *testing.T) {
		key := "secret_key"
		body := "test body"

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req.Header.Set("HashSHA256", "not_hex")

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("signs response", func(t *testing.T) {
		key := "secret_key"

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		// Проверяем, что ответ подписан
		responseHash := recorder.Header().Get("HashSHA256")
		assert.NotEmpty(t, responseHash)

		// Проверяем правильность подписи
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte("test response"))
		expectedHash := hex.EncodeToString(mac.Sum(nil))
		assert.Equal(t, expectedHash, responseHash)
	})

	t.Run("handles request without hash header", func(t *testing.T) {
		key := "secret_key"

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, recorder.Header().Get("HashSHA256"))
	})

	t.Run("preserves content-type header", func(t *testing.T) {
		key := "secret_key"
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		recorder := httptest.NewRecorder()
		handler := HashHandler(testHandler, key)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.NotEmpty(t, recorder.Header().Get("HashSHA256"))
	})
}

func TestRSADecryptMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	t.Run("passes through when private key is nil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("plain text"))

		recorder := httptest.NewRecorder()
		middleware := RSADecryptMiddleware(nil)
		handler := middleware(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "plain text", recorder.Body.String())
	})

	t.Run("passes through non-POST requests", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader("some data"))

		recorder := httptest.NewRecorder()
		middleware := RSADecryptMiddleware(privateKey)
		handler := middleware(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "some data", recorder.Body.String())
	})

	t.Run("decrypts POST request successfully", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		plaintext := "secret message"
		ciphertext, err := crypto.EncryptRSA(&privateKey.PublicKey, []byte(plaintext))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(ciphertext))

		recorder := httptest.NewRecorder()
		middleware := RSADecryptMiddleware(privateKey)
		handler := middleware(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, plaintext, recorder.Body.String())
	})

	t.Run("handles decryption failure", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("invalid ciphertext"))

		recorder := httptest.NewRecorder()
		middleware := RSADecryptMiddleware(privateKey)
		handler := middleware(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("handles read error", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Создаем reader, который возвращает ошибку
		errorReader := &errorReader{}
		req := httptest.NewRequest(http.MethodPost, "/test", errorReader)

		recorder := httptest.NewRecorder()
		middleware := RSADecryptMiddleware(privateKey)
		handler := middleware(testHandler)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestHashResponseWriter(t *testing.T) {
	t.Run("header operations", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		hrw := &hashResponseWriter{
			ResponseWriter: httptest.NewRecorder(),
			header:         make(http.Header),
			buffer:         buffer,
			statusCode:     http.StatusOK,
		}

		// Тестируем Header()
		header := hrw.Header()
		assert.NotNil(t, header)

		// Устанавливаем заголовок
		header.Set("Content-Type", "application/json")
		assert.Equal(t, "application/json", header.Get("Content-Type"))
	})

	t.Run("write header operation", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		hrw := &hashResponseWriter{
			ResponseWriter: httptest.NewRecorder(),
			header:         make(http.Header),
			buffer:         buffer,
			statusCode:     http.StatusOK,
		}

		hrw.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, hrw.statusCode)
	})

	t.Run("write operation", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		hrw := &hashResponseWriter{
			ResponseWriter: httptest.NewRecorder(),
			header:         make(http.Header),
			buffer:         buffer,
			statusCode:     http.StatusOK,
		}

		testData := []byte("test data")
		n, err := hrw.Write(testData)

		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, testData, buffer.Bytes())
	})

	t.Run("multiple write operations", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		hrw := &hashResponseWriter{
			ResponseWriter: httptest.NewRecorder(),
			header:         make(http.Header),
			buffer:         buffer,
			statusCode:     http.StatusOK,
		}

		data1 := []byte("hello ")
		data2 := []byte("world")

		n1, err1 := hrw.Write(data1)
		n2, err2 := hrw.Write(data2)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, len(data1), n1)
		assert.Equal(t, len(data2), n2)
		assert.Equal(t, "hello world", buffer.String())
	})
}

// Helper type for testing read errors
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func BenchmarkGzipHandler(b *testing.B) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark response"))
	})

	handler := GzipHandler(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}

func BenchmarkHashHandler(b *testing.B) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark response"))
	})

	handler := HashHandler(testHandler, "secret_key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}
}
