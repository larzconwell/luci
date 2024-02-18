package luci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestVars(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key_1", "value")
	ctx.URLParams.Add("key_2", "value")

	middlewares := Middlewares{
		WithValue(chi.RouteCtxKey, &ctx),
		withVars,
	}

	handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, map[string]string{
			"key_1": "value",
			"key_2": "value",
		}, Vars(req))
	})

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}

func TestVar(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key", "value")

	middlewares := Middlewares{
		WithValue(chi.RouteCtxKey, &ctx),
		withVars,
	}

	handler := middlewares.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "value", Var(req, "key"))
		assert.Empty(t, Var(req, "nonexistent_key"))
	})

	handler.ServeHTTP(nil, httptest.NewRequest(http.MethodGet, "/status", nil))
}
