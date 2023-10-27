package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestVars(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key_1", "value")
	ctx.URLParams.Add("key_2", "value")

	middlewares := chi.Chain(
		middleware.WithValue(chi.RouteCtxKey, &ctx),
		withRequestVars,
	)
	handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, map[string]string{
			"key_1": "value",
			"key_2": "value",
		}, RequestVars(req))
	})

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}

func TestRequestVar(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key", "value")

	middlewares := chi.Chain(
		middleware.WithValue(chi.RouteCtxKey, &ctx),
		withRequestVars,
	)
	handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "value", RequestVar(req, "key"))
		assert.Empty(t, RequestVar(req, "nonexistent_key"))
	})

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}
