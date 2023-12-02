package luci

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func contextHasKey(ctx context.Context, key any) bool {
	return ctx.Value(key) != nil
}

type TestApplication struct {
	mock.Mock
}

func (ta *TestApplication) Routes() []Route {
	routes, _ := ta.MethodCalled("Routes").Get(0).([]Route)
	return routes
}

func (ta *TestApplication) Middlewares() Middlewares {
	middlewares, _ := ta.MethodCalled("Middlewares").Get(0).(Middlewares)
	return middlewares
}

func (ta *TestApplication) Error(rw http.ResponseWriter, req *http.Request, status int, err error) {
	ta.MethodCalled("Error", rw, req, status, err)
}

func (ta *TestApplication) Respond(rw http.ResponseWriter, req *http.Request, value any) {
	ta.MethodCalled("Respond", rw, req, value)
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	t.Run("returns server", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return(nil)

		server := NewServer(DefaultConfig, &app)

		app.AssertExpectations(t)
		assert.Equal(t, DefaultConfig, server.config)
		assert.Equal(t, &app, server.app)
		assert.Equal(t, DefaultConfig.Logger, server.logger)
		assert.Equal(t, &http.Server{
			Addr:              DefaultConfig.Address,
			Handler:           server.server.Handler,
			ReadHeaderTimeout: DefaultConfig.ReadHeaderTimeout,
		}, server.server)
		assert.NotNil(t, server.server.Handler)
		assert.NotNil(t, server.routes)
		assert.NotNil(t, server.started)
	})

	t.Run("panics if route has no name", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})

		assert.PanicsWithError(t, "luci: route must have a name", func() {
			NewServer(DefaultConfig, &app)
		})

		app.AssertExpectations(t)
	})

	t.Run("panics if route names are not unique", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:        "status",
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
			{
				Name:        "status",
				Method:      http.MethodPost,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})

		assert.PanicsWithError(t, `luci: route "status" already exists`, func() {
			NewServer(DefaultConfig, &app)
		})

		app.AssertExpectations(t)
	})

	t.Run("panics if route has no handler", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "status",
				Method:  http.MethodGet,
				Pattern: "/status",
			},
		})

		assert.PanicsWithError(t, `luci: route "status" must have a handler`, func() {
			NewServer(DefaultConfig, &app)
		})

		app.AssertExpectations(t)
	})

	t.Run("add routes", func(t *testing.T) {
		t.Parallel()

		routeAsserts := func(t *testing.T, rw http.ResponseWriter, req *http.Request) {
			assert.IsType(t, new(responseWriter), rw)

			assert.True(t, contextHasKey(req.Context(), durationKey{}), "duration key")
			assert.True(t, contextHasKey(req.Context(), idKey{}), "id key")
			assert.True(t, contextHasKey(req.Context(), varsKey{}), "vars key")
			assert.True(t, contextHasKey(req.Context(), requestRouteKey{}), "request route key")
			assert.True(t, contextHasKey(req.Context(), loggerKey{}), "logger key")
			assert.True(t, contextHasKey(req.Context(), "app_middleware"), "app key")
			assert.True(t, contextHasKey(req.Context(), "route_middleware"), "route key")
		}

		var app TestApplication
		app.On("Middlewares").Return(Middlewares{
			WithValue("app_middleware", true),
		})
		app.On("Routes").Return([]Route{
			{
				Name:    "all_status",
				Pattern: "/status",
				Middlewares: Middlewares{
					WithValue("route_middleware", true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					routeAsserts(t, rw, req)

					rw.WriteHeader(http.StatusOK)
				},
			},
			{
				Name:    "get_status",
				Method:  http.MethodGet,
				Pattern: "/status",
				Middlewares: Middlewares{
					WithValue("route_middleware", true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					routeAsserts(t, rw, req)

					rw.WriteHeader(http.StatusCreated)
				},
			},
			{
				Name:    "post_status",
				Method:  http.MethodPost,
				Pattern: "/status",
				Middlewares: Middlewares{
					WithValue("route_middleware", true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					routeAsserts(t, rw, req)

					rw.WriteHeader(http.StatusAccepted)
				},
			},
		})

		server := NewServer(DefaultConfig, &app)
		app.AssertExpectations(t)

		methods := []string{
			http.MethodConnect,
			http.MethodDelete,
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
			http.MethodPatch,
			http.MethodPost,
			http.MethodPut,
			http.MethodTrace,
		}

		for _, method := range methods {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/status", nil)

			server.server.Handler.ServeHTTP(recorder, request)

			expected := http.StatusOK
			if method == http.MethodGet {
				expected = http.StatusCreated
			} else if method == http.MethodPost {
				expected = http.StatusAccepted
			}

			assert.Equal(t, expected, recorder.Code, method)
		}
	})

	t.Run("handles method not allowed", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(Middlewares{
			WithValue("app_middleware", true),
		})
		app.On("Routes").Return([]Route{
			{
				Name:        "get_status",
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})
		app.On("Error", mock.Anything, mock.Anything, http.StatusMethodNotAllowed, ErrMethodNotAllowed).Run(func(args mock.Arguments) {
			rw, ok := args.Get(0).(http.ResponseWriter)
			assert.True(t, ok, "1st argument should be a response writer")
			req, ok := args.Get(1).(*http.Request)
			assert.True(t, ok, "2nd argument should be a request")

			assert.IsType(t, new(responseWriter), rw)

			assert.True(t, contextHasKey(req.Context(), durationKey{}), "duration key")
			assert.True(t, contextHasKey(req.Context(), idKey{}), "id key")
			assert.True(t, contextHasKey(req.Context(), loggerKey{}), "logger key")
			assert.True(t, contextHasKey(req.Context(), "app_middleware"), "app key")
		})

		server := NewServer(DefaultConfig, &app)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/status", nil)

		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
	})

	t.Run("handles not found", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(Middlewares{
			WithValue("app_middleware", true),
		})
		app.On("Routes").Return([]Route{
			{
				Name:        "get_status",
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})
		app.On("Error", mock.Anything, mock.Anything, http.StatusNotFound, ErrNotFound).Run(func(args mock.Arguments) {
			rw, ok := args.Get(0).(http.ResponseWriter)
			assert.True(t, ok, "1st argument should be a response writer")
			req, ok := args.Get(1).(*http.Request)
			assert.True(t, ok, "2nd argument should be a request")

			assert.IsType(t, new(responseWriter), rw)

			assert.True(t, contextHasKey(req.Context(), durationKey{}), "duration key")
			assert.True(t, contextHasKey(req.Context(), idKey{}), "id key")
			assert.True(t, contextHasKey(req.Context(), loggerKey{}), "logger key")
			assert.True(t, contextHasKey(req.Context(), "app_middleware"), "app key")
		})

		server := NewServer(DefaultConfig, &app)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/notfound", nil)

		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
	})
}

func TestServerListenAndServe(t *testing.T) {
	t.Parallel()

	t.Run("listens on the configured address until the context is canceled", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "status",
				Method:  http.MethodGet,
				Pattern: "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					rw.WriteHeader(http.StatusOK)
				},
			},
		})

		server := NewServer(Config{Address: ":0"}, &app)
		ctx, cancel := context.WithCancel(context.Background())
		listenErr := make(chan error, 1)

		go func() {
			listenErr <- server.ListenAndServe(ctx)
		}()

		addr := server.Address()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://%s/status", addr), nil)
		assert.NoError(t, err)

		res, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		cancel()
		assert.NoError(t, <-listenErr)

		res, err = http.DefaultClient.Do(req)
		if res != nil {
			res.Body.Close()
		}

		opErr := new(net.OpError)
		assert.ErrorAs(t, err, &opErr)
		assert.Equal(t, "dial", opErr.Op)

		app.AssertExpectations(t)
	})

	t.Run("returns error when server shutdown takes longer than the timeout allows", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "status",
				Method:  http.MethodGet,
				Pattern: "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					cancel()
					<-time.After(200 * time.Millisecond)
					rw.WriteHeader(http.StatusOK)
				},
			},
		})

		server := NewServer(Config{
			Address:         ":0",
			ShutdownTimeout: time.Millisecond,
		}, &app)
		listenErr := make(chan error, 1)

		go func() {
			listenErr <- server.ListenAndServe(ctx)
		}()

		addr := server.Address()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://%s/status", addr), nil)
		assert.NoError(t, err)

		res, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode)

		assert.ErrorIs(t, <-listenErr, ErrForcedShutdown)

		app.AssertExpectations(t)
	})
}

func TestServerRoute(t *testing.T) {
	t.Parallel()

	var app TestApplication
	app.On("Middlewares").Return(nil)
	app.On("Routes").Return([]Route{
		{
			Name:        "all_status",
			Pattern:     "/status",
			HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
		},
		{
			Name:        "get_status",
			Method:      http.MethodGet,
			Pattern:     "/status",
			HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
		},
	})

	server := NewServer(DefaultConfig, &app)

	app.AssertExpectations(t)

	route, ok := server.Route("all_status")
	assert.True(t, ok)
	assert.Equal(t, "all_status", route.Name)
	assert.Empty(t, route.Method)

	route, ok = server.Route("get_status")
	assert.True(t, ok)
	assert.Equal(t, "get_status", route.Name)
	assert.Equal(t, http.MethodGet, route.Method)

	_, ok = server.Route("nonexistent_route")
	assert.False(t, ok)
}
