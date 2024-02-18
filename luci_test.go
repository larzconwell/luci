package luci

import (
	"io"
	"log/slog"
)

var (
	noopLogger = slog.New(slog.NewTextHandler(io.Discard, new(slog.HandlerOptions)))
	testConfig = buildConfig(Config{
		Address: ":0",
		Logger:  noopLogger,
	})
)
