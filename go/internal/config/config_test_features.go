//go:build testing

package config

import (
	"flag"
)

// SetupTestFlags adds test-specific flags to the flag set when built with the "testing" tag
func (c *Config) SetupTestFlags(fs *flag.FlagSet) {
	// Always include test-specific flags in test builds
	fs.BoolVar(&c.NonInteractive, "non-interactive", c.NonInteractive,
		"Skip all interactive prompts (for testing automation)")
}
