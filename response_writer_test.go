package luci

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	t.Parallel()
	rw := new(responseWriter)

	assert.Implements(t, (*io.ReaderFrom)(nil), rw)
	assert.Implements(t, (*http.Flusher)(nil), rw)
}

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

		assert.Equal(t, int64(7), wrw.length)
		assert.Equal(t, "abc1234", rw.Body.String())
	})
}

func TestResponseWriterReadFrom(t *testing.T) {
	t.Parallel()

	t.Run("calls WriteHeader with 200 if not already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}

		reader := bytes.NewBufferString("abc")

		_, err := wrw.ReadFrom(reader)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("doesn't call WriteHeader if already called", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}
		wrw.WriteHeader(http.StatusBadRequest)

		reader := bytes.NewBufferString("abc")

		_, err := wrw.ReadFrom(reader)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	t.Run("reads from the given reader and writes to the wrapped ResponseWriter", func(t *testing.T) {
		t.Parallel()

		rw := httptest.NewRecorder()
		wrw := &responseWriter{rw: rw}

		reader1 := bytes.NewBufferString("abc")

		n, err := wrw.ReadFrom(reader1)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), n)

		reader2 := bytes.NewBufferString("1234")

		n, err = wrw.ReadFrom(reader2)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), n)

		assert.Equal(t, int64(7), wrw.length)
		assert.Equal(t, "abc1234", rw.Body.String())
	})
}

func TestResponseWriterFlush(t *testing.T) {
	t.Parallel()

	rw := httptest.NewRecorder()
	wrw := &responseWriter{rw: rw}

	wrw.Flush()

	assert.True(t, rw.Flushed)
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
