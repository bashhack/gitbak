package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/logger"
)

// TestRunWithVersionFlag tests the version flag case
func TestRunWithVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app := NewDefaultApp(config.VersionInfo{
		Version: "test-version",
		Commit:  "test-commit",
		Date:    "test-date",
	})
	app.exit = func(int) {}
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
	app.exit = func(int) {}
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
	app.exit = func(int) {}
	app.execLookPath = func(name string) (string, error) {
		if name == "git" {
			return "", errors.New("git not found")
		}
		return "/usr/bin/" + name, nil
	}
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

// TestRunWithLockAcquisitionFailure tests the case where lock acquisition fails
func TestRunWithLockAcquisitionFailure(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		mockLocker := &MockLocker{
			AcquireErr: errors.New("lock acquisition failed"),
		}

		app := NewDefaultApp(config.VersionInfo{})
		app.Stdout = &stdout
		app.Stderr = &stderr
		app.Locker = mockLocker
		app.Logger = logger.New(false, "", true)
		app.exit = func(int) {}

		app.isRepository = func(path string) (bool, error) {
			return true, nil
		}

		app.execLookPath = func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}

		ctx := context.Background()
		err := app.Run(ctx)
		if err == nil {
			t.Error("Expected Run to return an error when lock acquisition fails")
		}

		if !mockLocker.AcquireCalled {
			t.Error("Expected locker.Acquire to be called")
		}

		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("lock")) {
			t.Errorf("Expected error to mention lock acquisition, got: %v", err)
		}
	})
}

// TestRunWithGitbakRunFailure tests the case where Gitbak.Run fails
func TestRunWithGitbakRunFailure(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		mockLocker := &MockLocker{}

		mockGitbaker := &MockGitbaker{
			RunErr: errors.New("gitbak run failed"),
		}

		app := NewDefaultApp(config.VersionInfo{})
		app.Stdout = &stdout
		app.Stderr = &stderr
		app.Locker = mockLocker
		app.Logger = logger.New(false, "", true)
		app.exit = func(int) {}
		app.Gitbak = mockGitbaker

		app.isRepository = func(path string) (bool, error) {
			return true, nil
		}

		app.execLookPath = func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}

		ctx := context.Background()
		err := app.Run(ctx)
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
	})
}

// TestRunWithLockReleaseFailure tests the cleanup when lock release fails
func TestRunWithLockReleaseFailure(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		mockLocker := &MockLocker{
			ReleaseErr: errors.New("lock release failed"),
		}

		mockGitbaker := &MockGitbaker{
			RunErr: errors.New("gitbak run failed"),
		}

		app := NewDefaultApp(config.VersionInfo{})
		app.Stdout = &stdout
		app.Stderr = &stderr
		app.Locker = mockLocker
		app.Logger = logger.New(false, "", true)
		app.exit = func(int) {}
		app.Gitbak = mockGitbaker

		app.isRepository = func(path string) (bool, error) {
			return true, nil
		}

		app.execLookPath = func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}

		ctx := context.Background()
		err := app.Run(ctx)
		if err == nil {
			t.Error("Expected Run to return an error when Gitbak.Run fails")
		}

		if !mockLocker.ReleaseCalled {
			t.Error("Expected locker.Release to be called during cleanup")
		}
	})
}

// TestRunWithLoggerAssertionFailure tests the case where Logger type assertion fails
func TestRunWithLoggerAssertionFailure(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		mockLocker := &MockLocker{}

		mockGitbaker := &MockGitbaker{}

		app := NewDefaultApp(config.VersionInfo{})
		app.Stdout = &stdout
		app.Stderr = &stderr
		app.Locker = mockLocker
		app.Logger = &MockLogger{}
		app.exit = func(int) {}
		app.Gitbak = mockGitbaker

		app.isRepository = func(path string) (bool, error) {
			return true, nil
		}

		app.execLookPath = func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}

		ctx := context.Background()
		err := app.Run(ctx)
		if err != nil {
			// We're not expecting an error here, we're testing that the fallback logger works
			t.Errorf("Run returned unexpected error: %v", err)
		}

		if !mockGitbaker.RunCalled {
			t.Error("Expected Gitbak.Run to be called (signaling fallback logger worked)")
		}
	})
}

// TestRunComplete tests a successful run
func TestRunComplete(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		mockGitbaker := &MockGitbaker{}

		mockLocker := &MockLocker{}

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
			execLookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
			Stdout:       &stdout,
			Stderr:       &stderr,
			exit:         func(int) {},
			Gitbak:       mockGitbaker,
			isRepository: func(path string) (bool, error) {
				return true, nil
			},
		}

		ctx := context.Background()
		err := app.Run(ctx)
		if err != nil {
			t.Errorf("Run returned unexpected error: %v", err)
		}

		if !mockLocker.AcquireCalled {
			t.Error("Expected locker.Acquire to be called")
		}
		if !mockGitbaker.RunCalled {
			t.Error("Expected Gitbak.Run to be called")
		}

		if !mockLocker.ReleaseCalled {
			t.Error("Expected locker.Release to be called during cleanup (even on success)")
		}
	})
}

// TestRunWithGitbakProvidedExternally tests providing a mock Gitbaker
func TestRunWithGitbakProvidedExternally(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		mockGitbaker := &MockGitbaker{}

		app := NewTestApp()
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

		ctx := context.Background()
		err := app.Run(ctx)
		if err != nil {
			t.Errorf("Run returned unexpected error: %v", err)
		}

		if !mockGitbaker.RunCalled {
			t.Error("Expected Gitbak.Run to be called")
		}
	})
}

// Test to cover the lock acquisition error path
func TestRunWithLockAcquisitionError(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		mockLocker := &MockLocker{
			AcquireErr: errors.New("test acquisition error"),
		}

		app := NewTestApp()
		app.Config.RepoPath = tempDir

		app = WithMockLogger(app, &MockLogger{})
		app = WithMockLocker(app, mockLocker)
		app = WithExecLookPath(app, func(string) (string, error) {
			return "/usr/bin/git", nil
		})
		app = WithSimpleIsRepository(app, func(path string) bool {
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
	})
}

// Test to cover the deferred lock release path
func TestRunWithLockReleaseErrorDeferred(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		mockLocker := &MockLocker{
			ReleaseErr: errors.New("test release error"),
		}

		mockGitbaker := &MockGitbaker{
			RunErr: errors.New("gitbak run error"),
		}

		app := NewTestApp()
		app.Config.RepoPath = tempDir

		app = WithMockLogger(app, &MockLogger{})
		app = WithMockLocker(app, mockLocker)
		app = WithExecLookPath(app, func(string) (string, error) {
			return "/usr/bin/git", nil
		})
		app = WithSimpleIsRepository(app, func(path string) bool {
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
	})
}

// TestRunWithNormalGitbakCreation tests the normal Gitbak creation path
func TestRunWithNormalGitbakCreation(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		app := NewDefaultApp(config.VersionInfo{})
		app.Config.RepoPath = tempDir
		app.Config.IntervalMinutes = 5
		app.Config.BranchName = "test-branch"
		app.Config.CommitPrefix = "[test] "
		app.Config.Verbose = true

		app.Logger = logger.New(false, "", true)
		app.Locker = &MockLocker{}
		app.execLookPath = func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		}
		app.Stdout = &bytes.Buffer{}
		app.Stderr = &bytes.Buffer{}
		app.exit = func(int) {}

		app.isRepository = func(path string) (bool, error) {
			return true, nil
		}

		// Create a mock for Gitbak to avoid real git operations
		// This test is just to cover the creation code path, not the actual run
		mockGitbaker := &MockGitbaker{}
		app.Gitbak = mockGitbaker

		ctx := context.Background()
		err := app.Run(ctx)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !mockGitbaker.RunCalled {
			t.Error("Expected Gitbak.Run to be called")
		}
	})
}

// TestRunWithNotAGitRepoError tests the error case when a path is not a git repository
func TestRunWithNotAGitRepoError(t *testing.T) {
	withTempWorkDir(t, func(tempDir string) {
		var stdout, stderr bytes.Buffer

		app := NewTestApp()
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

		// But Run should fail because it does check for git repo
		ctx := context.Background()
		err := app.Run(ctx)

		if err == nil {
			t.Fatal("Expected Run() to fail when path is not a git repo, got nil")
		}

		if !bytes.Contains([]byte(err.Error()), []byte("git repository")) {
			t.Errorf("Expected error to mention git repository, got: %v", err)
		}
	})
}
