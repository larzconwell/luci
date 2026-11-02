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

// Server maintains the running state of an application.
type Server struct {
	config    Config
	app       Application
	logger    *slog.Logger
	server    *http.Server
	router    chi.Router
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
	router.Use(
		middleware.StripSlashes,
		withResponseWriter,
		withDuration,
		withID,
	)

	badRequestMiddlewares := append(Middlewares{
		withLogger(config.Logger),
		withTimeout(fallbackErrorRespond, config.RouteTimeout),
		withRecover(fallbackErrorRespond),
	}, app.Middlewares()...)

	router.MethodNotAllowed(badRequestMiddlewares.Handler(
		errorRespond(app.Error, http.StatusMethodNotAllowed, ErrMethodNotAllowed),
	).ServeHTTP)
	router.NotFound(badRequestMiddlewares.Handler(
		errorRespond(app.Error, http.StatusNotFound, ErrNotFound),
	).ServeHTTP)

	routes := app.Routes()
	server := &Server{
		config: config,
		app:    app,
		logger: config.Logger,
		server: &http.Server{
			Addr:              config.Address,
			Handler:           router,
			ReadHeaderTimeout: config.ReadHeaderTimeout,
		},
		router:    router,
		routes:    make(map[string]Route, len(routes)),
		serveDone: make(chan error, 1),
		started:   make(chan struct{}),
	}

	root := Route{Timeout: config.RouteTimeout}
	for _, route := range routes {
		server.registerRoute(route, root)
	}

	return server
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

func (server *Server) registerRoute(route Route, parent Route) {
	if len(route.Routes) > 0 {
		switch {
		case route.Name != "":
			panic(errors.New("luci: route group must not have a name"))
		case route.HandlerFunc != nil:
			panic(errors.New("luci: route group must not have a handler"))
		case route.Pattern == "":
			panic(errors.New("luci: route group must have a pattern"))
		case route.Pattern[0] != '/':
			panic(errors.New("luci: route group pattern must begin with /"))
		}

		if route.Timeout == 0 {
			route.Timeout = parent.Timeout
		}

		if route.Method == "" {
			route.Method = parent.Method
		}

		middlewares := make(Middlewares, 0, len(parent.Middlewares)+len(route.Middlewares))
		middlewares = append(middlewares, parent.Middlewares...)
		route.Middlewares = append(middlewares, route.Middlewares...)

		route.Pattern = parent.Pattern + route.Pattern

		for _, child := range route.Routes {
			server.registerRoute(child, route)
		}

		return
	}

	switch {
	case route.Name == "":
		panic(errors.New("luci: route must have a name"))
	case route.HandlerFunc == nil:
		panic(fmt.Errorf(`luci: route "%s" must have a handler`, route.Name))
	case route.Pattern == "":
		panic(fmt.Errorf(`luci: route "%s" must have a pattern`, route.Name))
	case route.Pattern[0] != '/':
		panic(fmt.Errorf(`luci: route "%s" pattern must begin with /`, route.Name))
	}

	_, ok := server.routes[route.Name]
	if ok {
		panic(fmt.Errorf(`luci: route "%s" already exists`, route.Name))
	}

	if route.Timeout == 0 {
		route.Timeout = parent.Timeout
	}

	if route.Method == "" {
		route.Method = parent.Method
	}

	middlewares := make(Middlewares, 0, len(parent.Middlewares)+len(route.Middlewares))
	middlewares = append(middlewares, parent.Middlewares...)
	route.Middlewares = append(middlewares, route.Middlewares...)

	route.Pattern = parent.Pattern + route.Pattern
	if route.Pattern != "/" && route.Pattern[len(route.Pattern)-1] == '/' {
		route.Pattern = route.Pattern[:len(route.Pattern)-1]
	}

	server.routes[route.Name] = route

	router := server.router.With(
		withVars,
		WithValue(requestRouteKey{}, route),
		withLogger(server.config.Logger),
		withTimeout(server.app.Error, route.Timeout),
		withRecover(server.app.Error),
	)

	for _, middleware := range server.app.Middlewares() {
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
}
