package config

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"
)

func TestThreadSafeConfigBasics(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"test-binary"}
	defer func() { os.Args = oldArgs }()

	ts := NewThreadSafeConfig()

	if ts.IsReady() {
		t.Error("Expected config not to be ready before initialization")
	}

	versionInfo := VersionInfo{
		Version: "test-version",
		Commit:  "test-commit",
		Date:    "test-date",
	}
	ts = ts.WithVersionInfo(versionInfo)

	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	if !ts.IsReady() {
		t.Error("Expected config to be ready after initialization")
	}

	cfg := ts.Config()

	if cfg.VersionInfo.Version != "test-version" {
		t.Errorf("Expected version to be 'test-version', got '%s'", cfg.VersionInfo.Version)
	}

	var buf bytes.Buffer
	ts.PrintUsage(&buf)
	if buf.Len() == 0 {
		t.Error("Expected PrintUsage to write content to buffer")
	}
}

func TestThreadSafeConfigConcurrency(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"test-binary"}
	defer func() { os.Args = oldArgs }()

	ts := NewThreadSafeConfig()
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				cfg := ts.Config()

				if cfg.IntervalMinutes <= 0 {
					t.Errorf("Goroutine %d: Invalid IntervalMinutes value: %f",
						id, cfg.IntervalMinutes)
				}

				// Small random delay to increase the chance of race conditions
				if j%10 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestThreadSafeConfigInitializeOnce(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"test-binary"}
	defer func() { os.Args = oldArgs }()

	ts := NewThreadSafeConfig()

	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize config: %v", err)
	}

	err = ts.Initialize()
	if err == nil {
		t.Error("Expected error when initializing config second time, got nil")
	}
}
