package luci

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Server maintains the running state of an application.
type Server struct {
	config  Config
	app     Application
	logger  *slog.Logger
	server  *http.Server
	routes  map[string]Route
	started chan struct{}
	address string
}

// NewServer creates a server for the given application using the given configuration.
// NewServer panics if any route does not have a name, the name is not unique, or if the
// route doesn't have a handler defined.
func NewServer(config Config, app Application) *Server {
	config = buildConfig(config)

	mux := chi.NewMux()
	mux.MethodNotAllowed(errorRespond(app.Error, http.StatusMethodNotAllowed, ErrMethodNotAllowed))
	mux.NotFound(errorRespond(app.Error, http.StatusNotFound, ErrNotFound))

	middlewares := app.Middlewares()
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
			withResponseWriterWrapper,
			WithValue(requestRouteKey{}, route),
			withRequestVars,
			withID(app.Error),
			withLogger(config.Logger),
		)

		for _, middleware := range middlewares {
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

// ListenAndServe listens on the configured address and serves requests until the given context has been cancelled.
// ListenAndServe will gracefully shutdown on context cancellation up until the configured shutdown timeout has been
// reached, if the shutdown timeout is reached ErrForcedShutdown is returned.
func (server *Server) ListenAndServe(ctx context.Context) error {
	var config net.ListenConfig

	listener, err := config.Listen(ctx, "tcp", server.config.Address)
	if err != nil {
		return fmt.Errorf("luci: listen: %w", err)
	}
	// listener closed via (*http.Server).Serve

	server.address = listener.Addr().String()
	close(server.started)

	logger := server.logger.WithGroup("server").With(slog.String("address", server.address))
	logger.Info("Server started")

	done := make(chan error, 1)
	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}

		logger.With(
			slog.String("timeout", server.config.ShutdownTimeout.String()),
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

// Address can be used to retrieve the address the server is listening on.
// Address blocks until the server has begun listening on the address.
func (server *Server) Address() string {
	<-server.started
	return server.address
}

// Route retrieves a defined route by name, and whether a route was found with the given name.
func (server *Server) Route(name string) (Route, bool) {
	route, ok := server.routes[name]
	return route, ok
}
