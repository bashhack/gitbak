//go:build testing

package config

import (
	"flag"
)

// SetupTestFlags adds test-specific flags to the flag set
// This function is only included in builds with the "testing" tag
func (c *Config) SetupTestFlags(fs *flag.FlagSet) {
	// Add test-specific flags here
	fs.BoolVar(&c.NonInteractive, "non-interactive", c.NonInteractive,
		"Skip all interactive prompts (for testing automation)")
}
