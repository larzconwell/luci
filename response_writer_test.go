package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriterHeader(t *testing.T) {
	t.Parallel()

	rw := httptest.NewRecorder()
	rw.Header().Set("Content-Type", "application/json")

	wrw := &responseWriter{rw: rw}
	wrw.Header().Set("Content-Length", "50")

	assert.Equal(t, "application/json", wrw.Header().Get("Content-Type"))
	assert.Equal(t, "50", rw.Header().Get("Content-Length"))
}

func TestResponseWriterWriteHeader(t *testing.T) {
	t.Parallel()

	t.Run("calls WriteHeader on wrapped ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}
		wrw.WriteHeader(http.StatusBadRequest)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, http.StatusBadRequest, wrw.status)
		assert.True(t, wrw.wroteHeader)
	})

	t.Run("if WriteHeader is called multiple times only the first affects the state", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}

		wrw.WriteHeader(http.StatusBadRequest)
		wrw.WriteHeader(http.StatusOK)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, http.StatusBadRequest, wrw.status)
		assert.True(t, wrw.wroteHeader)
	})
}

func TestResponseWriterWrite(t *testing.T) {
	t.Parallel()

	t.Run("calls WriteHeader with 200 if not already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}

		_, err := wrw.Write([]byte("abc"))
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("doesn't call WriteHeader if already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}
		wrw.WriteHeader(http.StatusBadRequest)

		_, err := wrw.Write([]byte("abc"))
		assert.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("writes the given bytes to the wrapped ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}

		n, err := wrw.Write([]byte("abc"))
		assert.NoError(t, err)
		assert.Equal(t, 3, n)

		n, err = wrw.Write([]byte("1234"))
		assert.NoError(t, err)
		assert.Equal(t, 4, n)

		assert.Equal(t, 7, wrw.length)
		assert.Equal(t, "abc1234", rw.Body.String())
	})
}

func TestResponseWithResponseWriter(t *testing.T) {
	t.Parallel()

	handler := withResponseWriter(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.IsType(t, new(responseWriter), rw)
	}))

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))

	rw := httptest.NewRecorder()
	wrw := &responseWriter{rw: rw}

	_, err := wrw.Write([]byte("abc"))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rw.Code)
}
