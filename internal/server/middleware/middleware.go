package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(rw, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer gzr.Close()
			r.Body = io.NopCloser(gzr)
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			rw.Header().Set("Content-Encoding", "gzip")
			gzw := gzip.NewWriter(rw)
			defer gzw.Close()

			gzrw := gzipResponseWriter{Writer: gzw, ResponseWriter: rw}
			next.ServeHTTP(gzrw, r)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}
