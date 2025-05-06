package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/lock"
	"github.com/bashhack/gitbak/internal/logger"
)

// Custom error type for testing error handling
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// setupTestRepo creates a minimal Git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gitbak-app-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init", tempDir)
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("Initial content"), 0644)
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to create initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "add", "initial.txt")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to add initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp dir: %v", cleanErr)
		}
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

// cleanupTestRepo removes the test repository
func cleanupTestRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Logf("Failed to remove test repo: %v", err)
	}
}

// TestAppInitializeWithValidRepo tests initializing the app with a valid git repository
func TestAppInitializeWithValidRepo(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	logDir, err := os.MkdirTemp("", "gitbak-app-logs-*")
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			t.Logf("Failed to remove log directory: %v", err)
		}
	}()

	cfg := config.New()
	cfg.RepoPath = repoPath
	cfg.LogFile = filepath.Join(logDir, "gitbak.log")

	var stdout, stderr bytes.Buffer

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with valid repo: %v", err)
	}

	if app.Locker == nil {
		t.Error("Expected Locker to be initialized")
	}
	if app.Logger == nil {
		t.Error("Expected Logger to be initialized")
	}

	// Note: Gitbak is not initialized in the Initialize() method
	// It's initialized in the Run() method instead
}

// TestAppInitializeWithNonGitRepo tests initializing the app with a non-git directory
func TestAppInitializeWithNonGitRepo(t *testing.T) {
	nonGitDir, err := os.MkdirTemp("", "gitbak-non-git-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(nonGitDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	err = os.WriteFile(filepath.Join(nonGitDir, "test.txt"), []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logDir, err := os.MkdirTemp("", "gitbak-app-logs-*")
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			t.Logf("Failed to remove log directory: %v", err)
		}
	}()

	cfg := config.New()
	cfg.RepoPath = nonGitDir
	cfg.LogFile = filepath.Join(logDir, "gitbak.log")

	var stdout, stderr bytes.Buffer

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// Note: App.Initialize() doesn't check if the directory is a git repo
	// That check happens in Run() instead, so Initialize() will succeed
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with non-git directory: %v", err)
	}

	if app.Logger == nil {
		t.Error("Expected Logger to be initialized even with error")
	}

	// The Initialize method doesn't check if it's a git repository,
	// So the Locker would still be initialized
	if app.Locker == nil {
		t.Error("Expected Locker to be initialized")
	}

	// No error messages are expected at this point because
	// Initialize() doesn't check if it's a git repository
}

// TestAppInitializeWithEmptyDir tests initializing the app with an empty directory
func TestAppInitializeWithEmptyDir(t *testing.T) {
	emptyDir, err := os.MkdirTemp("", "gitbak-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(emptyDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = emptyDir

	logDir, err := os.MkdirTemp("", "gitbak-app-logs-*")
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			t.Logf("Failed to remove log directory: %v", err)
		}
	}()

	app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

	var stdout, stderr bytes.Buffer
	app.Stdout = &stdout
	app.Stderr = &stderr

	// Note: App.Initialize() doesn't check if the directory is a git repo
	// That check happens in Run() instead, so Initialize() will succeed
	err = app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with empty directory: %v", err)
	}

	if app.Logger == nil {
		t.Error("Expected Logger to be initialized even with error")
	}

	// No error message is expected because Initialize() doesn't check
	// if the directory is a git repository
}

// TestAppInitializeWithInvalidLogPath tests initializing with an invalid log path
func TestAppInitializeWithInvalidLogPath(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = repoPath

	app.Config.LogFile = "/non/existent/directory/gitbak.log"

	var stdout, stderr bytes.Buffer
	app.Stdout = &stdout
	app.Stderr = &stderr

	err := app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with invalid log path: %v", err)
	}

	if app.Logger == nil {
		t.Error("Expected Logger to be initialized with fallback options")
	}

	if app.Locker == nil {
		t.Error("Expected Locker to be initialized")
	}

	// Gitbak is initialized in Run(), not in Initialize()

	// The app will create the logger with fallback options, but it doesn't print warnings
	// about log file failures - that would be handled by the logger itself
	// The logger object is still created even with an invalid log path
}

// TestAppInitializeWithCustomInterval tests initializing with a custom interval
func TestAppInitializeWithCustomInterval(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = repoPath
	app.Config.IntervalMinutes = 10 // Custom interval

	logDir, err := os.MkdirTemp("", "gitbak-app-logs-*")
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			t.Logf("Failed to remove log directory: %v", err)
		}
	}()

	app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with custom interval: %v", err)
	}

	// Verify essential components are initialized
	if app.Locker == nil {
		t.Error("Expected Locker to be initialized")
	}
	if app.Logger == nil {
		t.Error("Expected Logger to be initialized")
	}

	// Note: Gitbak is initialized in Run(), not in Initialize()

	// Verify that the configuration has the correct interval setting
	if app.Config.IntervalMinutes != 10 {
		t.Errorf("Expected Config.IntervalMinutes to be 10, got %d", app.Config.IntervalMinutes)
	}
}

// TestAppInitializeWithNilLogger tests that a logger is created even if none is provided
func TestAppInitializeWithNilLogger(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = repoPath

	logDir, err := os.MkdirTemp("", "gitbak-app-logs-*")
	if err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			t.Logf("Failed to remove log directory: %v", err)
		}
	}()

	app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

	app.Logger = nil

	err = app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with nil logger: %v", err)
	}

	if app.Logger == nil {
		t.Error("Expected Logger to be created when nil")
	}

	// Verify the log file path was created if debug is enabled
	if app.Config.Debug {
		logPath := filepath.Join(logDir, "gitbak.log")
		fileInfo, err := os.Stat(logPath)
		if err != nil {
			t.Errorf("Expected log file to be created at %s, error: %v", logPath, err)
		} else if !fileInfo.Mode().IsRegular() {
			t.Errorf("Expected log file to be a regular file")
		}
	}
}

// TestAppInitializeWithExistingComponents tests that existing components are not replaced
func TestAppInitializeWithExistingComponents(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	app := NewDefaultApp(config.VersionInfo{})
	app.Config.RepoPath = repoPath

	customLogger := logger.New(false, "", false)
	app.Logger = customLogger

	// Create a real Locker since Initialize() sets it
	var locker Locker
	originalLocker, locker_err := lock.New(repoPath)
	if locker_err != nil {
		t.Fatalf("Failed to create original locker: %v", locker_err)
	}
	locker = originalLocker
	app.Locker = locker

	mockGitbak := &MockGitbaker{}
	app.Gitbak = mockGitbak

	err := app.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize with existing components: %v", err)
	}

	if app.Locker != originalLocker {
		t.Error("Expected Locker to remain unchanged")
	}

	if app.Gitbak != mockGitbak {
		t.Error("Expected Gitbak to remain unchanged")
	}

	if app.Logger != customLogger {
		t.Error("Expected Logger to remain unchanged")
	}

	// Also verify that the components are still functional,
	// for example, let's check if the Locker still works correctly
	releaseErr := app.Locker.Release()
	if releaseErr != nil {
		t.Errorf("Expected original locker to be functional, got release error: %v", releaseErr)
	}
}

// TestInitializeErrors tests error paths in the Initialize method
func TestInitializeErrors(t *testing.T) {
	t.Run("Config.Finalize error", func(t *testing.T) {
		app := &App{
			Config: &config.Config{
				// Set invalid config that will cause Finalize to fail
				IntervalMinutes: -1, // Negative interval should cause error
			},
		}

		err := app.Initialize()
		if err == nil {
			t.Error("Expected error from Initialize when Config.Finalize fails, got nil")
		} else {
			t.Logf("Got expected error: %v", err)
		}
	})
}

// TestReleaseErrorHandling tests error handling in the CleanupOnSignal method
func TestReleaseErrorHandling(t *testing.T) {
	app := NewTestApp()

	mockLocker := &MockLocker{ReleaseErr: &customError{"mock release error"}}
	app = WithMockLocker(app, mockLocker)

	var stderrBuf bytes.Buffer
	app.Stderr = &stderrBuf

	app.CleanupOnSignal()

	if !mockLocker.ReleaseCalled {
		t.Error("Expected locker.Release to be called")
	}

	stderr := stderrBuf.String()
	if stderr == "" {
		t.Error("Expected error to be written to stderr")
	} else {
		t.Logf("Got expected error output: %s", stderr)
	}
}
