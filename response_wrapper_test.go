package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriterWrapperHeader(t *testing.T) {
	t.Parallel()

	rw := httptest.NewRecorder()
	rw.Header().Set("Content-Type", "application/json")

	rww := &responseWriterWrapper{rw: rw}
	rww.Header().Set("Content-Length", "50")

	assert.Equal(t, "application/json", rww.Header().Get("Content-Type"))
	assert.Equal(t, "50", rw.Header().Get("Content-Length"))
}

func TestResponseWriterWrapperWriteHeader(t *testing.T) {
	t.Parallel()

	t.Run("calls WriteHeader on wrapped ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		rww := &responseWriterWrapper{rw: rw}
		rww.WriteHeader(http.StatusBadRequest)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, http.StatusBadRequest, rww.status)
		assert.True(t, rww.wroteHeader)
	})

	t.Run("if WriteHeader is called multiple times only the first affects the state", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		rww := &responseWriterWrapper{rw: rw}

		rww.WriteHeader(http.StatusBadRequest)
		rww.WriteHeader(http.StatusOK)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.Equal(t, http.StatusBadRequest, rww.status)
		assert.True(t, rww.wroteHeader)
	})
}

func TestResponseWriterWrapperWrite(t *testing.T) {
	t.Parallel()

	t.Run("calls WriteHeader with 200 if not already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		rww := &responseWriterWrapper{rw: rw}

		_, err := rww.Write([]byte("abc"))
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("doesn't call WriteHeader if already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		rww := &responseWriterWrapper{rw: rw}
		rww.WriteHeader(http.StatusBadRequest)

		_, err := rww.Write([]byte("abc"))
		assert.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("writes the given bytes to the wrapped ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		rww := &responseWriterWrapper{rw: rw}

		n, err := rww.Write([]byte("abc"))
		assert.NoError(t, err)
		assert.Equal(t, 3, n)

		n, err = rww.Write([]byte("1234"))
		assert.NoError(t, err)
		assert.Equal(t, 4, n)

		assert.Equal(t, 7, rww.length)
		assert.Equal(t, "abc1234", rw.Body.String())
	})
}

func TestResponseWithResponseWriterWrapper(t *testing.T) {
	t.Parallel()

	handler := withResponseWriterWrapper(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.IsType(t, new(responseWriterWrapper), rw)
	}))

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))

	rw := httptest.NewRecorder()
	rww := &responseWriterWrapper{rw: rw}

	_, err := rww.Write([]byte("abc"))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rw.Code)
}
