package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/lock"
	"github.com/bashhack/gitbak/internal/logger"
)

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

// TestAppInitializeScenarios tests various scenarios for the Initialize method
func TestAppInitializeScenarios(t *testing.T) {
	tests := map[string]struct {
		setupFunc     func(t *testing.T) (*App, string)
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, app *App, repoPath string)
	}{
		"ValidRepo": {
			setupFunc: func(t *testing.T) (*App, string) {
				var repoPath string
				withGitRepo(t, func(path string) {
					repoPath = path
				})

				logDir := t.TempDir()

				cfg := config.New()
				cfg.RepoPath = repoPath
				cfg.LogFile = filepath.Join(logDir, "gitbak.log")

				var stdout, stderr bytes.Buffer

				app := NewApp(AppOptions{
					Config: cfg,
					Stdout: &stdout,
					Stderr: &stderr,
				})

				return app, repoPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				if app.Locker == nil {
					t.Error("Expected Locker to be initialized")
				}
				if app.Logger == nil {
					t.Error("Expected Logger to be initialized")
				}
				// Note: gitbak is not initialized in the Initialize() method,
				// it's initialized in the Run() method instead
			},
		},
		"NonGitRepo": {
			setupFunc: func(t *testing.T) (*App, string) {
				nonGitDir := t.TempDir()

				if err := os.WriteFile(filepath.Join(nonGitDir, "test.txt"), []byte("Test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				logDir := t.TempDir()

				cfg := config.New()
				cfg.RepoPath = nonGitDir
				cfg.LogFile = filepath.Join(logDir, "gitbak.log")

				var stdout, stderr bytes.Buffer

				app := NewApp(AppOptions{
					Config: cfg,
					Stdout: &stdout,
					Stderr: &stderr,
				})

				return app, nonGitDir
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				if app.Logger == nil {
					t.Error("Expected Logger to be initialized for non-git directories")
				}

				// The Initialize method doesn't check if it's a git repository,
				// So the Locker would still be initialized
				if app.Locker == nil {
					t.Error("Expected Locker to be initialized")
				}

				// No error messages are expected at this point because
				// Initialize() doesn't check if it's a git repository
			},
		},
		"EmptyDir": {
			setupFunc: func(t *testing.T) (*App, string) {
				emptyDir := t.TempDir()

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.RepoPath = emptyDir

				logDir := t.TempDir()

				app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

				var stdout, stderr bytes.Buffer
				app.Stdout = &stdout
				app.Stderr = &stderr

				return app, emptyDir
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				if app.Logger == nil {
					t.Error("Expected Logger to be initialized for empty directories")
				}

				// No error message is expected because Initialize() doesn't check
				// if the directory is a git repository
			},
		},
		"InvalidLogPath": {
			setupFunc: func(t *testing.T) (*App, string) {
				var repoPath string
				withGitRepo(t, func(path string) {
					repoPath = path
				})

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.RepoPath = repoPath

				app.Config.LogFile = "/non/existent/directory/gitbak.log"

				var stdout, stderr bytes.Buffer
				app.Stdout = &stdout
				app.Stderr = &stderr

				return app, repoPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				if app.Logger == nil {
					t.Error("Expected Logger to be initialized with fallback options")
				}

				if app.Locker == nil {
					t.Error("Expected Locker to be initialized")
				}

				// Note: gitbak is initialized in Run(), not in Initialize()

				// The app will create the logger with fallback options, but it doesn't print warnings
				// about log file failures - that would be handled by the logger itself
				// The logger object is still created even with an invalid log path
			},
		},
		"CustomInterval": {
			setupFunc: func(t *testing.T) (*App, string) {
				var repoPath string
				withGitRepo(t, func(path string) {
					repoPath = path
				})

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.RepoPath = repoPath
				app.Config.IntervalMinutes = 10 // Custom interval

				logDir := t.TempDir()

				app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

				return app, repoPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				// Verify essential components are initialized
				if app.Locker == nil {
					t.Error("Expected Locker to be initialized")
				}
				if app.Logger == nil {
					t.Error("Expected Logger to be initialized")
				}

				// Note: gitbak is initialized in Run(), not in Initialize()

				// Verify that the configuration has the correct interval setting
				if app.Config.IntervalMinutes != 10 {
					t.Errorf("Expected Config.IntervalMinutes to be 10, got %.1f", app.Config.IntervalMinutes)
				}
			},
		},
		"NilLogger": {
			setupFunc: func(t *testing.T) (*App, string) {
				var repoPath string
				withGitRepo(t, func(path string) {
					repoPath = path
				})

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.RepoPath = repoPath

				logDir := t.TempDir()

				app.Config.LogFile = filepath.Join(logDir, "gitbak.log")

				app.Logger = nil

				return app, repoPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				if app.Logger == nil {
					t.Error("Expected Logger to be created when nil")
				}

				// Verify the log file path was created if debug is enabled
				if app.Config.Debug {
					logPath := filepath.Join(filepath.Dir(app.Config.LogFile), "gitbak.log")
					fileInfo, err := os.Stat(logPath)
					if err != nil {
						t.Errorf("Expected log file to be created at %s, error: %v", logPath, err)
					} else if !fileInfo.Mode().IsRegular() {
						t.Errorf("Expected log file to be a regular file")
					}
				}
			},
		},
		"ExistingComponents": {
			setupFunc: func(t *testing.T) (*App, string) {
				var repoPath string
				withGitRepo(t, func(path string) {
					repoPath = path
				})

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
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

				return app, repoPath
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				customLogger := app.Logger
				mockGitbak := app.Gitbak.(*MockGitbaker)

				if err := app.Initialize(); err != nil {
					t.Fatalf("Failed to initialize with existing components: %v", err)
				}

				if app.Logger != customLogger {
					t.Error("Expected Logger to remain unchanged")
				}

				if app.Gitbak != mockGitbak {
					t.Error("Expected Gitbak to remain unchanged")
				}

				// Also verify that the components are still functional,
				// for example, let's check if the Locker still works correctly
				releaseErr := app.Locker.Release()
				if releaseErr != nil {
					t.Errorf("Expected original locker to be functional, got release error: %v", releaseErr)
				}
			},
		},
		"ConfigFinalizeError": {
			setupFunc: func(t *testing.T) (*App, string) {
				app := &App{
					Config: &config.Config{
						// Set invalid config that will cause Finalize to fail
						IntervalMinutes: -1, // Negative interval should cause error
					},
				}
				return app, ""
			},
			expectError: true,
			validateFunc: func(t *testing.T, app *App, repoPath string) {
				// No additional validation needed
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			app, repoPath := test.setupFunc(t)
			err := app.Initialize()

			if test.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !test.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if err != nil && test.errorContains != "" {
				if !strings.Contains(err.Error(), test.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", test.errorContains, err)
				}
			}

			if test.validateFunc != nil {
				test.validateFunc(t, app, repoPath)
			}
		})
	}
}

// TestReleaseErrorHandling tests error handling in the CleanupOnSignal method
func TestReleaseErrorHandling(t *testing.T) {
	t.Parallel()
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
		expectedErrMsg := "mock release error"
		if !strings.Contains(stderr, expectedErrMsg) {
			t.Errorf("Expected error message to contain '%s', got: %s", expectedErrMsg, stderr)
		} else {
			t.Logf("Got expected error output: %s", stderr)
		}
	}
}
