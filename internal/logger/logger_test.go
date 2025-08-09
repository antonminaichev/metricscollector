package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitialize(t *testing.T) {
	// Сохраняем исходный логгер
	originalLog := Log

	t.Run("valid log level", func(t *testing.T) {
		err := Initialize("info")
		assert.NoError(t, err)
		assert.NotEqual(t, zap.NewNop(), Log)
	})

	t.Run("invalid log level", func(t *testing.T) {
		err := Initialize("invalid_level")
		assert.Error(t, err)
	})

	t.Run("debug level", func(t *testing.T) {
		err := Initialize("debug")
		assert.NoError(t, err)
		assert.NotNil(t, Log)
	})

	t.Run("warn level", func(t *testing.T) {
		err := Initialize("warn")
		assert.NoError(t, err)
		assert.NotNil(t, Log)
	})

	t.Run("error level", func(t *testing.T) {
		err := Initialize("error")
		assert.NoError(t, err)
		assert.NotNil(t, Log)
	})

	// Восстанавливаем исходный логгер
	Log = originalLog
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	responseData := &responseData{status: 0, size: 0}
	recorder := httptest.NewRecorder()

	lw := &loggingResponseWriter{
		ResponseWriter: recorder,
		responseData:   responseData,
	}

	testData := []byte("Hello, World!")
	n, err := lw.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, len(testData), responseData.size)
	assert.Equal(t, string(testData), recorder.Body.String())
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	responseData := &responseData{status: 0, size: 0}
	recorder := httptest.NewRecorder()

	lw := &loggingResponseWriter{
		ResponseWriter: recorder,
		responseData:   responseData,
	}

	statusCode := http.StatusNotFound
	lw.WriteHeader(statusCode)

	assert.Equal(t, statusCode, responseData.status)
	assert.Equal(t, statusCode, recorder.Code)
}

func TestLoggingResponseWriter_MultipleWrites(t *testing.T) {
	responseData := &responseData{status: 0, size: 0}
	recorder := httptest.NewRecorder()

	lw := &loggingResponseWriter{
		ResponseWriter: recorder,
		responseData:   responseData,
	}

	// Несколько записей
	data1 := []byte("Hello, ")
	data2 := []byte("World!")

	n1, err1 := lw.Write(data1)
	n2, err2 := lw.Write(data2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, len(data1), n1)
	assert.Equal(t, len(data2), n2)
	assert.Equal(t, len(data1)+len(data2), responseData.size)
	assert.Equal(t, "Hello, World!", recorder.Body.String())
}

func TestWithLogging(t *testing.T) {
	// Инициализируем логгер для тестов
	err := Initialize("debug")
	require.NoError(t, err)
	defer func() {
		Log = zap.NewNop() // Восстанавливаем nop logger
	}()

	t.Run("logs GET request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		})

		loggingHandler := WithLogging(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		loggingHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test response", recorder.Body.String())
	})

	t.Run("logs POST request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("created"))
		})

		loggingHandler := WithLogging(handler)

		req := httptest.NewRequest(http.MethodPost, "/api/create", nil)
		recorder := httptest.NewRecorder()

		loggingHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "created", recorder.Body.String())
	})

	t.Run("logs error response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		})

		loggingHandler := WithLogging(handler)

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		recorder := httptest.NewRecorder()

		loggingHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "error", recorder.Body.String())
	})

	t.Run("logs empty response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		loggingHandler := WithLogging(handler)

		req := httptest.NewRequest(http.MethodDelete, "/delete", nil)
		recorder := httptest.NewRecorder()

		loggingHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("preserves headers", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		})

		loggingHandler := WithLogging(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		recorder := httptest.NewRecorder()

		loggingHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Equal(t, `{"status": "ok"}`, recorder.Body.String())
	})
}

func TestResponseData(t *testing.T) {
	t.Run("initial values", func(t *testing.T) {
		rd := &responseData{}
		assert.Equal(t, 0, rd.status)
		assert.Equal(t, 0, rd.size)
	})

	t.Run("can be modified", func(t *testing.T) {
		rd := &responseData{}
		rd.status = 404
		rd.size = 100

		assert.Equal(t, 404, rd.status)
		assert.Equal(t, 100, rd.size)
	})
}

func BenchmarkWithLogging(b *testing.B) {
	err := Initialize("error") // Минимальный уровень для производительности
	require.NoError(b, err)
	defer func() {
		Log = zap.NewNop()
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark response"))
	})

	loggingHandler := WithLogging(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
		recorder := httptest.NewRecorder()
		loggingHandler.ServeHTTP(recorder, req)
	}
}
