package luci

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestApplication struct {
	mock.Mock
}

func (ta *TestApplication) Routes() []Route {
	routes, _ := ta.MethodCalled("Routes").Get(0).([]Route)
	return routes
}

func (ta *TestApplication) Middlewares() []Middleware {
	middlewares, _ := ta.MethodCalled("Middlewares").Get(0).([]Middleware)
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

	t.Run("creates server", func(t *testing.T) {
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

	t.Run("adds routes", func(t *testing.T) {
		t.Parallel()

		baseMiddlewareCount := 2

		var app TestApplication
		app.On("Middlewares").Return([]Middleware{
			WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+1), true),
		})
		app.On("Routes").Return([]Route{
			{
				Name:    "all_status",
				Pattern: "/status",
				Middlewares: []Middleware{
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+2), true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
			{
				Name:    "get_status",
				Method:  http.MethodGet,
				Pattern: "/status",
				Middlewares: []Middleware{
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+2), true),
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+3), true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
			{
				Name:    "post_status",
				Method:  http.MethodPost,
				Pattern: "/status",
				Middlewares: []Middleware{
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+2), true),
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+3), true),
					WithValue(fmt.Sprintf("middleware_%d", baseMiddlewareCount+4), true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})

		server := NewServer(DefaultConfig, &app)
		mux, ok := server.server.Handler.(*chi.Mux)
		assert.True(t, ok)

		app.AssertExpectations(t)

		var count int
		var methods []string
		err := chi.Walk(mux, func(method string, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			methods = append(methods, method)
			assert.Equal(t, "/status", pattern)
			assert.NotNil(t, handler)

			switch method {
			case http.MethodGet:
				assert.Len(t, middlewares, baseMiddlewareCount+4)
			case http.MethodPost:
				assert.Len(t, middlewares, baseMiddlewareCount+5)
			default:
				assert.Len(t, middlewares, baseMiddlewareCount+3)
			}

			count++
			return nil
		})
		assert.NoError(t, err)

		assert.Greater(t, count, 2)
		assert.Len(t, methods, count)
		assert.Contains(t, methods, http.MethodGet)
		assert.Contains(t, methods, http.MethodPost)

		assert.Len(t, server.routes, 3)
		assert.Contains(t, server.routes, "all_status")
		assert.Contains(t, server.routes, "get_status")
		assert.Contains(t, server.routes, "post_status")
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

		assert.Panics(t, func() {
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

		assert.Panics(t, func() {
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

		assert.Panics(t, func() {
			NewServer(DefaultConfig, &app)
		})

		app.AssertExpectations(t)
	})

	t.Run("adds the current route middleware", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "status",
				Method:  http.MethodGet,
				Pattern: "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					route := RequestRoute(req)

					assert.Equal(t, "status", route.Name)
					assert.Equal(t, http.MethodGet, route.Method)
					assert.Equal(t, "/status", route.Pattern)
					assert.Empty(t, route.Middlewares)
					assert.NotNil(t, route.HandlerFunc)

					rw.WriteHeader(http.StatusOK)
				},
			},
		})

		server := NewServer(DefaultConfig, &app)
		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("adds the request vars middleware", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/user/abc123/post/abc123", nil)

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "user_post",
				Method:  http.MethodPost,
				Pattern: "/user/{user}/post/{post}",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					assert.Equal(t, map[string]string{
						"user": "abc123",
						"post": "abc123",
					}, RequestVars(req))

					rw.WriteHeader(http.StatusOK)
				},
			},
		})

		server := NewServer(DefaultConfig, &app)
		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("adds the request id middleware", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:    "status",
				Method:  http.MethodGet,
				Pattern: "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {
					assert.NotEmpty(t, RequestID(req))

					rw.WriteHeader(http.StatusOK)
				},
			},
		})

		server := NewServer(DefaultConfig, &app)
		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("sets the method not allowed handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/status", nil)

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:        "status",
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})
		app.On("Error", recorder, mock.Anything, http.StatusMethodNotAllowed, ErrMethodNotAllowed)

		server := NewServer(DefaultConfig, &app)
		server.server.Handler.ServeHTTP(recorder, request)

		app.AssertExpectations(t)
	})

	t.Run("sets the not found handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

		var app TestApplication
		app.On("Middlewares").Return(nil)
		app.On("Routes").Return([]Route{
			{
				Name:        "status",
				Method:      http.MethodGet,
				Pattern:     "/status",
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})
		app.On("Error", recorder, mock.Anything, http.StatusNotFound, ErrNotFound)

		server := NewServer(DefaultConfig, &app)
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
