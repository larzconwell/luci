package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), loggerKey{}, noopLogger))

	assert.Same(t, noopLogger, Logger(request))
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	t.Run("adds logger to request context", func(t *testing.T) {
		t.Parallel()

		handler := withLogger(noopLogger)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			assert.NotNil(t, Logger(req))
		}))

		rw := &responseWriter{rw: httptest.NewRecorder()}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/status", nil)

		handler.ServeHTTP(rw, req)
	})

	t.Run("panics if response writer has not been wrapped", func(t *testing.T) {
		t.Parallel()

		handler := withLogger(noopLogger)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			assert.Fail(t, "handler should not be called")
		}))

		rw := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/status", nil)

		assert.PanicsWithError(t, "luci: withLogger has not been called with responseWriter", func() {
			handler.ServeHTTP(rw, req)
		})
	})
}
