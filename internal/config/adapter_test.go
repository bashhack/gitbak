package config

import (
	"errors"
	"testing"
)

func TestNewConfigAdapter(t *testing.T) {
	provider := NewThreadSafeConfig()
	adapter, err := NewConfigAdapter(provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}
	if adapter.provider != provider {
		t.Error("Adapter not initialized with correct provider")
	}

	// Test with nil provider
	adapter, err = NewConfigAdapter(nil)
	if err == nil {
		t.Error("Expected error when creating adapter with nil provider")
	}
	if adapter != nil {
		t.Error("Expected nil adapter when provider is nil")
	}
}

func TestConfigAdapterAccess(t *testing.T) {
	provider := NewThreadSafeConfig()

	provider.mu.Lock()
	provider.cfg.RepoPath = "/test/path"
	provider.ready = true
	provider.mu.Unlock()

	adapter, err := NewConfigAdapter(provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	accessCalled := false
	accessErr := adapter.Access(func(cfg Config) error {
		accessCalled = true
		if cfg.RepoPath != "/test/path" {
			t.Errorf("Expected RepoPath=/test/path, got %s", cfg.RepoPath)
		}
		return nil
	})

	if !accessCalled {
		t.Error("Access function was not called")
	}
	if accessErr != nil {
		t.Errorf("Unexpected error: %v", accessErr)
	}

	testErr := errors.New("test error")
	accessErr = adapter.Access(func(cfg Config) error {
		return testErr
	})
	if !errors.Is(testErr, accessErr) {
		t.Errorf("Expected %v, got %v", testErr, accessErr)
	}

	// Test with nil provider (this shouldn't happen in normal usage)
	badAdapter := &ConfigAdapter{provider: nil}
	badErr := badAdapter.Access(func(cfg Config) error {
		t.Error("This should not be called")
		return nil
	})
	if badErr == nil {
		t.Error("Expected error when provider is nil")
	}
}

func TestConfigAdapterGetRepoPath(t *testing.T) {
	provider := NewThreadSafeConfig()

	provider.mu.Lock()
	provider.cfg.RepoPath = "/test/path"
	provider.ready = true
	provider.mu.Unlock()

	adapter, err := NewConfigAdapter(provider)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	path := adapter.GetRepoPath()
	if path != "/test/path" {
		t.Errorf("Expected RepoPath=/test/path, got %s", path)
	}
}

func TestConfigImplementsConfigProvider(t *testing.T) {
	cfg := New()
	cfg.RepoPath = "/config/path"

	var provider ConfigProvider = cfg

	result := provider.Config()

	if result.RepoPath != "/config/path" {
		t.Errorf("Expected RepoPath=/config/path, got %s", result.RepoPath)
	}
}
