package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/logger"
)

// Mock interfaces for testing

// MockRunLocker for testing lock acquisition failures
type MockRunLocker struct {
	AcquireErr    error
	ReleaseErr    error
	AcquireCalled bool
	ReleaseCalled bool
}

func (m *MockRunLocker) Acquire() error {
	m.AcquireCalled = true
	return m.AcquireErr
}

func (m *MockRunLocker) Release() error {
	m.ReleaseCalled = true
	return m.ReleaseErr
}

// MockRunGitbaker for testing Gitbak.Run failures
type MockRunGitbaker struct {
	RunErr        error
	SummaryCalled bool
	RunCalled     bool
}

func (m *MockRunGitbaker) PrintSummary() {
	m.SummaryCalled = true
}

func (m *MockRunGitbaker) Run(ctx context.Context) error {
	m.RunCalled = true
	return m.RunErr
}

// MockRunExecLookPath for testing command existence checks
type MockRunExecLookPath struct {
	PathErr error
}

func (m *MockRunExecLookPath) LookPath(name string) (string, error) {
	if name == "git" && m.PathErr != nil {
		return "", m.PathErr
	}
	return "/usr/bin/" + name, nil
}

// MockRunLogger is a simple mock implementing the Logger interface
type MockRunLogger struct{}

// Basic logging methods
func (m *MockRunLogger) Info(format string, args ...interface{})    {}
func (m *MockRunLogger) Warning(format string, args ...interface{}) {}
func (m *MockRunLogger) Error(format string, args ...interface{})   {}

// Enhanced logging methods
func (m *MockRunLogger) InfoToUser(format string, args ...interface{})    {}
func (m *MockRunLogger) WarningToUser(format string, args ...interface{}) {}
func (m *MockRunLogger) Success(format string, args ...interface{})       {}
func (m *MockRunLogger) StatusMessage(format string, args ...interface{}) {}

// MockRunIsRepository is used to mock the git.IsRepository function during tests
type MockRunIsRepository struct {
	ReturnValue bool
}

func (m *MockRunIsRepository) IsRepository(path string) bool {
	return m.ReturnValue
}

// TestRunWithVersionFlag tests the version flag case
func TestRunWithVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := NewDefaultApp(config.VersionInfo{
		Version: "test-version",
		Commit:  "test-commit",
		Date:    "test-date",
	})
	app.Config.Version = true
	app.Stdout = &stdout
	app.Stderr = &stderr

	ctx := context.Background()
	err := app.Run(ctx)
	if err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("gitbak test-version")) {
		t.Errorf("Expected output to contain version info, got: %s", output)
	}
}

// TestRunWithLogoFlagWithMock tests the logo flag case with mocked functions
func TestRunWithLogoFlagWithMock(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.ShowLogo = true
	app.Stdout = &stdout
	app.Stderr = &stderr

	ctx := context.Background()
	err := app.Run(ctx)
	if err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	output := stdout.String()
	if !bytes.Contains([]byte(output), []byte("@@@@@@")) {
		t.Errorf("Expected output to contain ASCII art logo with @@ characters")
	}
}

// TestRunWithMissingGitCommand tests the case where git is not found
func TestRunWithMissingGitCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := NewDefaultApp(config.VersionInfo{})
	app.execLookPath = (&MockRunExecLookPath{
		PathErr: errors.New("git not found"),
	}).LookPath
	app.Stdout = &stdout
	app.Stderr = &stderr

	ctx := context.Background()
	err := app.Run(ctx)
	if err == nil {
		t.Error("Expected Run to return an error when git is not found")
	}

	errOutput := stderr.String()
	if !bytes.Contains([]byte(errOutput), []byte("Error:")) {
		t.Errorf("Expected error output to mention an error, got: %s", errOutput)
	}
}

// TestRunWithNoLockAcquisitionFailure tests the case where lock acquisition succeeds
func TestRunWithLockAcquisitionFailure(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	var stdout, stderr bytes.Buffer

	mockLocker := &MockRunLocker{
		AcquireErr: errors.New("lock acquisition failed"),
	}

	app := NewDefaultApp(config.VersionInfo{})
	app.Stdout = &stdout
	app.Stderr = &stderr
	app.Locker = mockLocker
	app.Logger = logger.New(false, "", true)

	app.isRepository = func(path string) bool {
		return true
	}

	app.execLookPath = (&MockRunExecLookPath{}).LookPath

	ctx := context.Background()
	err = app.Run(ctx)
	if err == nil {
		t.Error("Expected Run to return an error when lock acquisition fails")
	}

	if !mockLocker.AcquireCalled {
		t.Error("Expected locker.Acquire to be called")
	}

	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("lock")) {
		t.Errorf("Expected error to mention lock acquisition, got: %v", err)
	}
}

// TestRunWithGitbakRunFailure tests the case where Gitbak.Run fails
func TestRunWithGitbakRunFailure(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	var stdout, stderr bytes.Buffer

	mockLocker := &MockRunLocker{}

	mockGitbaker := &MockRunGitbaker{
		RunErr: errors.New("gitbak run failed"),
	}

	app := NewDefaultApp(config.VersionInfo{})
	app.Stdout = &stdout
	app.Stderr = &stderr
	app.Locker = mockLocker
	app.Logger = logger.New(false, "", true)
	app.Gitbak = mockGitbaker

	app.isRepository = func(path string) bool {
		return true
	}

	app.execLookPath = (&MockRunExecLookPath{}).LookPath

	ctx := context.Background()
	err = app.Run(ctx)
	if err == nil {
		t.Error("Expected Run to return an error when Gitbak.Run fails")
	}

	if !mockGitbaker.RunCalled {
		t.Error("Expected Gitbak.Run to be called")
	}

	if err != nil && err.Error() != "gitbak run failed" {
		t.Errorf("Expected error to be 'gitbak run failed', got: %v", err)
	}

	if !mockLocker.ReleaseCalled {
		t.Error("Expected locker.Release to be called during cleanup")
	}
}

// TestRunWithLockReleaseFailure tests the cleanup when lock release fails
func TestRunWithLockReleaseFailure(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	var stdout, stderr bytes.Buffer

	mockLocker := &MockRunLocker{
		ReleaseErr: errors.New("lock release failed"),
	}

	mockGitbaker := &MockRunGitbaker{
		RunErr: errors.New("gitbak run failed"),
	}

	app := NewDefaultApp(config.VersionInfo{})
	app.Stdout = &stdout
	app.Stderr = &stderr
	app.Locker = mockLocker
	app.Logger = logger.New(false, "", true)
	app.Gitbak = mockGitbaker

	app.isRepository = func(path string) bool {
		return true
	}

	app.execLookPath = (&MockRunExecLookPath{}).LookPath

	ctx := context.Background()
	err = app.Run(ctx)
	if err == nil {
		t.Error("Expected Run to return an error when Gitbak.Run fails")
	}

	if !mockLocker.ReleaseCalled {
		t.Error("Expected locker.Release to be called during cleanup")
	}
}

// TestRunWithLoggerAssertionFailure tests the case where Logger type assertion fails
func TestRunWithLoggerAssertionFailure(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	var stdout, stderr bytes.Buffer

	mockLocker := &MockRunLocker{}

	mockGitbaker := &MockRunGitbaker{}

	app := NewDefaultApp(config.VersionInfo{})
	app.Stdout = &stdout
	app.Stderr = &stderr
	app.Locker = mockLocker
	app.Logger = &MockRunLogger{}
	app.Gitbak = mockGitbaker

	app.isRepository = func(path string) bool {
		return true
	}

	app.execLookPath = (&MockRunExecLookPath{}).LookPath

	ctx := context.Background()
	err = app.Run(ctx)
	if err != nil {
		// We're not expecting an error here, we're testing that the fallback logger works
		t.Errorf("Run returned unexpected error: %v", err)
	}

	// Verify Gitbak was accessed (which means the fallback logger works)
	if !mockGitbaker.RunCalled {
		t.Error("Expected Gitbak.Run to be called")
	}
}

// TestRunComplete tests a successful run
func TestRunComplete(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: Failed to remove temporary directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	var stdout, stderr bytes.Buffer

	mockGitbaker := &MockRunGitbaker{}

	mockLocker := &MockRunLocker{}

	app := &App{
		Config: &config.Config{
			RepoPath:        tempDir,
			IntervalMinutes: 5,
			BranchName:      "test-branch",
			CommitPrefix:    "[test] ",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		Logger:       logger.New(false, "", true),
		Locker:       mockLocker,
		execLookPath: (&MockRunExecLookPath{}).LookPath,
		Stdout:       &stdout,
		Stderr:       &stderr,
		exit:         func(int) {},
		Gitbak:       mockGitbaker,
		isRepository: func(path string) bool {
			return true
		},
	}

	ctx := context.Background()
	err = app.Run(ctx)
	if err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	if !mockLocker.AcquireCalled {
		t.Error("Expected locker.Acquire to be called")
	}
	if !mockGitbaker.RunCalled {
		t.Error("Expected Gitbak.Run to be called")
	}

	// Verify the lock is released (even on success)
	if !mockLocker.ReleaseCalled {
		t.Error("Expected locker.Release to be called during cleanup")
	}
}

// TestRunWithGitbakProvidedExternally tests providing a mock Gitbaker
func TestRunWithGitbakProvidedExternally(t *testing.T) {
	mockGitbaker := &MockGitbaker{}

	app := NewTestApp()
	app.Config.RepoPath = "/test/repo"
	app.Config.IntervalMinutes = 5

	app.Stdout = &bytes.Buffer{}
	app.Stderr = &bytes.Buffer{}

	app = WithMockLogger(app, &MockLogger{})
	app = WithMockLocker(app, &MockLocker{})
	app = WithIsRepository(app, func(path string) bool {
		return true
	})
	app = WithExecLookPath(app, func(string) (string, error) {
		return "/usr/bin/git", nil
	})

	app.Gitbak = mockGitbaker

	ctx := context.Background()
	err := app.Run(ctx)
	if err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	if !mockGitbaker.RunCalled {
		t.Error("Expected Gitbak.Run to be called")
	}
}

// Test to cover the lock acquisition error path
func TestRunWithLockAcquisitionError(t *testing.T) {
	mockLocker := &MockLocker{
		AcquireErr: errors.New("test acquisition error"),
	}

	app := NewTestApp()
	app.Config.RepoPath = "/test/repo"

	app = WithMockLogger(app, &MockLogger{})
	app = WithMockLocker(app, mockLocker)
	app = WithExecLookPath(app, func(string) (string, error) {
		return "/usr/bin/git", nil
	})
	app = WithIsRepository(app, func(path string) bool {
		return true
	})

	app.Stdout = &bytes.Buffer{}
	app.Stderr = &bytes.Buffer{}

	ctx := context.Background()
	err := app.Run(ctx)

	if err == nil {
		t.Error("Expected an error from lock acquisition failure, got nil")
	}

	if !mockLocker.AcquireCalled {
		t.Error("Expected Locker.Acquire to be called")
	}
}

// Test to cover the deferred lock release path
func TestRunWithLockAcquisitionErrorDeferred(t *testing.T) {
	mockLocker := &MockLocker{
		ReleaseErr: errors.New("test release error"),
	}

	mockGitbaker := &MockGitbaker{
		RunErr: errors.New("gitbak run error"),
	}

	app := NewTestApp()
	app.Config.RepoPath = "/test/repo"

	app = WithMockLogger(app, &MockLogger{})
	app = WithMockLocker(app, mockLocker)
	app = WithExecLookPath(app, func(string) (string, error) {
		return "/usr/bin/git", nil
	})
	app = WithIsRepository(app, func(path string) bool {
		return true
	})

	app.Stdout = &bytes.Buffer{}
	app.Stderr = &bytes.Buffer{}

	app.Gitbak = mockGitbaker

	ctx := context.Background()
	_ = app.Run(ctx)

	if !mockLocker.ReleaseCalled {
		t.Error("Expected Locker.Release to be called by deferred function")
	}
}

// TestRunWithNormalGitbakCreation tests the normal Gitbak creation path
func TestRunWithNormalGitbakCreation(t *testing.T) {
	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = "/test/repo"
	app.Config.IntervalMinutes = 5
	app.Config.BranchName = "test-branch"
	app.Config.CommitPrefix = "[test] "
	app.Config.Verbose = true

	app.Logger = logger.New(false, "", true)
	app.Locker = &MockRunLocker{}
	app.execLookPath = (&MockRunExecLookPath{}).LookPath
	app.Stdout = &bytes.Buffer{}
	app.Stderr = &bytes.Buffer{}

	app.isRepository = func(path string) bool {
		return true
	}

	// Create a mock for Gitbak to avoid real git operations
	// This test is just to cover the creation code path, not the actual run
	mockGitbaker := &MockRunGitbaker{}
	app.Gitbak = mockGitbaker

	ctx := context.Background()
	err := app.Run(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify Gitbak.Run was called
	if !mockGitbaker.RunCalled {
		t.Error("Expected Gitbak.Run to be called")
	}
}

// TestRunWithNotAGitRepoError tests the error case when a path is not a git repository
func TestRunWithNotAGitRepoError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := NewTestApp()
	app.Config.RepoPath = "/test/repo"
	app.Stdout = &stdout
	app.Stderr = &stderr

	app = WithExecLookPath(app, func(file string) (string, error) {
		return "/usr/bin/git", nil // Make sure checkRequiredCommands passes
	})

	app = WithIsRepository(app, func(path string) bool {
		return false // Path is not a git repo
	})

	ctx := context.Background()
	err := app.Run(ctx)

	if err == nil {
		t.Error("Expected an error when path is not a git repo, got nil")
	}

	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("git repository")) {
		t.Errorf("Expected error to mention git repository, got: %v", err)
	}
}
