// Package main is an example project utilizing luci to build an HTTP server with CRUD actions on user objects.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/larzconwell/luci/v2"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, sigRestore := signal.NotifyContext(context.Background(), os.Interrupt)
	defer sigRestore()

	// Restore the OS signal handler once we've gotten a signal
	// to allow for force stopping with further signals.
	go func() {
		<-ctx.Done()
		sigRestore()
	}()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	app := NewApplication(luci.Config{
		Address:         ":7879",
		ShutdownTimeout: time.Second,
		Logger:          logger,
	})

	err := app.server.Run(ctx)
	if err != nil {
		logger.With(slog.Any("error", err)).Error("failed to run server")
		return 1
	}

	return 0
}
