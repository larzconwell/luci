package luci

import (
	"context"
	"errors"
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
	request = request.WithContext(context.WithValue(request.Context(), requestRouteKey{}, route))

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

func TestRoutePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		route        Route
		vals         []string
		expectedPath string
		expectedErr  error
	}{
		{
			route:        Route{Pattern: "/health"},
			expectedPath: "/health",
		},
		{
			route:        Route{Pattern: "/static/*"},
			vals:         []string{"css/index.css"},
			expectedPath: "/static/css/index.css",
		},
		{
			route:        Route{Pattern: "/media/{}"},
			vals:         []string{"image_123.jpg"},
			expectedPath: "/media/image_123.jpg",
		},
		{
			route:        Route{Pattern: "/{resource}"},
			vals:         []string{"admin"},
			expectedPath: "/admin",
		},
		{
			route:        Route{Pattern: "/media/random files/{file}"},
			vals:         []string{"luci 123.jpg"},
			expectedPath: "/media/random%20files/luci%20123.jpg",
		},
		{
			route:        Route{Pattern: "/user/{user}"},
			vals:         []string{"luci"},
			expectedPath: "/user/luci",
		},
		{
			route:        Route{Pattern: "/user/{user}/edit"},
			vals:         []string{"luci"},
			expectedPath: "/user/luci/edit",
		},
		{
			route:        Route{Pattern: "/user/{user}/post/{post}"},
			vals:         []string{"luci", "123"},
			expectedPath: "/user/luci/post/123",
		},
		{
			route:        Route{Pattern: "/user/{user}/static/*"},
			vals:         []string{"luci", "js/luci.js"},
			expectedPath: "/user/luci/static/js/luci.js",
		},
		{
			route:        Route{Pattern: "/user/{user}/post/{post}/static/*"},
			vals:         []string{"luci", "123", "js/luci.js"},
			expectedPath: "/user/luci/post/123/static/js/luci.js",
		},
		{
			route:        Route{Pattern: "/search/user/{first}_{last}"},
			vals:         []string{"luci", "last"},
			expectedPath: "/search/user/luci_last",
		},
		{
			route:        Route{Pattern: "/search/user/{first}_*"},
			vals:         []string{"luci", "last"},
			expectedPath: "/search/user/luci_last",
		},
		{
			route:        Route{Pattern: `/search/date/{date:\d\d\d\d-\d\d-\d\d}`},
			vals:         []string{"2023-10-28"},
			expectedPath: "/search/date/2023-10-28",
		},
		{
			route:        Route{Pattern: "/anon/{:[a-z]+[0-9]*}"},
			vals:         []string{"abc123"},
			expectedPath: "/anon/abc123",
		},
		{
			expectedErr: errors.New("luci: route pattern must not be empty"),
		},
		{
			route:       Route{Pattern: "pattern"},
			expectedErr: errors.New("luci: route pattern must begin with /"),
		},
		{
			route:       Route{Pattern: "/user/{user"},
			vals:        []string{"luci"},
			expectedErr: errors.New("luci: invalid route pattern"),
		},
		{
			route:       Route{Pattern: "/user/{user}"},
			expectedErr: errors.New("luci: must provide the expected number of values (expected 1 received 0)"),
		},
		{
			route:       Route{Pattern: "/user/{user}"},
			vals:        []string{"luci", "last"},
			expectedErr: errors.New("luci: must provide the expected number of values (expected 1 received 2)"),
		},
		{
			route:       Route{Pattern: "/user/{user}/static/*"},
			vals:        []string{"luci", "last", "js/luci.js"},
			expectedErr: errors.New("luci: must provide the expected number of values (expected 2 received 3)"),
		},
		{
			route:       Route{Pattern: "/user/{user}"},
			vals:        []string{"luci/last"},
			expectedErr: errors.New(`luci: value for variable "user" must not contain /`),
		},
		{
			route:       Route{Pattern: "/invalid_regex/{:[}"},
			vals:        []string{"abc"},
			expectedErr: errors.New("luci: variable \"\" must have valid regex: error parsing regexp: missing closing ]: `[`"),
		},
		{
			route:       Route{Pattern: `/date/{date:\d\d\d\d-\d\d-\d\d}`},
			vals:        []string{"does_not_match"},
			expectedErr: errors.New(`luci: value for variable "date" does not match regex`),
		},
	}

	for idx, test := range tests {
		path, err := test.route.Path(test.vals...)

		if test.expectedErr != nil {
			assert.Error(t, err, idx)
			if err != nil {
				assert.Equal(t, test.expectedErr.Error(), err.Error(), idx)
			}
		} else {
			assert.NoError(t, err)
		}

		assert.Equal(t, test.expectedPath, path, idx)
	}
}
