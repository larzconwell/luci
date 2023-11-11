package luci

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	config  Config
	app     Application
	logger  *slog.Logger
	server  *http.Server
	routes  map[string]Route
	started chan struct{}
	address string
}

func NewServer(config Config, app Application) *Server {
	config = buildConfig(config)

	mux := chi.NewMux()
	mux.MethodNotAllowed(errorRespond(app.Error, http.StatusMethodNotAllowed, ErrMethodNotAllowed))
	mux.NotFound(errorRespond(app.Error, http.StatusNotFound, ErrNotFound))

	routes := app.Routes()
	routesByName := make(map[string]Route, len(routes))
	for _, route := range routes {
		if route.Name == "" {
			panic("luci: route must have a name")
		}

		_, ok := routesByName[route.Name]
		if ok {
			panic("luci: routes must have unique names")
		}

		if route.HandlerFunc == nil {
			panic("luci: route must have a handler")
		}

		router := mux.With(
			middleware.WithValue(routeContextKey{}, route),
			withRequestVars,
		)

		for _, middleware := range app.Middlewares() {
			router.Use(middleware)
		}

		for _, middleware := range route.Middlewares {
			router.Use(middleware)
		}

		if route.Method == "" {
			router.HandleFunc(route.Pattern, route.HandlerFunc)
		} else {
			router.MethodFunc(route.Method, route.Pattern, route.HandlerFunc)
		}

		routesByName[route.Name] = route
	}

	server := &http.Server{
		Addr:              config.Address,
		Handler:           mux,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}

	return &Server{
		config:  config,
		app:     app,
		logger:  config.Logger,
		server:  server,
		routes:  routesByName,
		started: make(chan struct{}),
	}
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	var config net.ListenConfig

	listener, err := config.Listen(ctx, "tcp", server.config.Address)
	if err != nil {
		return fmt.Errorf("luci: listen: %w", err)
	}
	// listener closed via (*http.Server).Serve

	server.address = listener.Addr().String()
	close(server.started)

	logger := server.logger.With(slog.String("address", server.address))
	logger.Info("Server started")

	done := make(chan error, 1)
	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}

		logger.With(
			slog.Duration("timeout", server.config.ShutdownTimeout),
		).Info("Server closing")

		ctx, cancel := context.WithTimeout(context.Background(), server.config.ShutdownTimeout)
		defer cancel()

		done <- server.server.Shutdown(ctx)
	}()

	err = server.server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		close(done)
		return fmt.Errorf("luci: serve: %w", err)
	}

	err = <-done
	logger.Info("Server closed")

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrForcedShutdown
	} else if err != nil {
		return fmt.Errorf("luci: shutdown: %w", err)
	}

	return nil
}

func (server *Server) Address() string {
	<-server.started
	return server.address
}

func (server *Server) Route(name string) (Route, bool) {
	route, ok := server.routes[name]
	return route, ok
}
