package luci

import (
	"log/slog"
	"time"
)

var (
	// DefaultConfig is the base configuration that's used when creating a server.
	DefaultConfig = Config{
		Address:           ":http",
		ReadHeaderTimeout: time.Second,
		ShutdownTimeout:   5 * time.Second,
		Logger:            slog.Default().WithGroup("server"),
	}
)

// Config defines how a server should behave when it's running.
// See DefaultConfig for configuration defaults.
type Config struct {
	// Address defines the address to listen on.
	Address string
	// ReadHeaderTimeout defines the timeout to read request headers.
	// See net/http.Server.ReadHeaderTimeout for details.
	ReadHeaderTimeout time.Duration
	// ShutdownTimeout defines the timeout for the server to gracefully
	// shutdown on context cancellation.
	ShutdownTimeout time.Duration
	// Logger defines the logger the server uses when logging startup/shutdown/requests.
	// It is recommended to pass in a logger group via log/slog.Logger.WithGroup.
	Logger *slog.Logger
}

func buildConfig(config Config) Config {
	built := DefaultConfig

	if config.Address != "" {
		built.Address = config.Address
	}

	if config.ReadHeaderTimeout != 0 {
		built.ReadHeaderTimeout = config.ReadHeaderTimeout
	}

	if config.ShutdownTimeout != 0 {
		built.ShutdownTimeout = config.ShutdownTimeout
	}

	if config.Logger != nil {
		built.Logger = config.Logger
	}

	return built
}
