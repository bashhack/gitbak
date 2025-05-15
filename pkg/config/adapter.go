package config

import (
	"fmt"
)

// ConfigProvider defines an interface for retrieving configuration.
// This allows us to use either a regular Config or ThreadSafeConfig
// interchangeably throughout the application.
type ConfigProvider interface {
	// Config returns the current configuration
	Config() Config
}

// ConfigAdapter implements the backward compatibility layer that
// allows existing code to continue working with Config while using
// the ThreadSafeConfig underneath.
type ConfigAdapter struct {
	provider ConfigProvider
}

// NewConfigAdapter creates an adapter for the given provider.
// Returns an error if the provider is nil.
func NewConfigAdapter(provider ConfigProvider) (*ConfigAdapter, error) {
	if provider == nil {
		return nil, fmt.Errorf("config provider cannot be nil")
	}
	return &ConfigAdapter{provider: provider}, nil
}

// Access forwards Config object access to the provider.
// This is a helper method that allows safe access to the underlying
// configuration, particularly useful during refactoring.
func (ca *ConfigAdapter) Access(access func(cfg Config) error) error {
	if ca.provider == nil {
		return fmt.Errorf("no config provider available")
	}

	return access(ca.provider.Config())
}

// GetRepoPath is an example of a convenient accessor.
// Accessors like this can be added as needed during refactoring
// to allow for gradual migration without breaking existing code.
func (ca *ConfigAdapter) GetRepoPath() string {
	return ca.provider.Config().RepoPath
}

// Migration helper for the standard Config type to implement ConfigProvider
// This allows using a regular Config as a ConfigProvider during transition.
func (c *Config) Config() Config {
	return *c
}
