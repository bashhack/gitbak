package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
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

// TestAppCoreScenarios tests various core app functionality scenarios
func TestAppCoreScenarios(t *testing.T) {
	tests := map[string]struct {
		setupFunc     func(t *testing.T) *App
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, app *App)
	}{
		"NewDefaultApp": {
			setupFunc: func(t *testing.T) *App {
				versionInfo := config.VersionInfo{
					Version: "test",
					Commit:  "test-commit",
					Date:    "test-date",
				}

				app := NewDefaultApp(versionInfo)
				app.exit = func(int) {}

				tmpDir := t.TempDir()
				app.Config.RepoPath = tmpDir

				return app
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App) {
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
			},
		},
		"NewAppWithMinimalOptions": {
			setupFunc: func(t *testing.T) *App {
				// Only provide the required Config parameter
				tmpDir := t.TempDir()
				cfg := config.New()
				cfg.RepoPath = tmpDir

				app := NewApp(AppOptions{
					Config: cfg,
				})

				return app
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App) {
				// Test that all defaults were set
				if app.Stdout == nil {
					t.Error("Expected Stdout to have default value (os.Stdout)")
				}
				if app.Stderr == nil {
					t.Error("Expected Stderr to have default value (os.Stderr)")
				}
				if app.exit == nil {
					t.Error("Expected exit to have default value (os.Exit)")
				}
				if app.execLookPath == nil {
					t.Error("Expected execLookPath to have default value (exec.LookPath)")
				}
				if app.isRepository == nil {
					t.Error("Expected isRepository to have default value (git.IsRepository)")
				}
			},
		},
		"AppShowVersion": {
			setupFunc: func(t *testing.T) *App {
				versionInfo := config.VersionInfo{
					Version: "test",
					Commit:  "abc123",
					Date:    "2023-01-01",
				}

				var stdout bytes.Buffer
				tmpDir := t.TempDir()

				mockExit := &mockExit{}

				cfg := config.New()
				cfg.VersionInfo = versionInfo
				cfg.RepoPath = tmpDir

				app := NewApp(AppOptions{
					Config: cfg,
					Stdout: &stdout,
					Exit:   mockExit.Exit,
				})

				return app
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App) {
				var stdout bytes.Buffer
				app.Stdout = &stdout

				app.ShowVersion()

				expected := "gitbak test (abc123) built on 2023-01-01\n"
				if stdout.String() != expected {
					t.Errorf("Expected output %q, got %q", expected, stdout.String())
				}
			},
		},
		"AppShowLogo": {
			setupFunc: func(t *testing.T) *App {
				var stdout bytes.Buffer
				tmpDir := t.TempDir()

				cfg := config.New()
				cfg.VersionInfo = config.VersionInfo{}
				cfg.RepoPath = tmpDir

				app := NewApp(AppOptions{
					Config: cfg,
					Stdout: &stdout,
				})

				return app
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App) {
				var stdout bytes.Buffer
				app.Stdout = &stdout

				app.ShowLogo()

				output := stdout.String()
				if !bytes.Contains([]byte(output), []byte("Automatic Commit Safety Net")) {
					t.Errorf("Logo output does not contain expected tagline: %s", output)
				}
			},
		},
		"AppCheckRequiredCommandsSuccess": {
			setupFunc: func(t *testing.T) *App {
				mockLookPath := &mockExecLookPath{
					lookupMap: map[string]string{
						"git":    "/usr/bin/git",
						"grep":   "/usr/bin/grep",
						"sed":    "/usr/bin/sed",
						"shasum": "/usr/bin/shasum",
					},
					lookupErr: map[string]error{},
				}

				tmpDir := t.TempDir()
				cfg := config.New()
				cfg.RepoPath = tmpDir

				app := NewApp(AppOptions{
					Config:       cfg,
					ExecLookPath: mockLookPath.LookPath,
				})

				return app
			},
			expectError: false,
			validateFunc: func(t *testing.T, app *App) {
				err := app.checkRequiredCommands()
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			},
		},
		"AppCheckRequiredCommandsFailure": {
			setupFunc: func(t *testing.T) *App {
				mockLookPath := &mockExecLookPath{
					lookupMap: map[string]string{
						"grep":   "/usr/bin/grep",
						"sed":    "/usr/bin/sed",
						"shasum": "/usr/bin/shasum",
					},
					lookupErr: map[string]error{
						"git": fmt.Errorf("command not found"),
					},
				}

				tmpDir := t.TempDir()
				cfg := config.New()
				cfg.RepoPath = tmpDir

				app := NewApp(AppOptions{
					Config:       cfg,
					ExecLookPath: mockLookPath.LookPath,
				})

				return app
			},
			expectError:   true,
			errorContains: "git is not found in PATH",
			validateFunc: func(t *testing.T, app *App) {
				err := app.checkRequiredCommands()
				if err == nil {
					t.Error("Expected error when git is missing, got nil")
				}
			},
		},
		"RepositoryDetectionError": {
			setupFunc: func(t *testing.T) *App {
				app := NewTestApp()
				mockLogger := &MockLogger{}
				app = WithMockLogger(app, mockLogger)

				tmpDir := t.TempDir()
				app.Config.RepoPath = tmpDir

				testErr := gitbakErrors.New("simulated repository detection error")
				app = WithIsRepository(app, func(path string) (bool, error) {
					return false, testErr
				})

				return app
			},
			expectError: true,
			validateFunc: func(t *testing.T, app *App) {
				mockLogger := app.Logger.(*MockLogger)

				ctx := context.Background()
				err := app.Run(ctx)

				if err == nil {
					t.Error("Expected Run to return an error, got nil")
				}

				if !gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
					t.Errorf("Expected error to be wrapped with ErrGitOperationFailed, got: %v", err)
				}

				if !mockLogger.WarningCalled {
					t.Error("Expected warning to be logged")
				}
			},
		},
		"NonRepositoryPath": {
			setupFunc: func(t *testing.T) *App {
				app := NewTestApp()
				mockLogger := &MockLogger{}
				app = WithMockLogger(app, mockLogger)

				tmpDir := t.TempDir()
				app.Config.RepoPath = tmpDir

				app = WithIsRepository(app, func(path string) (bool, error) {
					return false, nil // Path is not a repository, but no error
				})

				return app
			},
			expectError: true,
			validateFunc: func(t *testing.T, app *App) {
				ctx := context.Background()
				err := app.Run(ctx)

				if err == nil {
					t.Error("Expected Run to return an error for non-repository path")
				}

				if !gitbakErrors.Is(err, gitbakErrors.ErrNotGitRepository) {
					t.Errorf("Expected error %v, got %v", gitbakErrors.ErrNotGitRepository, err)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			app := test.setupFunc(t)

			if app.isRepository == nil {
				app = WithSimpleIsRepository(app, func(path string) bool {
					return true
				})
			}

			if app.Config.RepoPath != "" {
				if _, err := os.Stat(app.Config.RepoPath); os.IsNotExist(err) {
					newTmpDir := t.TempDir()
					t.Logf("Warning: Test repo path %s doesn't exist, using %s instead",
						app.Config.RepoPath, newTmpDir)
					app.Config.RepoPath = newTmpDir
				}
			} else {
				app.Config.RepoPath = t.TempDir()
			}

			if test.validateFunc != nil {
				test.validateFunc(t, app)
			}
		})
	}
}

// TestRunWithLogoFlag tests running the app with the logo flag
func TestRunWithLogoFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	tmpDir := t.TempDir()

	dummyFile := filepath.Join(tmpDir, "fake.txt")
	if err := os.WriteFile(dummyFile, []byte("fake content"), 0644); err != nil {
		t.Logf("Warning: Failed to create fake file: %v", err)
	}

	cfg := config.New()
	cfg.ShowLogo = true
	cfg.RepoPath = tmpDir

	app := NewApp(AppOptions{
		Config: cfg,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	app = WithSimpleIsRepository(app, func(path string) bool {
		return true
	})

	if err := app.Initialize(); err != nil {
		t.Logf("Warning: Failed to initialize app: %v", err)
	}

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
	t.Run("All components nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.New()
		cfg.RepoPath = tmpDir

		app := NewApp(AppOptions{
			Config: cfg,
			Locker: nil,
			Gitbak: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()
		// No assertions - we're just making sure it doesn't panic
	})

	t.Run("Only gitbak present", func(t *testing.T) {
		mockGitbak := &MockGitbaker{}
		tmpDir := t.TempDir()

		cfg := config.New()
		cfg.RepoPath = tmpDir

		app := NewApp(AppOptions{
			Config: cfg,
			Gitbak: mockGitbak,
			Locker: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()

		if !mockGitbak.SummaryCalled {
			t.Error("Expected PrintSummary to be called, but it wasn't")
		}
	})

	t.Run("Locker with success", func(t *testing.T) {
		mockLocker := &MockLocker{ReleaseErr: nil}
		tmpDir := t.TempDir()

		cfg := config.New()
		cfg.RepoPath = tmpDir

		app := NewApp(AppOptions{
			Config: cfg,
			Locker: mockLocker,
			Gitbak: nil,
			Logger: nil,
		})

		app.CleanupOnSignal()

		if !mockLocker.Released {
			t.Error("Expected Release to be called, but it wasn't")
		}
	})

	t.Run("Locker with error and Logger", func(t *testing.T) {
		tempDir := t.TempDir()
		logFile := filepath.Join(tempDir, "test.log")

		mockLocker := &MockLocker{ReleaseErr: fmt.Errorf("test error")}
		testLogger := logger.New(true, logFile, true)

		cfg := config.New()
		cfg.RepoPath = tempDir

		app := NewApp(AppOptions{
			Config: cfg,
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

	t.Run("Locker with error but no Logger", func(t *testing.T) {
		mockLocker := &MockLocker{ReleaseErr: fmt.Errorf("test error")}
		tmpDir := t.TempDir()

		var buf bytes.Buffer

		cfg := config.New()
		cfg.RepoPath = tmpDir

		app := NewApp(AppOptions{
			Config: cfg,
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

	t.Run("All components present", func(t *testing.T) {
		tempDir := t.TempDir()

		mockGitbak := &MockGitbaker{}
		mockLocker := &MockLocker{ReleaseErr: nil}
		testLogger := logger.New(true, filepath.Join(tempDir, "test.log"), true)

		cfg := config.New()
		cfg.RepoPath = tempDir

		app := NewApp(AppOptions{
			Config: cfg,
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
	app.exit = func(int) {}
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
