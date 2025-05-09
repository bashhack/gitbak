package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
)

// MockGitbaker implements a mock of the Gitbaker interface for testing.
// It tracks method calls and their parameters for verification in tests.
type MockGitbaker struct {
	SummaryCalled bool
	RunCalled     bool
	RunErr        error
	LastContext   context.Context
}

func (m *MockGitbaker) PrintSummary() {
	m.SummaryCalled = true
}

func (m *MockGitbaker) Run(ctx context.Context) error {
	m.RunCalled = true
	m.LastContext = ctx
	return m.RunErr
}

// MockLocker implements the Locker interface for testing.
// It tracks lock acquisition and release operations, allowing tests to verify
// the correct locking behavior and simulate lock acquisition failures.
type MockLocker struct {
	AcquireErr    error // Error to return from Acquire()
	ReleaseErr    error // Error to return from Release()
	AcquireCalled bool  // Set to true when Acquire() is called
	ReleaseCalled bool  // Set to true when Release() is called
	Released      bool  // Represents the lock state after Release() is called
}

func (m *MockLocker) Acquire() error {
	m.AcquireCalled = true
	return m.AcquireErr
}

func (m *MockLocker) Release() error {
	m.ReleaseCalled = true
	m.Released = true
	return m.ReleaseErr
}

// MockLogger implements the Logger interface for testing.
// It captures all logging calls and the most recent message for each type of log,
// allowing tests to verify that appropriate logging occurred.
type MockLogger struct {
	InfoCalled          bool   // Set to true when Info() is called
	InfoToUserCalled    bool   // Set to true when InfoToUser() is called
	WarningCalled       bool   // Set to true when Warning() is called
	WarningToUserCalled bool   // Set to true when WarningToUser() is called
	ErrorCalled         bool   // Set to true when Error() is called
	SuccessCalled       bool   // Set to true when Success() is called
	StatusCalled        bool   // Set to true when StatusMessage() is called
	LastMessage         string // Contains the most recent message passed to any log method
}

// Standard logging methods

// Info logs an info message
func (m *MockLogger) Info(format string, args ...interface{}) {
	m.InfoCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Warning logs an warning message
func (m *MockLogger) Warning(format string, args ...interface{}) {
	m.WarningCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Error logs an error message
func (m *MockLogger) Error(format string, args ...interface{}) {
	m.ErrorCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Enhanced user-facing logging methods

// InfoToUser logs an info message to the user
func (m *MockLogger) InfoToUser(format string, args ...interface{}) {
	m.InfoToUserCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// WarningToUser logs a warning message to the user
func (m *MockLogger) WarningToUser(format string, args ...interface{}) {
	m.WarningToUserCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Success logs a success message
func (m *MockLogger) Success(format string, args ...interface{}) {
	m.SuccessCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// StatusMessage logs a status message
func (m *MockLogger) StatusMessage(format string, args ...interface{}) {
	m.StatusCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// withTempWorkDir creates a temporary directory, changes to it, and
// executes the provided function. It handles cleanup and restoring
// the original working directory.
func withTempWorkDir(t *testing.T, fn func(dir string)) {
	t.Helper()

	// Get the current working directory, but don't fail the test if it fails...
	// this should give some resilience over issues in CI environments where
	// the original directory might be deleted (as appears to have just happened
	// to me in GH Actions...)
	orig, err := os.Getwd()
	if err != nil {
		t.Logf("Warning: Failed to get current working directory: %v", err)
		// Continue with temp directory creation even if we can't get the original directory
	}

	tmp := t.TempDir()

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Only try to restore the original directory if we successfully got it
	if orig != "" {
		defer func() {
			if err := os.Chdir(orig); err != nil {
				t.Logf("Failed to restore working directory: %v", err)
			}
		}()
	}

	fn(tmp)
}

// withGitRepo creates a temporary directory with a Git repository initialized in it,
// changes to that directory, and executes the provided function. It handles cleanup
// and restoring the original working directory.
func withGitRepo(t *testing.T, fn func(gitRepoPath string)) {
	t.Helper()

	withTempWorkDir(t, func(dir string) {
		cmd := exec.Command("git", "init", dir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to initialize git repo: %v", err)
		}

		cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@example.com")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git user email: %v", err)
		}

		cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test User")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to configure git user name: %v", err)
		}

		initialFile := filepath.Join(dir, "initial.txt")
		if err := os.WriteFile(initialFile, []byte("Initial content"), 0644); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		cmd = exec.Command("git", "-C", dir, "add", "initial.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to add initial file: %v", err)
		}

		cmd = exec.Command("git", "-C", dir, "commit", "-m", "Initial commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create initial commit: %v", err)
		}

		// Run the function with the git repository path
		fn(dir)
	})
}

// Testing helper functions
// The following functions facilitate testing by allowing injection of mocks and
// customization of the App's behavior. They follow a builder pattern where each
// function returns the modified App, allowing them to be chained together.

// NewTestApp creates a new App with default test settings.
// It initializes a standard App but replaces the exit function with a no-op
// to prevent tests from terminating the process. This provides a base App
// that can be further customized with the With* functions.
func NewTestApp() *App {
	app := NewDefaultApp(config.VersionInfo{})

	app.exit = func(int) {}

	return app
}

// WithMockLocker adds a mock locker to the app.
// This allows tests to control locking behavior and verify that lock
// acquisition and release happen at the correct times. It accepts a
// MockLocker instance that can be pre-configured for specific test scenarios.
func WithMockLocker(app *App, mockLocker *MockLocker) *App {
	app.Locker = mockLocker
	return app
}

// WithMockLogger adds a mock logger to the app.
// It accepts a MockLogger instance for consistency with other mock injection functions.
func WithMockLogger(app *App, mockLogger *MockLogger) *App {
	app.Logger = mockLogger
	return app
}

// WithIsRepository replaces the app's isRepository function with a custom implementation.
// This allows tests to simulate different repository detection scenarios without
// requiring actual Git repositories. The provided function should accept a path string
// and return a boolean indicating if it's a repository, along with any error.
func WithIsRepository(app *App, fn func(string) (bool, error)) *App {
	app.isRepository = fn
	return app
}

// WithSimpleIsRepository replaces the isRepository function with a version that never returns errors.
// This is a convenience method for tests that only need to control the boolean result
// of repository detection without simulating error conditions. The provided function
// simply returns true for valid repositories and false otherwise.
func WithSimpleIsRepository(app *App, simpleFn func(string) bool) *App {
	app.isRepository = func(path string) (bool, error) {
		return simpleFn(path), nil
	}
	return app
}

// WithExecLookPath replaces the app's execLookPath function used for executable detection.
// This allows tests to simulate scenarios where Git or other required programs
// are either missing or located in specific paths. The provided function should
// accept a command name and return its path or an error if not found.
func WithExecLookPath(app *App, fn func(string) (string, error)) *App {
	app.execLookPath = fn
	return app
}

// WithExit replaces the app's exit function to prevent tests from terminating the process.
// The provided function will be called instead of os.Exit, allowing tests to
// verify that the app exits with the expected status code under various conditions.
func WithExit(app *App, fn func(int)) *App {
	app.exit = fn
	return app
}
