package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	ms "github.com/antonminaichev/metricscollector/internal/server/memstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type want struct {
	responseCode int
	responseBody string
	contentType  string
	storage      *ms.MemStorage
}

func TestHealthCheck(t *testing.T) {
	type want struct {
		responseCode int
		responseBody string
		contentType  string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "Health check",
			want: want{
				responseCode: http.StatusOK,
				responseBody: `{"status": "ok"}`,
				contentType:  "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			HealthCheck(w, request)

			res := w.Result()
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want.responseBody, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestPostMetric(t *testing.T) {
	type want struct {
		responseCode int
		contentType  string
		storage      *ms.MemStorage
	}
	tests := []struct {
		name string
		url  string
		want want
	}{
		{
			name: "Post metric counter positive",
			url:  "/update/counter/test/1",
			want: want{
				responseCode: http.StatusOK,
				contentType:  "text/plain",
				storage: &ms.MemStorage{
					Counter: map[string]int64{
						"test": 1,
					},
				},
			},
		},
		{
			name: "Post metric gauge positive",
			url:  "/update/gauge/test/1.543",
			want: want{
				responseCode: http.StatusOK,
				contentType:  "text/plain",
				storage: &ms.MemStorage{
					Gauge: map[string]float64{
						"test": 1.543,
					},
				},
			},
		},
		{
			name: "Incorrect counter value",
			url:  "/update/counter/test/abc",
			want: want{
				responseCode: http.StatusBadRequest,
				contentType:  "text/plain",
				storage:      &ms.MemStorage{},
			},
		},
		{
			name: "Incorrect gauge value",
			url:  "/update/gauge/test/abc",
			want: want{
				responseCode: http.StatusBadRequest,
				contentType:  "text/plain",
				storage:      &ms.MemStorage{},
			},
		},
		{
			name: "Post metric bad url",
			url:  "/test",
			want: want{
				responseCode: http.StatusNotFound,
				contentType:  "text/plain",
				storage:      &ms.MemStorage{},
			},
		},
		{
			name: "Post metric incorrect metric type",
			url:  "/update/test/test/22",
			want: want{
				responseCode: http.StatusBadRequest,
				contentType:  "text/plain",
				storage:      &ms.MemStorage{},
			},
		},
		{
			name: "Post metric without metric name",
			url:  "/update/gauge//1.543",
			want: want{
				responseCode: http.StatusNotFound,
				contentType:  "text/plain",
				storage:      &ms.MemStorage{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.url, nil)
			w := httptest.NewRecorder()
			PostMetric(w, request, tt.want.storage)

			res := w.Result()
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
