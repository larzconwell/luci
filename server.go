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
	started chan struct{}
	address string
}

func NewServer(config Config, app Application) *Server {
	config = buildConfig(config)

	mux := chi.NewMux()
	mux.MethodNotAllowed(errorRespond(app.Error, http.StatusMethodNotAllowed, ErrMethodNotAllowed))
	mux.NotFound(errorRespond(app.Error, http.StatusNotFound, ErrNotFound))

	for name, route := range app.Routes() {
		route.Name = name
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

		router.MethodFunc(route.Method, route.Pattern, route.HandlerFunc)
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
		started: make(chan struct{}),
	}
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	var config net.ListenConfig

	listener, err := config.Listen(ctx, "tcp", server.config.Address)
	if err != nil {
		return fmt.Errorf("Listen: %w", err)
	}
	// listener closed via (*http.Server).Serve

	server.address = listener.Addr().String()
	close(server.started)

	logger := server.logger.With("address", server.address)
	logger.Info("Server started")

	done := make(chan error, 1)
	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}

		logger.Info("Server closing", "timeout", server.config.ShutdownTimeout.String())

		ctx, cancel := context.WithTimeout(context.Background(), server.config.ShutdownTimeout)
		defer cancel()

		done <- server.server.Shutdown(ctx)
	}()

	err = server.server.Serve(listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		close(done)
		return fmt.Errorf("Serve: %w", err)
	}

	err = <-done
	logger.Info("Server closed")

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrForcedShutdown
	} else if err != nil {
		return fmt.Errorf("Shutdown: %w", err)
	}

	return nil
}

func (server *Server) Address() string {
	<-server.started
	return server.address
}
