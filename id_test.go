package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	t.Parallel()

	id := "luci"

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), requestIDKey{}, id))

	assert.Equal(t, id, RequestID(request))
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()

	t.Run("sets request id to Request-Id in request header", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "luci", RequestID(req))
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("Request-Id", "luci")

		handler.ServeHTTP(recorder, request)
	})

	t.Run("sets request id to X-Request-Id in request header", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "luci", RequestID(req))
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("X-Request-Id", "luci")

		handler.ServeHTTP(recorder, request)
	})

	t.Run("sets request id to Request-Id over X-Request-Id if both exist in request header", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "luci", RequestID(req))
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("Request-Id", "luci")
		request.Header.Set("X-Request-Id", "foobar")

		handler.ServeHTTP(recorder, request)
	})

	t.Run("sets request id to valid ULID if no request id is provided", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			id := RequestID(req)
			assert.NotEmpty(t, id)

			_, err := ulid.ParseStrict(id)
			assert.NoError(t, err)
		}))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	})

	t.Run("adds Request-Id and X-Request-Id to response header", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("Request-Id", "luci")

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, "luci", recorder.Header().Get("Request-Id"))
		assert.Equal(t, "luci", recorder.Header().Get("X-Request-Id"))
	})
}
