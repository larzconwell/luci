package luci

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithRecover(t *testing.T) {
	t.Parallel()

	t.Run("does not call error handler if no panic occurs", func(t *testing.T) {
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
			rw.WriteHeader(http.StatusOK)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.False(t, called)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("does not call error handler when panic occurs with http.ErrAbortHandler", func(t *testing.T) {
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

		handler.ServeHTTP(recorder, request)

		assert.False(t, called)
	})

	t.Run("does not call error handler when panic occurs but response has already started", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			fmt.Println("calling error handler")
			called = true
		}

		middlewares := Middlewares{
			withResponseWriter,
			withLogger(slog.Default()),
			withRecover(errorHandler),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			panic(io.ErrUnexpectedEOF)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.False(t, called)
	})

	t.Run("calls error handler when panic occurs with error type", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true

			assert.Equal(t, http.StatusInternalServerError, status)
			assert.Equal(t, io.ErrUnexpectedEOF, err)
		}

		middlewares := Middlewares{
			withResponseWriter,
			withRecover(errorHandler),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			panic(io.ErrUnexpectedEOF)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.True(t, called)
	})

	t.Run("calls error handler when panic occurs with non error value", func(t *testing.T) {
		t.Parallel()

		var called bool
		errorHandler := func(rw http.ResponseWriter, req *http.Request, status int, err error) {
			called = true

			assert.Equal(t, http.StatusInternalServerError, status)
			assert.Equal(t, errors.New("5"), err)
		}

		middlewares := Middlewares{
			withResponseWriter,
			withRecover(errorHandler),
		}

		handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			panic(5)
		})

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		handler.ServeHTTP(recorder, request)

		assert.True(t, called)
	})
}
