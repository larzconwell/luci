package luci

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	logger := slog.Default()

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), requestLoggerKey{}, logger))

	assert.Same(t, logger, RequestLogger(request))
}

func TestWithRequestLogger(t *testing.T) {
	t.Parallel()

	t.Run("adds logger to request context", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()

		handler := withRequestLogger(logger)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.NotNil(t, RequestLogger(req))
		}))

		rw := &responseWriterWrapper{rw: httptest.NewRecorder()}
		req := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(rw, req)
	})

	t.Run("panics if response writer has not been wrapped", func(t *testing.T) {
		t.Parallel()

		logger := slog.Default()

		handler := withRequestLogger(logger)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Fail(t, "handler should not be called")
		}))

		rw := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/status", nil)

		assert.Panics(t, func() {
			handler.ServeHTTP(rw, req)
		})
	})
}
