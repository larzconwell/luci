package luci

import (
	"fmt"
	"io"
	"net/http"
)

type responseWriter struct {
	rw          http.ResponseWriter
	wroteHeader bool
	status      int
	length      int64
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
	rw.finishHeader()

	n, err := rw.rw.Write(b)
	rw.length += int64(n)
	if err != nil {
		return n, fmt.Errorf("luci: write: %w", err)
	}

	return n, nil
}

func (rw *responseWriter) ReadFrom(r io.Reader) (int64, error) {
	rw.finishHeader()

	// Let io.Copy determine if the ResponseWriter is also a io.ReaderFrom.
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

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := rw.rw.(http.Pusher)
	if ok {
		err := pusher.Push(target, opts)
		if err != nil {
			return fmt.Errorf("luci: push: %w", err)
		}

		return nil
	}

	return http.ErrNotSupported
}

func (rw *responseWriter) finishHeader() {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
}

func withResponseWriter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		wrw := &responseWriter{rw: rw}

		next.ServeHTTP(wrw, req)
	})
}
