package luci

import (
	"log/slog"
	"time"
)

var (
	DefaultConfig = Config{ //nolint:gochecknoglobals
		Address:           ":http",
		ReadHeaderTimeout: time.Second,
		ShutdownTimeout:   5 * time.Second,
		Logger:            slog.Default(),
	}
)

type Config struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
	Logger            *slog.Logger
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
