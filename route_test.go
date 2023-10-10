package luci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestRoute(t *testing.T) {
	t.Parallel()

	route := Route{
		Name:    "status",
		Method:  http.MethodGet,
		Pattern: "/status",
	}

	request := httptest.NewRequest(http.MethodGet, "/status", nil)
	request = request.WithContext(context.WithValue(request.Context(), routeContextKey{}, route))

	assert.Equal(t, route, RequestRoute(request))
}

func TestRouteString(t *testing.T) {
	t.Parallel()

	route := Route{
		Name:    "status",
		Method:  http.MethodGet,
		Pattern: "/status",
	}

	assert.Equal(t, "status GET /status", route.String())
}
