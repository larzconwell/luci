// Package main is an example project utilizing luci to build an HTTP server with CRUD actions on user objects.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/larzconwell/luci"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := NewApplication(luci.Config{
		Address:         ":7879",
		ShutdownTimeout: time.Second,
		Logger:          slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	})

	return app.ListenAndServe(ctx)
}
