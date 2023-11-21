package luci

import (
	"fmt"
	"net/http"
)

type responseWriterWrapper struct {
	rw          http.ResponseWriter
	wroteHeader bool
	status      int
	length      int
}

func (rww *responseWriterWrapper) Header() http.Header {
	return rww.rw.Header()
}

func (rww *responseWriterWrapper) WriteHeader(status int) {
	if rww.wroteHeader {
		return
	}

	rww.wroteHeader = true
	rww.status = status

	rww.rw.WriteHeader(status)
}

func (rww *responseWriterWrapper) Write(b []byte) (int, error) {
	if !rww.wroteHeader {
		rww.WriteHeader(http.StatusOK)
	}

	n, err := rww.rw.Write(b)
	rww.length += n
	if err != nil {
		return n, fmt.Errorf("luci: write: %w", err)
	}

	return n, nil
}

func withResponseWriterWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rww := &responseWriterWrapper{rw: rw}

		next.ServeHTTP(rww, req)
	})
}
