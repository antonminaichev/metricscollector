// Middleware packacge is used for middleware functions.
package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
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

// GzipHandler is a middleware to gzip request payload.
func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(rw, "Failed to create gzip reader", http.StatusBadRequest)
				return
			}
			defer func() {
				if err := gzr.Close(); err != nil {
					http.Error(rw, "Failed to close gzip writer", http.StatusInternalServerError)
				}
			}()
			r.Body = io.NopCloser(gzr)
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			rw.Header().Set("Content-Encoding", "gzip")
			gzw := gzip.NewWriter(rw)
			defer func() {
				if err := gzw.Close(); err != nil {
					http.Error(rw, "Failed to close gzip writer", http.StatusInternalServerError)
				}
			}()

			gzrw := gzipResponseWriter{Writer: gzw, ResponseWriter: rw}
			next.ServeHTTP(gzrw, r)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}

// HashHandler checks HMAC-SHA256 of incoming requests and signs answers.
// If keys is empty, checking is skipped.
func HashHandler(next http.Handler, key string) http.Handler {
	if key == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recvSig := r.Header.Get("HashSHA256")
		if recvSig != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			mac := hmac.New(sha256.New, []byte(key))
			mac.Write(body)
			expected := mac.Sum(nil)
			recvBytes, err := hex.DecodeString(recvSig)
			log.Printf("Hash expected: %s", hex.EncodeToString(expected))
			log.Printf("Hash recieved: %s", recvSig)
			if err != nil || !hmac.Equal(recvBytes, expected) {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		buf := &bytes.Buffer{}
		hw := &hashResponseWriter{
			ResponseWriter: w,
			header:         make(http.Header),
			buffer:         buf,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(hw, r)

		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(buf.Bytes())
		if ct := hw.header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		for k, vals := range hw.header {
			if strings.EqualFold(k, "Content-Type") {
				continue
			}
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.Header().Set("HashSHA256", hex.EncodeToString(mac.Sum(nil)))
		w.WriteHeader(hw.statusCode)
		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Printf("failed to write response body: %v", err)
		}
	})
}

type hashResponseWriter struct {
	http.ResponseWriter
	header     http.Header
	buffer     *bytes.Buffer
	statusCode int
}

func (h *hashResponseWriter) Header() http.Header         { return h.header }
func (h *hashResponseWriter) WriteHeader(status int)      { h.statusCode = status }
func (h *hashResponseWriter) Write(b []byte) (int, error) { return h.buffer.Write(b) }
