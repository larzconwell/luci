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

	t.Run("uses request id from X-Request-Id header", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			assert.Equal(t, "luci", RequestID(req))
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("X-Request-Id", "luci")

		handler.ServeHTTP(recorder, request)
	})

	t.Run("generates a valid ULID request id", func(t *testing.T) {
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

	t.Run("adds X-Request-Id to response headers", func(t *testing.T) {
		t.Parallel()

		handler := withRequestID(nil)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		request.Header.Set("X-Request-Id", "luci")

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, "luci", recorder.Header().Get("X-Request-Id"))
	})
}
