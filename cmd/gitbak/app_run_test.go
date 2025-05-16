package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/bashhack/gitbak/pkg/config"
	"github.com/bashhack/gitbak/pkg/logger"
)

// TestAppRunScenarios tests various scenarios for the Run method
func TestAppRunScenarios(t *testing.T) {
	tests := map[string]struct {
		setupFunc      func(t *testing.T) (*App, context.Context)
		expectError    bool
		errorContains  string
		validateOutput func(t *testing.T, app *App, stdout, stderr *bytes.Buffer, err error)
		validateState  func(t *testing.T, app *App)
	}{
		"VersionFlag": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				tmpDir := t.TempDir()

				app := NewDefaultApp(config.VersionInfo{
					Version: "test-version",
					Commit:  "test-commit",
					Date:    "test-date",
				})
				app.exit = func(int) {}
				app.Config.Version = true
				app.Config.RepoPath = tmpDir
				app.Stdout = &stdout
				app.Stderr = &stderr

				app.isRepository = func(path string) (bool, error) {
					return true, nil
				}

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateOutput: func(t *testing.T, app *App, stdout, stderr *bytes.Buffer, err error) {
				output := stdout.String()
				if !bytes.Contains([]byte(output), []byte("gitbak test-version")) {
					t.Errorf("Expected output to contain version info, got: %s", output)
				}
			},
		},
		"LogoFlag": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				tmpDir := t.TempDir()

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.ShowLogo = true
				app.Config.RepoPath = tmpDir
				app.Stdout = &stdout
				app.Stderr = &stderr

				app.isRepository = func(path string) (bool, error) {
					return true, nil
				}

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateOutput: func(t *testing.T, app *App, stdout, stderr *bytes.Buffer, err error) {
				output := stdout.String()
				if !bytes.Contains([]byte(output), []byte("@@@@@@")) {
					t.Errorf("Expected output to contain ASCII art logo with @@ characters")
				}
			},
		},
		"MissingGitCommand": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				tmpDir := t.TempDir()

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.RepoPath = tmpDir
				app.execLookPath = func(name string) (string, error) {
					if name == "git" {
						return "", errors.New("git is not found in PATH")
					}
					return "/usr/bin/" + name, nil
				}
				app.Stdout = &stdout
				app.Stderr = &stderr

				app.isRepository = func(path string) (bool, error) {
					return true, nil
				}

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "git is not found in PATH",
			validateOutput: func(t *testing.T, app *App, stdout, stderr *bytes.Buffer, err error) {
				errOutput := stderr.String()
				if !bytes.Contains([]byte(errOutput), []byte("Error:")) {
					t.Errorf("Expected error output to mention an error, got: %s", errOutput)
				}
			},
		},
		"LockAcquisitionFailure": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockLocker := &MockLocker{
						AcquireErr: errors.New("lock acquisition failed"),
					}

					app = NewDefaultApp(config.VersionInfo{})
					app.Stdout = &stdout
					app.Stderr = &stderr
					app.Locker = mockLocker
					app.Logger = logger.New(false, "", true)
					app.exit = func(int) {}
					app.Config.RepoPath = tempDir

					app.isRepository = func(path string) (bool, error) {
						return true, nil
					}

					app.execLookPath = func(name string) (string, error) {
						return "/usr/bin/" + name, nil
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "lock acquisition failed",
			validateState: func(t *testing.T, app *App) {
				mockLocker := app.Locker.(*MockLocker)
				if !mockLocker.AcquireCalled {
					t.Error("Expected locker.Acquire to be called")
				}
			},
		},
		"GitbakRunFailure": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockLocker := &MockLocker{}

					mockGitbaker := &MockGitbaker{
						RunErr: errors.New("gitbak run failed"),
					}

					app = NewDefaultApp(config.VersionInfo{})
					app.Stdout = &stdout
					app.Stderr = &stderr
					app.Locker = mockLocker
					app.Logger = logger.New(false, "", true)
					app.exit = func(int) {}
					app.Gitbak = mockGitbaker
					app.Config.RepoPath = tempDir

					app.isRepository = func(path string) (bool, error) {
						return true, nil
					}

					app.execLookPath = func(name string) (string, error) {
						return "/usr/bin/" + name, nil
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "gitbak run failed",
			validateState: func(t *testing.T, app *App) {
				mockGitbaker := app.Gitbak.(*MockGitbaker)
				mockLocker := app.Locker.(*MockLocker)

				if !mockGitbaker.RunCalled {
					t.Error("Expected Gitbak.Run to be called")
				}

				if !mockLocker.ReleaseCalled {
					t.Error("Expected locker.Release to be called during cleanup")
				}

				if mockGitbaker.CommitsCount != 0 {
					t.Errorf("Expected CommitsCount to remain 0 on error, got: %d", mockGitbaker.CommitsCount)
				}
			},
		},
		"LockReleaseFailure": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockLocker := &MockLocker{
						ReleaseErr: errors.New("lock release failed"),
					}

					mockGitbaker := &MockGitbaker{
						RunErr: errors.New("gitbak run failed"),
					}

					app = NewDefaultApp(config.VersionInfo{})
					app.Stdout = &stdout
					app.Stderr = &stderr
					app.Locker = mockLocker
					app.Logger = logger.New(false, "", true)
					app.exit = func(int) {}
					app.Gitbak = mockGitbaker
					app.Config.RepoPath = tempDir

					app.isRepository = func(path string) (bool, error) {
						return true, nil
					}

					app.execLookPath = func(name string) (string, error) {
						return "/usr/bin/" + name, nil
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "gitbak run failed",
			validateState: func(t *testing.T, app *App) {
				mockLocker := app.Locker.(*MockLocker)
				if !mockLocker.ReleaseCalled {
					t.Error("Expected locker.Release to be called during cleanup")
				}
			},
		},
		"LoggerAssertionFailure": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockLocker := &MockLocker{}
					mockGitbaker := &MockGitbaker{}

					app = NewDefaultApp(config.VersionInfo{})
					app.Stdout = &stdout
					app.Stderr = &stderr
					app.Locker = mockLocker
					app.Logger = &MockLogger{}
					app.exit = func(int) {}
					app.Gitbak = mockGitbaker
					app.Config.RepoPath = tempDir

					app.isRepository = func(path string) (bool, error) {
						return true, nil
					}

					app.execLookPath = func(name string) (string, error) {
						return "/usr/bin/" + name, nil
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateState: func(t *testing.T, app *App) {
				mockGitbaker := app.Gitbak.(*MockGitbaker)
				if !mockGitbaker.RunCalled {
					t.Error("Expected Gitbak.Run to be called (signaling fallback logger worked)")
				}

				if mockGitbaker.CommitsCount != 1 {
					t.Errorf("Expected CommitsCount to be 1 after successful run, got: %d", mockGitbaker.CommitsCount)
				}
			},
		},
		"SuccessfulRun": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockGitbaker := &MockGitbaker{}
					mockLocker := &MockLocker{}

					app = &App{
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
						execLookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
						Stdout:       &stdout,
						Stderr:       &stderr,
						exit:         func(int) {},
						Gitbak:       mockGitbaker,
						isRepository: func(path string) (bool, error) {
							return true, nil
						},
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateState: func(t *testing.T, app *App) {
				mockLocker := app.Locker.(*MockLocker)
				mockGitbaker := app.Gitbak.(*MockGitbaker)

				if !mockLocker.AcquireCalled {
					t.Error("Expected locker.Acquire to be called")
				}
				if !mockGitbaker.RunCalled {
					t.Error("Expected Gitbak.Run to be called")
				}
				if !mockLocker.ReleaseCalled {
					t.Error("Expected locker.Release to be called during cleanup (even on success)")
				}

				if mockGitbaker.CommitsCount != 1 {
					t.Errorf("Expected CommitsCount to be 1 after successful run, got: %d", mockGitbaker.CommitsCount)
				}
			},
		},
		"ExternalGitbakProvider": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockGitbaker := &MockGitbaker{}

					app = NewTestApp()
					app.Config.RepoPath = tempDir
					app.Config.IntervalMinutes = 5

					app.Stdout = &bytes.Buffer{}
					app.Stderr = &bytes.Buffer{}

					app = WithMockLogger(app, &MockLogger{})
					app = WithMockLocker(app, &MockLocker{})
					app = WithSimpleIsRepository(app, func(path string) bool {
						return true
					})
					app = WithExecLookPath(app, func(string) (string, error) {
						return "/usr/bin/git", nil
					})

					app.Gitbak = mockGitbaker
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateState: func(t *testing.T, app *App) {
				mockGitbaker := app.Gitbak.(*MockGitbaker)
				if !mockGitbaker.RunCalled {
					t.Error("Expected Gitbak.Run to be called")
				}

				if mockGitbaker.CommitsCount != 1 {
					t.Errorf("Expected CommitsCount to be 1 after successful run, got: %d", mockGitbaker.CommitsCount)
				}
			},
		},
		"ContinueSessionFromExistingCommits": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					mockGitbaker := &MockGitbaker{
						CommitsCount: 5,
					}

					app = NewTestApp()
					app.Config.RepoPath = tempDir
					app.Config.IntervalMinutes = 5
					app.Config.ContinueSession = true
					app.Config.BranchName = "test-branch"

					app.Stdout = &bytes.Buffer{}
					app.Stderr = &bytes.Buffer{}

					app = WithMockLogger(app, &MockLogger{})
					app = WithMockLocker(app, &MockLocker{})
					app = WithSimpleIsRepository(app, func(path string) bool {
						return true
					})
					app = WithExecLookPath(app, func(string) (string, error) {
						return "/usr/bin/git", nil
					})

					app.Gitbak = mockGitbaker
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateState: func(t *testing.T, app *App) {
				mockGitbaker := app.Gitbak.(*MockGitbaker)
				if !mockGitbaker.RunCalled {
					t.Error("Expected Gitbak.Run to be called")
				}

				// In continue mode, we expect the commit counter to be incremented from the initial value
				if mockGitbaker.CommitsCount != 6 {
					t.Errorf("Expected CommitsCount to be 6 after continuing session, got: %d", mockGitbaker.CommitsCount)
				}
			},
		},
		"NotGitRepository": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				var app *App

				withTempWorkDir(t, func(tempDir string) {
					app = NewTestApp()
					app.Config.RepoPath = tempDir
					app.Stdout = &stdout
					app.Stderr = &stderr

					app = WithExecLookPath(app, func(file string) (string, error) {
						return "/usr/bin/git", nil // Make sure checkRequiredCommands passes
					})

					app = WithSimpleIsRepository(app, func(path string) bool {
						return false // Path is not a git repo
					})

					// First, Initialize should succeed (it doesn't check for git repo)
					if err := app.Initialize(); err != nil {
						t.Fatalf("Initialize() unexpectedly failed: %v", err)
					}
				})

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "git repository",
		},
		"InitializeFailure": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				tmpDir := t.TempDir()

				app := NewTestApp()
				app.Config.IntervalMinutes = -1 // Invalid config will cause Initialize to fail
				app.Config.RepoPath = tmpDir
				app.Stdout = &stdout
				app.Stderr = &stderr

				app.isRepository = func(path string) (bool, error) {
					return true, nil
				}

				ctx := context.Background()
				return app, ctx
			},
			expectError:   true,
			errorContains: "invalid",
		},
		"RunWithHelp": {
			setupFunc: func(t *testing.T) (*App, context.Context) {
				var stdout, stderr bytes.Buffer
				tmpDir := t.TempDir()

				app := NewDefaultApp(config.VersionInfo{})
				app.exit = func(int) {}
				app.Config.ShowHelp = true
				app.Config.RepoPath = tmpDir
				app.Stdout = &stdout
				app.Stderr = &stderr

				app.isRepository = func(path string) (bool, error) {
					return true, nil
				}

				ctx := context.Background()
				return app, ctx
			},
			expectError: false,
			validateOutput: func(t *testing.T, app *App, stdout, stderr *bytes.Buffer, err error) {
				// Help flag doesn't directly output anything, it's handled by flag package
				// But Run() should succeed when help flag is set
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			app, ctx := test.setupFunc(t)

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

			stdout, stderr := app.Stdout.(*bytes.Buffer), app.Stderr.(*bytes.Buffer)

			err := app.Run(ctx)

			if test.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !test.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if err != nil && test.errorContains != "" {
				if !bytes.Contains([]byte(err.Error()), []byte(test.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %v", test.errorContains, err)
				}
			}

			if test.validateOutput != nil {
				test.validateOutput(t, app, stdout, stderr, err)
			}

			if test.validateState != nil {
				test.validateState(t, app)
			}
		})
	}
}
