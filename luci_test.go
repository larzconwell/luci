package luci

import (
	"log/slog"
)

var (
	noopLogger = slog.New(slog.DiscardHandler)
	testConfig = buildConfig(Config{
		Address: ":0",
		Logger:  noopLogger,
	})
)
