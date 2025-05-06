package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bashhack/gitbak/internal/config"
)

// Version information - injected at build time
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	versionInfo := config.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	app := NewDefaultApp(versionInfo)

	if err := app.Config.ParseFlags(); err != nil {
		_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
		app.exit(1)
	}

	// Initialize the app (logger, lock, etc.)
	if err := app.Initialize(); err != nil {
		_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
		app.exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-c
		fmt.Printf("\nReceived signal %v, stopping gitbak...\n", sig)

		// Cancel the context to signal graceful shutdown
		cancel()

		// Wait a short time for the application to respond to context cancellation
		// If it doesn't stop gracefully within a reasonable time, force termination
		go func() {
			time.Sleep(2 * time.Second)
			app.CleanupOnSignal()
			app.exit(0)
		}()
	}()

	// Run the application with the cancellable context
	if err := app.Run(ctx); err != nil {
		_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
		app.exit(1)
	}
}
