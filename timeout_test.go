package luci

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	t.Run("panics if not wrapped with a responseWriter", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		handler := withTimeout(errorHandler, time.Millisecond*200)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		assert.PanicsWithError(t, "luci: withTimeout has not been called with responseWriter", func() {
			handler.ServeHTTP(recorder, request)
		})

		assert.False(t, called)
	})

	t.Run("panics if next handler panics", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, time.Millisecond*200),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			panic(io.ErrUnexpectedEOF)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		assert.PanicsWithValue(t, io.ErrUnexpectedEOF, func() {
			handler.ServeHTTP(recorder, request)
		})

		assert.False(t, called)
	})

	t.Run("does nothing if next handler panics with http.ErrAbortHandler", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withRecover(errorHandler),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			panic(http.ErrAbortHandler)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		assert.NotPanics(t, func() {
			handler.ServeHTTP(recorder, request)
		})

		assert.False(t, called)
	})

	t.Run("does not call error handler if no timeout occurs", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, time.Millisecond*200),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.False(t, called)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("does not call error handler when timeout occurs but response has already started", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		timeout := time.Millisecond * 100
		middlewares := Middlewares{
			withResponseWriter,
			withLogger(noopLogger),
			withTimeout(errorHandler, timeout),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			<-time.After(timeout * 2)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.False(t, called)
	})

	t.Run("calls error handler when timeout occurs", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true

			assert.Equal(t, http.StatusServiceUnavailable, status)
			assert.Equal(t, http.ErrHandlerTimeout, err)
		}

		timeout := time.Millisecond * 100
		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, timeout),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			<-time.After(timeout * 2)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.True(t, called)
	})

	t.Run("request context is closed when timeout occurs", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		timeout := time.Millisecond * 100
		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, timeout),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			select {
			case <-req.Context().Done():
			case <-time.After(timeout * 2):
				assert.Fail(t, "context channel was not closed on timeout")
			}
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.True(t, called)
	})

	t.Run("further writes return http.ErrHandlerTimeout when timeout occurs", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, time.Millisecond*100),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			<-req.Context().Done()
			<-time.After(time.Millisecond * 200)

			_, err := rw.Write([]byte("data"))
			assert.Equal(t, http.ErrHandlerTimeout, err)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.True(t, called)
	})

	t.Run("further writes return context error when request context is canceled", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withTimeout(errorHandler, time.Millisecond*100),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			<-req.Context().Done()
			<-time.After(time.Millisecond * 200)

			_, err := rw.Write([]byte("data"))
			assert.Equal(t, context.Canceled, err)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		ctx, cancel := context.WithCancel(request.Context())
		cancel()

		handler.ServeHTTP(recorder, request.WithContext(ctx))

		assert.True(t, called)
	})
}
