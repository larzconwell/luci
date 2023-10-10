package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestRequestParams(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key_1", "value")
	ctx.URLParams.Add("key_2", "value")

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, &ctx))

	assert.Equal(t, map[string]string{
		"key_1": "value",
		"key_2": "value",
	}, RequestParams(request))
}

func TestRequestParam(t *testing.T) {
	t.Parallel()

	var ctx chi.Context
	ctx.URLParams.Add("key", "value")

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, &ctx))

	assert.Equal(t, "value", RequestParam(request, "key"))
	assert.Empty(t, RequestParam(request, "nonexistent_key"))
}
