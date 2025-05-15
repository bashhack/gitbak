package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bashhack/gitbak/internal/config"
	"github.com/bashhack/gitbak/internal/constants"
	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
	"github.com/bashhack/gitbak/internal/git"
	"github.com/bashhack/gitbak/internal/lock"
	"github.com/bashhack/gitbak/internal/logger"
)

// Gitbaker performs Git operations
type Gitbaker interface {
	PrintSummary()
	Run(ctx context.Context) error
}

// Locker manages file locking
type Locker interface {
	Acquire() error
	Release() error
}

// AppOptions contains app configuration and dependencies.
// This struct allows injection of both required and optional dependencies,
// enabling flexible configuration and easier testing.
type AppOptions struct {
	// Config holds the application configuration settings (required).
	// The application will panic if this field is nil.
	Config *config.Config

	// Optional components

	// Logger provides logging functionality (optional, a default will be created if nil).
	// Used for both internal logging and user-facing messages.
	Logger logger.Logger

	// Locker manages repository locking (optional, a default will be created if nil).
	// Used to prevent multiple gitbak instances from running on the same repository.
	Locker Locker

	// Gitbak performs Git operations (optional, a default will be created if nil).
	// Handles the core functionality of repository monitoring and automatic commits.
	Gitbak Gitbaker

	// I/O dependencies

	// Stdout is the writer for standard output (optional, defaults to os.Stdout).
	// Used for user-facing messages and normal operation output.
	Stdout io.Writer

	// Stderr is the writer for error output (optional, defaults to os.Stderr).
	// Used for error messages and warnings.
	Stderr io.Writer

	// System dependencies

	// Exit is the function to terminate the application (optional, defaults to os.Exit).
	// Allows customization of the exit behavior, particularly useful in tests.
	Exit func(code int)

	// ExecLookPath is used to find executables in PATH (optional, defaults to exec.LookPath).
	// Used to locate the git executable.
	ExecLookPath func(file string) (string, error)

	// IsRepository checks if a path is a valid Git repository (optional, defaults to git.IsRepository).
	// Used during initialization to validate the repository path.
	IsRepository func(string) (bool, error)
}

// App is the main gitbak application.
// It orchestrates all components and manages the application lifecycle,
// handling initialization, command execution, and cleanup.
type App struct {
	// Config holds the application configuration and settings.
	Config *config.Config

	// Logger provides logging functionality for both internal and user-facing messages.
	Logger logger.Logger

	// Locker manages repository locking to prevent concurrent gitbak instances.
	Locker Locker

	// Gitbak performs Git operations and implements the core gitbak functionality.
	Gitbak Gitbaker

	// I/O streams

	// Stdout is the writer for standard output messages.
	Stdout io.Writer

	// Stderr is the writer for error messages and warnings.
	Stderr io.Writer

	// System dependencies

	// exit is the function to terminate the application with a status code.
	exit func(code int)

	// execLookPath is used to find the git executable in the system PATH.
	execLookPath func(file string) (string, error)

	// isRepository checks if a path is a valid Git repository.
	isRepository func(string) (bool, error)
}

// NewDefaultApp creates an App with standard dependencies.
// It initializes a new Config with the provided version information,
// loads environment variables, and sets up standard OS dependencies.
// This is the primary application constructor for normal usage.
//
// Parameters:
//   - versionInfo: Contains version, commit, and build date information
//
// Returns:
//   - An App instance ready for initialization and execution
func NewDefaultApp(versionInfo config.VersionInfo) *App {
	cfg := config.New()
	cfg.VersionInfo = versionInfo
	cfg.LoadFromEnvironment()

	opts := AppOptions{
		Config:       cfg,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Exit:         os.Exit,
		ExecLookPath: exec.LookPath,
		IsRepository: git.IsRepository,
	}

	return NewApp(opts)
}

// NewApp creates an App with custom dependencies specified in opts.
// It validates that required dependencies are provided and panics
// if Config is nil. For any optional dependencies that are nil,
// this function will create appropriate defaults during initialization.
//
// Parameters:
//   - opts: AppOptions struct containing dependencies and configuration
//
// Returns:
//   - An App instance with the provided configuration and dependencies
//
// Panics:
//   - If opts.Config is nil
func NewApp(opts AppOptions) *App {
	if opts.Config == nil {
		panic("Config is required in AppOptions")
	}

	app := &App{
		Config:       opts.Config,
		Logger:       opts.Logger,
		Locker:       opts.Locker,
		Gitbak:       opts.Gitbak,
		Stdout:       opts.Stdout,
		Stderr:       opts.Stderr,
		exit:         opts.Exit,
		execLookPath: opts.ExecLookPath,
		isRepository: opts.IsRepository,
	}

	// Set defaults for nil dependencies
	if app.Stdout == nil {
		app.Stdout = os.Stdout
	}
	if app.Stderr == nil {
		app.Stderr = os.Stderr
	}
	if app.exit == nil {
		app.exit = os.Exit
	}
	if app.execLookPath == nil {
		app.execLookPath = exec.LookPath
	}
	if app.isRepository == nil {
		app.isRepository = git.IsRepository
	}

	return app
}

// Initialize sets up components not provided during construction
func (a *App) Initialize() error {
	if err := a.Config.Finalize(); err != nil {
		// Since Config.Finalize() already returns a properly wrapped error,
		// we don't need to wrap it again if it's already our error type
		if gitbakErrors.Is(err, gitbakErrors.ErrInvalidConfiguration) {
			return err
		}
		return gitbakErrors.Wrap(gitbakErrors.ErrInvalidConfiguration, err.Error())
	}

	if a.Logger == nil {
		a.Logger = logger.New(a.Config.Debug, a.Config.LogFile, a.Config.Verbose)
	}

	if a.Locker == nil {
		locker, err := lock.New(a.Config.RepoPath)
		if err != nil {
			return gitbakErrors.Wrap(err, "failed to initialize lock")
		}
		a.Locker = locker
	}

	if a.Gitbak == nil {
		gitbakConfig := git.GitbakConfig{
			RepoPath:        a.Config.RepoPath,
			IntervalMinutes: a.Config.IntervalMinutes,
			BranchName:      a.Config.BranchName,
			CommitPrefix:    a.Config.CommitPrefix,
			CreateBranch:    a.Config.CreateBranch,
			Verbose:         a.Config.Verbose,
			ShowNoChanges:   a.Config.ShowNoChanges,
			ContinueSession: a.Config.ContinueSession,
			NonInteractive:  a.Config.NonInteractive,
			MaxRetries:      a.Config.MaxRetries,
		}
		gitbak, err := git.NewGitbak(gitbakConfig, a.Logger)
		if err != nil {
			return fmt.Errorf("failed to create gitbak instance: %w", err)
		}
		a.Gitbak = gitbak
	}

	return nil
}

// Run executes the application with the given context
// Handles special flags and runs the gitbak process
func (a *App) Run(ctx context.Context) error {
	// Ensure the app is fully initialised before doing any work.
	if err := a.Initialize(); err != nil {
		return err
	}

	// Handle special flags first
	if a.Config.Version {
		a.ShowVersion()
		return nil
	}

	if a.Config.ShowLogo {
		a.ShowLogo()
		return nil
	}

	if a.Config.ShowHelp {
		// This should never be reached normally because flag package
		// handles help internally, but we check it just in case
		return nil
	}

	// Ensure we always clean up logger / lock, even on early error paths
	defer func() {
		if err := a.Close(); err != nil {
			_, _ = fmt.Fprintf(a.Stderr, "❌ Error during cleanup: %v\n", err)
		}
	}()

	// Verify prerequisites
	if err := a.checkRequiredCommands(); err != nil {
		_, _ = fmt.Fprintf(a.Stderr, "❌ Error: %v. Please install it and try again.\n", err)
		return err
	}

	isRepo, err := a.isRepository(a.Config.RepoPath)
	if err != nil {
		a.Logger.Warning("Failed to check if path is a git repository: %v", err)
		return gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed, err.Error())
	}
	if !isRepo {
		return gitbakErrors.ErrNotGitRepository
	}
	a.Logger.Info("Git repository verified")

	// Acquire resource lock
	if err := a.Locker.Acquire(); err != nil {
		// Since Locker.Acquire() already returns a properly wrapped error,
		// we don't need to wrap it again
		if gitbakErrors.Is(err, gitbakErrors.ErrAlreadyRunning) {
			return err
		}
		return gitbakErrors.Wrap(gitbakErrors.ErrLockAcquisitionFailure, err.Error())
	}

	// Run main gitbak process
	return a.Gitbak.Run(ctx)
}

// ShowVersion displays version information
func (a *App) ShowVersion() {
	_, _ = fmt.Fprintf(a.Stdout, "gitbak %s (%s) built on %s\n",
		a.Config.VersionInfo.Version,
		a.Config.VersionInfo.Commit,
		a.Config.VersionInfo.Date)
}

// ShowLogo displays ASCII art logo
func (a *App) ShowLogo() {
	_, _ = fmt.Fprintln(a.Stdout, constants.Logo)
	_, _ = fmt.Fprintln(a.Stdout, "")

	asciiArtWidth := 80
	padding := (asciiArtWidth - len(constants.Tagline)) / 2
	centeredTagline := fmt.Sprintf("%s%s", strings.Repeat(" ", padding), constants.Tagline)
	_, _ = fmt.Fprintln(a.Stdout, centeredTagline)
}

// checkRequiredCommands verifies git is available in PATH
func (a *App) checkRequiredCommands() error {
	_, err := a.execLookPath("git")
	if err != nil {
		return fmt.Errorf("git is not found in PATH")
	}
	return nil
}

// Close releases resources held by the App
func (a *App) Close() error {
	var errs []error

	// Release lock if it exists
	if a.Locker != nil {
		if err := a.Locker.Release(); err != nil {
			if a.Logger != nil {
				a.Logger.Error("Failed to release lock during cleanup: %v", err)
			} else {
				_, _ = fmt.Fprintf(a.Stderr, "❌ Failed to release lock during cleanup: %v\n", err)
			}
			errs = append(errs, err)
		}
	}

	if a.Logger != nil {
		if err := a.Logger.Close(); err != nil {
			_, _ = fmt.Fprintf(a.Stderr, "❌ Failed to close logger: %v\n", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return gitbakErrors.Join(errs...)
	}
	return nil
}

// CleanupOnSignal releases locks and shows a summary on interruption
func (a *App) CleanupOnSignal() {
	// Close resources...
	if err := a.Close(); err != nil {
		_, _ = fmt.Fprintf(a.Stderr, "❌ Error during cleanup: %v\n", err)
	}

	// Show summary only if we're not running in --logo or --version mode
	if !a.Config.ShowLogo && !a.Config.Version && a.Gitbak != nil {
		a.Gitbak.PrintSummary()
	}
}
