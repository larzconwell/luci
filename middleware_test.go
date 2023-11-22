package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithValue(t *testing.T) {
	t.Parallel()

	expected := time.Now().UTC()
	middleware := WithValue("time", expected)

	handler := middleware(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Exactly(t, expected, req.Context().Value("time"))
	}))

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}

func TestMiddlewaresHandler(t *testing.T) {
	t.Parallel()

	middleware := func(key any) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				newReq := req.WithContext(context.WithValue(req.Context(), key, time.Now()))
				next.ServeHTTP(rw, newReq)
			})
		}
	}

	middlewares := Middlewares{
		middleware("first"),
		middleware("second"),
		middleware("third"),
	}

	middlewares.Handler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		first, _ := req.Context().Value("first").(time.Time)
		second, _ := req.Context().Value("second").(time.Time)
		third, _ := req.Context().Value("third").(time.Time)

		assert.True(t, first.Before(second))
		assert.True(t, first.Before(third))
		assert.True(t, second.Before(third))
	}))
}

func TestMiddlewaresHandlerFun(t *testing.T) {
	t.Parallel()

	middleware := func(key any) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				newReq := req.WithContext(context.WithValue(req.Context(), key, time.Now()))
				next.ServeHTTP(rw, newReq)
			})
		}
	}

	middlewares := Middlewares{
		middleware("first"),
		middleware("second"),
		middleware("third"),
	}

	middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		first, _ := req.Context().Value("first").(time.Time)
		second, _ := req.Context().Value("second").(time.Time)
		third, _ := req.Context().Value("third").(time.Time)

		assert.True(t, first.Before(second))
		assert.True(t, first.Before(third))
		assert.True(t, second.Before(third))
	})
}
