package main

import (
	"context"
	"fmt"
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

	config := luci.Config{
		Address:           ":7879",
		ReadHeaderTimeout: time.Second,
		ShutdownTimeout:   time.Second,
	}

	app := NewApplication(config)
	return app.ListenAndServe(ctx)
}