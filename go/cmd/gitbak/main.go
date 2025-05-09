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
		time.Sleep(5 * time.Second)

		// If we're still running after the timeout, force cleanup and exit
		select {
		case <-ctx.Done():
			// Context was properly canceled, main function will handle cleanup
			return
		default:
			// Force cleanup and exit if context cancellation didn't work
			app.CleanupOnSignal()
			app.exit(0)
		}
	}()

	// Run the application with the cancellable context
	if err := app.Run(ctx); err != nil {
		// Don't treat context cancellation as an error since that's our normal signal shutdown path
		if err.Error() != "context canceled" {
			_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
			_ = app.Close()
			app.exit(1)
		}
	}

	// Print summary only if we ran the main gitbak process (not for --logo or --version)
	// and if the gitbak instance was initialized
	if !app.Config.ShowLogo && !app.Config.Version && app.Gitbak != nil {
		app.Gitbak.PrintSummary()
	}
	_ = app.Close()
}
