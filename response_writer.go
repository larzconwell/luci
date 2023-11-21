package luci

import (
	"fmt"
	"net/http"
)

type responseWriter struct {
	rw          http.ResponseWriter
	wroteHeader bool
	status      int
	length      int
}

func (rw *responseWriter) Header() http.Header {
	return rw.rw.Header()
}

func (rw *responseWriter) WriteHeader(status int) {
	if rw.wroteHeader {
		return
	}

	rw.wroteHeader = true
	rw.status = status

	rw.rw.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.rw.Write(b)
	rw.length += n
	if err != nil {
		return n, fmt.Errorf("luci: write: %w", err)
	}

	return n, nil
}

func withResponseWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wrw := &responseWriter{rw: rw}

		next.ServeHTTP(wrw, req)
	})
}
