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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestApplication struct {
	mock.Mock
}

func (ta *TestApplication) Routes() map[string]Route {
	routes, _ := ta.MethodCalled("Routes").Get(0).(map[string]Route)
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
		app.On("Routes").Return(map[string]Route{})

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
	})

	t.Run("adds routes", func(t *testing.T) {
		t.Parallel()

		var app TestApplication
		app.On("Middlewares").Return([]Middleware{
			middleware.WithValue("middleware_1", true),
		})
		app.On("Routes").Return(map[string]Route{
			"get_status": {
				Method:  http.MethodGet,
				Pattern: "/status",
				Middlewares: []Middleware{
					middleware.WithValue("middleware_2", true),
					middleware.WithValue("middleware_3", true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
			"post_status": {
				Method:  http.MethodPost,
				Pattern: "/status",
				Middlewares: []Middleware{
					middleware.WithValue("middleware_2", true),
				},
				HandlerFunc: func(rw http.ResponseWriter, req *http.Request) {},
			},
		})

		server := NewServer(DefaultConfig, &app)
		mux, ok := server.server.Handler.(*chi.Mux)
		assert.True(t, ok)

		app.AssertExpectations(t)

		var count int
		err := chi.Walk(mux, func(method string, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			assert.Equal(t, "/status", pattern)
			assert.NotNil(t, handler)
			assert.Contains(t, []string{http.MethodGet, http.MethodPost}, method)

			if method == http.MethodGet {
				assert.Len(t, middlewares, 4) // The server adds middleware
			} else {
				assert.Len(t, middlewares, 3) // The server adds middleware
			}

			count++
			return nil
		})
		assert.NoError(t, err)

		app.AssertExpectations(t)
		assert.Equal(t, 2, count)
	})

	t.Run("adds the current route middleware", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/status", nil)

		var app TestApplication
		app.On("Middlewares").Return([]Middleware{})
		app.On("Routes").Return(map[string]Route{
			"status": {
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

	t.Run("sets the method not allowed handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/status", nil)

		var app TestApplication
		app.On("Middlewares").Return([]Middleware{})
		app.On("Routes").Return(map[string]Route{
			"status": {
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
		app.On("Middlewares").Return([]Middleware{})
		app.On("Routes").Return(map[string]Route{
			"status": {
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
		app.On("Middlewares").Return([]Middleware{})
		app.On("Routes").Return(map[string]Route{
			"status": {
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
		app.On("Middlewares").Return([]Middleware{})
		app.On("Routes").Return(map[string]Route{
			"status": {
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
