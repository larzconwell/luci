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
	config    Config
	app       Application
	logger    *slog.Logger
	server    *http.Server
	routes    map[string]Route
	serveDone chan error
	started   chan struct{}
	address   string
}

// NewServer creates a server for the given application using the given configuration.
// NewServer panics if any route does not have a name, the name is not unique, or if the
// route doesn't have a handler defined.
func NewServer(config Config, app Application) *Server {
	config = buildConfig(config)

	router := chi.NewRouter()

	appMiddlewares := app.Middlewares()
	badRequestMiddlewares := append(Middlewares{
		withResponseWriter,
		withDuration,
		withID,
		withLogger(config.Logger),
		withRecover(app.Error),
	}, appMiddlewares...)

	router.MethodNotAllowed(badRequestMiddlewares.Handler(
		errorRespond(app.Error, http.StatusMethodNotAllowed, ErrMethodNotAllowed),
	).ServeHTTP)
	router.NotFound(badRequestMiddlewares.Handler(
		errorRespond(app.Error, http.StatusNotFound, ErrNotFound),
	).ServeHTTP)

	routes := app.Routes()
	routesByName := make(map[string]Route, len(routes))

	for _, route := range routes {
		if route.Name == "" {
			panic(errors.New("luci: route must have a name"))
		}

		_, ok := routesByName[route.Name]
		if ok {
			panic(fmt.Errorf(`luci: route "%s" already exists`, route.Name))
		}

		if route.Pattern == "" {
			panic(fmt.Errorf(`luci: route "%s" must have a pattern`, route.Name))
		} else if route.Pattern[0] != '/' {
			panic(fmt.Errorf(`luci: route "%s" pattern must begin with /`, route.Name))
		}

		if route.HandlerFunc == nil {
			panic(fmt.Errorf(`luci: route "%s" must have a handler`, route.Name))
		}

		timeout := route.Timeout
		if timeout == 0 {
			timeout = config.RouteTimeout
		}

		router := router.With(
			withResponseWriter,
			withDuration,
			withID,
			withVars,
			WithValue(requestRouteKey{}, route),
			withLogger(config.Logger),
			withTimeout(app.Error, timeout),
			withRecover(app.Error),
		)

		for _, middleware := range appMiddlewares {
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
		Handler:           router,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}

	return &Server{
		config:    config,
		app:       app,
		logger:    config.Logger,
		server:    server,
		routes:    routesByName,
		serveDone: make(chan error, 1),
		started:   make(chan struct{}),
	}
}

// Route retrieves a defined route by name, and whether a route was found with the given name.
func (server *Server) Route(name string) (Route, bool) {
	route, ok := server.routes[name]
	return route, ok
}

// Address can be used to retrieve the address the server is listening on.
// Address blocks until the server has begun listening on the address.
func (server *Server) Address() string {
	<-server.started
	return server.address
}

// Run is a wrapper for (*luci.Server).Start and calling the returned wait
// function with the given context.
func (server *Server) Run(ctx context.Context) error {
	wait, err := server.Start(ctx)
	if err != nil {
		return err
	}

	return wait(ctx)
}

// Start listens on the configured address and begins serving requests in the background.
// The context given to Start is only used for the duration of creating the listener and
// does not affected the lifetime of the running server.
//
// Start returns a wait function and once called it will block until context cancellation
// occurs and begin a graceful shutdown up until the configured shutdown timeout has been
// reached, once it has been exceeded the server forcefully shuts down and
// ErrForcedShutdown is returned.
func (server *Server) Start(ctx context.Context) (func(context.Context) error, error) {
	var config net.ListenConfig

	listener, err := config.Listen(ctx, "tcp", server.config.Address)
	if err != nil {
		return nil, fmt.Errorf("luci: listen: %w", err)
	}
	// No need to explicitly close, handled via (*http.Server).Serve

	server.address = listener.Addr().String()
	close(server.started)

	server.logger = server.logger.WithGroup("server").With(slog.String("address", server.address))
	server.logger.Info("server started")

	go func() {
		err := server.server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		} else if err != nil {
			err = fmt.Errorf("luci: serve: %w", err)
		}

		server.serveDone <- err
	}()

	return server.wait, nil
}

func (server *Server) wait(ctx context.Context) error {
	// Wait for either (*http.Server).Serve to fail or for the context to finish.
	select {
	case err := <-server.serveDone:
		return err
	case <-ctx.Done():
	}

	server.logger.With(
		slog.String("timeout", server.config.ShutdownTimeout.String()),
	).Info("server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), server.config.ShutdownTimeout)
	defer cancel()

	// Disable context checking, we intentionally use a background context to build
	// the shutdown context, the top level context tells when to start the shutdown
	// but does not include the shutdown itself.
	//nolint:contextcheck
	//gosec:disable G118
	err := server.server.Shutdown(ctx)

	// If (*http.Server).Shutdown triggered (*http.Server).Serve to return there
	// should be no error, if there is the (*http.Server).Shutdown error is more
	// important to surface.
	//nolint:errcheck
	<-server.serveDone

	server.logger.Info("server shut down")

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrForcedShutdown
	} else if err != nil {
		return fmt.Errorf("luci: shutdown: %w", err)
	}

	return nil
}
