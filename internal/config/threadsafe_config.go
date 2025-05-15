package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// ThreadSafeConfig provides thread-safe access to configuration settings.
// It uses read-write locks to ensure safe concurrent access while
// maintaining the idiomatic Go approach to configuration management.
type ThreadSafeConfig struct {
	mu    sync.RWMutex
	cfg   Config
	ready bool
}

// NewThreadSafeConfig creates a new thread-safe configuration with default values.
// This is the starting point for creating a thread-safe configuration instance.
func NewThreadSafeConfig() *ThreadSafeConfig {
	return &ThreadSafeConfig{
		cfg: *New(),
	}
}

// WithVersionInfo sets the version info on the config and returns the config.
// This follows the builder pattern for configuration setup.
func (c *ThreadSafeConfig) WithVersionInfo(info VersionInfo) *ThreadSafeConfig {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ready {
		panic("cannot modify config after initialization")
	}

	c.cfg.VersionInfo = info
	return c
}

// Initialize prepares the configuration for use by loading from environment,
// parsing flags, and finalizing. This method should only be called once,
// typically during application startup.
func (c *ThreadSafeConfig) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ready {
		return fmt.Errorf("config already initialized")
	}

	// Load configuration from all sources
	c.cfg.LoadFromEnvironment()

	// Skip flag parsing in test mode
	if len(os.Args) > 0 && !strings.HasSuffix(os.Args[0], ".test") {
		if err := c.cfg.ParseFlags(); err != nil {
			return err
		}
	}

	if err := c.cfg.Finalize(); err != nil {
		return err
	}

	c.ready = true
	return nil
}

// Config returns a copy of the current configuration.
// This method is safe for concurrent use and can be called
// from any goroutine after Initialize() has been called.
func (c *ThreadSafeConfig) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ready {
		panic("config accessed before initialization")
	}

	// Return a copy to prevent modification of internal state
	return c.cfg
}

// PrintUsage prints a formatted help message with flag descriptions.
// This method is primarily used for command-line help and is thread-safe.
func (c *ThreadSafeConfig) PrintUsage(w io.Writer) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a temporary FlagSet just for printing usage
	fs := flag.NewFlagSet("gitbak", flag.ContinueOnError)
	c.cfg.SetupFlags(fs)
	c.cfg.PrintUsage(fs, w)
}

// IsReady returns whether the configuration has been successfully initialized
// and is ready for use.
func (c *ThreadSafeConfig) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}
