package luci

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type responseWriter struct {
	rw          http.ResponseWriter
	wroteHeader bool
	status      int
	length      int64
	mu          sync.Mutex
}

func (rw *responseWriter) Header() http.Header {
	return rw.rw.Header()
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.wroteHeader {
		return
	}

	rw.lockedWriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.lockedFinishHeader()

	n, err := rw.rw.Write(b)
	rw.length += int64(n)
	if err != nil {
		return n, fmt.Errorf("luci: write: %w", err)
	}

	return n, nil
}

func (rw *responseWriter) ReadFrom(r io.Reader) (int64, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.lockedFinishHeader()

	// Let io.Copy determine if the underlying http.ResponseWriter is a io.ReaderFrom.
	n, err := io.Copy(rw.rw, r)
	rw.length += n
	if err != nil {
		return n, fmt.Errorf("luci: read from: %w", err)
	}

	return n, nil
}

func (rw *responseWriter) Flush() {
	flusher, ok := rw.rw.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) stats() (wroteHeader bool, status int, length int64) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	return rw.wroteHeader, rw.status, rw.length
}

func (rw *responseWriter) lockedFinishHeader() {
	if !rw.wroteHeader {
		rw.lockedWriteHeader(http.StatusOK)
	}
}

func (rw *responseWriter) lockedWriteHeader(status int) {
	rw.wroteHeader = true
	rw.status = status

	rw.rw.WriteHeader(status)
}

func withResponseWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wrw := &responseWriter{rw: rw}

		next.ServeHTTP(wrw, req)
	})
}
