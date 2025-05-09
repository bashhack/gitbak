//go:build !testing

package config

import (
	"flag"
	"os"
)

// SetupTestFlags conditionally adds test-specific flags to the flag set
// This function is included in builds without the "testing" tag.
func (c *Config) SetupTestFlags(fs *flag.FlagSet) {
	// For compatibility with existing tests, still honor GITBAK_TESTING=1
	// environment variable to include test-specific flags
	if os.Getenv("GITBAK_TESTING") == "1" {
		fs.BoolVar(&c.NonInteractive, "non-interactive", c.NonInteractive,
			"Skip all interactive prompts (for testing automation)")
	}
	// Otherwise no test-specific flags in production builds
}
