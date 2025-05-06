package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/logger"
)

// mockExit helps test process exit handling
type mockExit struct {
	code   int
	called bool
}

func (m *mockExit) Exit(code int) {
	m.code = code
	m.called = true
}

// mockExecLookPath helps test command availability checks
type mockExecLookPath struct {
	lookupMap map[string]string
	lookupErr map[string]error
}

func (m *mockExecLookPath) LookPath(file string) (string, error) {
	if err, ok := m.lookupErr[file]; ok {
		return "", err
	}
	if path, ok := m.lookupMap[file]; ok {
		return path, nil
	}
	return "", fmt.Errorf("command not found: %s", file)
}

func TestNewDefaultApp(t *testing.T) {
	versionInfo := config.VersionInfo{
		Version: "test",
		Commit:  "test-commit",
		Date:    "test-date",
	}

	app := NewDefaultApp(versionInfo)

	if app.Config.VersionInfo.Version != "test" {
		t.Errorf("Expected Version=test, got %s", app.Config.VersionInfo.Version)
	}
	if app.Config.VersionInfo.Commit != "test-commit" {
		t.Errorf("Expected Commit=test-commit, got %s", app.Config.VersionInfo.Commit)
	}
	if app.Config.VersionInfo.Date != "test-date" {
		t.Errorf("Expected Date=test-date, got %s", app.Config.VersionInfo.Date)
	}

	if app.Stdout == nil {
		t.Error("Expected Stdout to be set, got nil")
	}
	if app.Stderr == nil {
		t.Error("Expected Stderr to be set, got nil")
	}
	if app.execLookPath == nil {
		t.Error("Expected execLookPath to be set, got nil")
	}
	if app.exit == nil {
		t.Error("Expected exit to be set, got nil")
	}
}

func TestAppShowVersion(t *testing.T) {
	versionInfo := config.VersionInfo{
		Version: "test",
		Commit:  "abc123",
		Date:    "2023-01-01",
	}

	var stdout bytes.Buffer

	mockExit := &mockExit{}

	cfg := config.New()
	cfg.VersionInfo = versionInfo

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
		Exit:   mockExit.Exit,
	})

	app.ShowVersion()

	expected := "gitbak test (abc123) built on 2023-01-01\n"
	if stdout.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, stdout.String())
	}
}

func TestAppShowLogo(t *testing.T) {
	var stdout bytes.Buffer

	cfg := config.New()
	cfg.VersionInfo = config.VersionInfo{}

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
	})

	app.ShowLogo()

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("Automatic Commit Safety Net")) {
		t.Errorf("Logo output does not contain expected tagline: %s", output)
	}
}

func TestAppCheckRequiredCommands(t *testing.T) {
	mockLookPath := &mockExecLookPath{
		lookupMap: map[string]string{
			"git":    "/usr/bin/git",
			"grep":   "/usr/bin/grep",
			"sed":    "/usr/bin/sed",
			"shasum": "/usr/bin/shasum",
		},
		lookupErr: map[string]error{},
	}

	app := NewApp(AppOptions{
		Config:       config.New(),
		ExecLookPath: mockLookPath.LookPath,
	})

	err := app.checkRequiredCommands()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	mockLookPath.lookupMap = map[string]string{
		"grep":   "/usr/bin/grep",
		"sed":    "/usr/bin/sed",
		"shasum": "/usr/bin/shasum",
	}
	mockLookPath.lookupErr = map[string]error{
		"git": fmt.Errorf("command not found"),
	}

	app = NewApp(AppOptions{
		Config:       config.New(),
		ExecLookPath: mockLookPath.LookPath,
	})

	err = app.checkRequiredCommands()
	if err == nil {
		t.Error("Expected error when git is missing, got nil")
	}
}

func TestRunWithLogoFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cfg := config.New()
	cfg.ShowLogo = true

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	ctx := context.Background()
	err := app.Run(ctx)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("Automatic Commit Safety Net")) {
		t.Errorf("Logo output does not contain expected tagline: %s", output)
	}

	if stderr.String() != "" {
		t.Errorf("Expected empty stderr, got: %s", stderr.String())
	}
}

// TestCleanupOnSignal provides comprehensive test coverage for the CleanupOnSignal method
func TestCleanupOnSignal(t *testing.T) {
	// Test 1: All components are nil (most basic case)
	t.Run("All components nil", func(t *testing.T) {
		app := NewApp(AppOptions{
			Config: config.New(),
			Locker: nil,
			Gitbak: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()
		// No assertions - we're just making sure it doesn't panic
	})

	// Test 2: Only Gitbak is present
	t.Run("Only Gitbak present", func(t *testing.T) {
		mockGitbak := &MockGitbaker{}

		app := NewApp(AppOptions{
			Config: config.New(),
			Gitbak: mockGitbak,
			Locker: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()

		if !mockGitbak.SummaryCalled {
			t.Error("Expected PrintSummary to be called, but it wasn't")
		}
	})

	// Test 3: Locker with success
	t.Run("Locker with success", func(t *testing.T) {
		mockLocker := &MockLocker{ReleaseErr: nil}

		app := NewApp(AppOptions{
			Config: config.New(),
			Locker: mockLocker,
			Gitbak: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()

		if !mockLocker.Released {
			t.Error("Expected Release to be called, but it wasn't")
		}
	})

	// Test 4: Locker with error and Logger
	t.Run("Locker with error and Logger", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "gitbak-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp directory: %v", err)
			}
		}()

		logFile := filepath.Join(tempDir, "test.log")

		mockLocker := &MockLocker{ReleaseErr: fmt.Errorf("test error")}
		testLogger := logger.New(true, logFile, true)

		app := NewApp(AppOptions{
			Config: config.New(),
			Locker: mockLocker,
			Logger: testLogger,
		})

		app.CleanupOnSignal()

		if !mockLocker.Released {
			t.Error("Expected Release to be called, but it wasn't")
		}

		logContent, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !bytes.Contains(logContent, []byte("Failed to release lock during cleanup: test error")) {
			t.Errorf("Expected error to be logged, but log content was: %s", logContent)
		}
	})

	// Test 5: Locker with error but no Logger (stderr output)
	t.Run("Locker with error but no Logger", func(t *testing.T) {
		mockLocker := &MockLocker{ReleaseErr: fmt.Errorf("test error")}

		var buf bytes.Buffer

		app := NewApp(AppOptions{
			Config: config.New(),
			Locker: mockLocker,
			Logger: nil,
			Stderr: &buf,
		})

		app.CleanupOnSignal()

		if !mockLocker.Released {
			t.Error("Expected Release to be called, but it wasn't")
		}

		if !bytes.Contains(buf.Bytes(), []byte("Failed to release lock during cleanup: test error")) {
			t.Errorf("Expected error to be printed to stderr, but output was: %s", buf.String())
		}
	})

	// Test 6: All components present
	t.Run("All components present", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "gitbak-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp directory: %v", err)
			}
		}()

		mockGitbak := &MockGitbaker{}
		mockLocker := &MockLocker{ReleaseErr: nil}
		testLogger := logger.New(true, filepath.Join(tempDir, "test.log"), true)

		app := NewApp(AppOptions{
			Config: config.New(),
			Gitbak: mockGitbak,
			Locker: mockLocker,
			Logger: testLogger,
		})

		app.CleanupOnSignal()

		if !mockGitbak.SummaryCalled {
			t.Error("Expected PrintSummary to be called, but it wasn't")
		}
		if !mockLocker.Released {
			t.Error("Expected Release to be called, but it wasn't")
		}
	})
}

// Integration tests that require a real file system are skipped by default
func TestAppInitialize(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	tmpDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = tmpDir
	app.Config.LogFile = ""

	// Initialize should fail if not a git repo
	err = app.Initialize()
	if err != nil {
		// This is expected since it's not a git repo
		if app.Logger == nil {
			t.Error("Expected Logger to be initialized even with error")
		}
	}
}
