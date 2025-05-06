package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bashhack/gitbak/internal/common"
	"github.com/bashhack/gitbak/internal/config"
	internalErrors "github.com/bashhack/gitbak/internal/errors"
	"github.com/bashhack/gitbak/internal/git"
	"github.com/bashhack/gitbak/internal/lock"
	"github.com/bashhack/gitbak/internal/logger"
)

// Core interfaces for dependency injection

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

// Logger alias to common.Logger
type Logger = common.Logger

// AppOptions contains app configuration and dependencies
type AppOptions struct {
	// Required
	Config *config.Config

	// Optional components
	Logger Logger
	Locker Locker
	Gitbak Gitbaker

	// I/O dependencies
	Stdout io.Writer
	Stderr io.Writer

	// System dependencies
	Exit         func(code int)
	ExecLookPath func(file string) (string, error)
	IsRepository func(string) bool
}

// App is the main gitbak application
type App struct {
	Config *config.Config
	Logger Logger
	Locker Locker
	Gitbak Gitbaker

	// I/O streams
	Stdout io.Writer
	Stderr io.Writer

	// System dependencies
	exit         func(code int)
	execLookPath func(file string) (string, error)
	isRepository func(string) bool
}

// NewDefaultApp creates an App with standard dependencies
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

// NewApp creates an App with custom dependencies
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
		if internalErrors.Is(err, internalErrors.ErrInvalidConfiguration) {
			return err
		}
		return internalErrors.Wrap(internalErrors.ErrInvalidConfiguration, err.Error())
	}

	if a.Logger == nil {
		a.Logger = logger.New(a.Config.Debug, a.Config.LogFile, a.Config.Verbose)
	}

	if a.Locker == nil {
		locker, err := lock.New(a.Config.RepoPath)
		if err != nil {
			return internalErrors.Wrap(err, "failed to initialize lock")
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
		}
		a.Gitbak = git.NewGitbak(gitbakConfig, a.Logger)
	}

	return nil
}

// Run executes the application with the given context
// Handles special flags and runs the gitbak process
func (a *App) Run(ctx context.Context) error {
	// Handle special flags first
	if a.Config.Version {
		a.ShowVersion()
		return nil
	}

	if a.Config.ShowLogo {
		a.ShowLogo()
		return nil
	}

	// Verify prerequisites
	if err := a.checkRequiredCommands(); err != nil {
		_, _ = fmt.Fprintf(a.Stderr, "❌ Error: %v. Please install it and try again.\n", err)
		return err
	}

	if !a.isRepository(a.Config.RepoPath) {
		return internalErrors.ErrNotGitRepository
	}
	a.Logger.Info("Git repository verified")

	// Acquire resource lock
	if err := a.Locker.Acquire(); err != nil {
		// Since Locker.Acquire() already returns a properly wrapped error,
		// we don't need to wrap it again
		if internalErrors.Is(err, internalErrors.ErrAlreadyRunning) {
			return err
		}
		return internalErrors.Wrap(internalErrors.ErrLockAcquisitionFailure, err.Error())
	}
	defer func() {
		if releaseErr := a.Locker.Release(); releaseErr != nil {
			a.Logger.Error("Failed to release lock during cleanup: %v", releaseErr)
		}
	}()

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
	logo := `@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@%####%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@%#######%%@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@%#*+++=====+++##%@@@@@@@@@@@@@@@@@@%%#+++=+++*+++++*#%@@@@@@@@@@@@@
@@@@@@@@@@@%*====+#%%%%%*====+#%@@@@@@@@@@@@@@%#+++*#%%%%%%%%%%#*++*%@@@@@@@@@@@
@@@@@@@@@%*+=====+%%%%%%*======+#%@@@@@@@@@@@#++*%%@@@@@%++@@@@@@%#++#@@@@@@@@@@
@@@@@@@@%+========+%%%%*=========#%@@@@@@@@@#++%@@@@@@@@%++%@@@@@@@%*+*%@@@@@@@@
@@@@@@@%*==========+**+===========#%@@@@@@@#++%@@@@@@@@@%++%@@@@@@@@%*=*%@@@@@@@
@@@@@@@*++###*++===+##*+===++*##+=+%@@@@@@%++#@@@@@@@@@@%++%@@@@@@@@@%*=#@@@@@@@
@@@@@@@*=+%%%%%%*=*%#*%#++#%%%%%*==++#*++++=*%@@@@@@@@@@#++#@@@@@@@@@@#+#@@@@@@@
@@@@@@@*=+%%%%%#*=*%#*%#++#%%%%%*=+#%%%####=*%@@@@@@@@@@#+=+%@@@@@@@@@#+#@@@@@@@
@@@@@@@#+=***++====+**+=====+***+=+%@@@@@@%+=#@@@@@@@@@@@%#*+*%@@@@@@%++#@@@@@@@
@@@@@@@%*+=========****+=========+#%@@@@@@@#++#@@@@@@@@@@@@@#*+#@@@@%*+*@@@@@@@@
@@@@@@@@%*========*%%%%#========+#%@@@@@@@@@#=+#@@@@@@@@@@@@@@%%@@@%*+*%@@@@@@@@
@@@@@@@@@%#+=====+%%@%%%*+====+*%@@@@@@@@@@@%#++*%@@@@@@@@@@@@@@@%#++#%@@@@@@@@@
@@@@@@@@@@@%*+===+######+===+*%%@@@@@@@@@@@@@@%#*+**#%%%@@%%%%##*++*%@@@@@@@@@@@
@@@@@@@@@@@@@%##*++=====++*#%@@@@@@@%*+*%@@@@@@@%%#**++++++++++**#%@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@%%%##++*#%#####%##*+*#*+*##%####%%##***+*###%%@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@%#***##****#***+**++***##***##*****#%@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%%%%%#**%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@%##%@@@@%%%%%%@@%##%@@@@@@@@@@@@@@@@@@@%##%@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@%%%%%%**#%%%%#++%%%%%%+=#%%%@@@@@@@%%%@@@@%==#@@@@@@@@@@@@@@@@@
@@@@@@@@@@%#*+++++*%%#+*+#%%#++==+++#%%+=+*++*%%%%#*+++*#%%#==#%#+*#@@@@@@@@@@@@
@@@@@@@@@%*=+###+=+%%%#+=*%%%#*==###%%%+=+##*++#%#**###+=+%#==*++*#%@@@@@@@@@@@@
@@@@@@@@@%+=*%%%#=+%%%%+=*%%%%#==#%%%%%+=*%%%*=+%#*+**#+=+%#====+#%@@@@@@@@@@@@@
@@@@@@@@@%*++*#**++#%#*+=+*#%%#+=*###%%+=+##*++#%*=+###+=+%#==##*+*%@@@@@@@@@@@@
@@@@@@@@@@%#*****++%%*******%%%%****#%%#*#****%%%%#*******%%**%@%%##%@@@@@@@@@@@
@@@@@@@@@@%*****++*%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@%####%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@`
	_, _ = fmt.Fprintln(a.Stdout, logo)
	_, _ = fmt.Fprintln(a.Stdout, "")

	tagline := "Automatic Commit Safety Net for Pair Programming"
	asciiArtWidth := 80
	padding := (asciiArtWidth - len(tagline)) / 2
	centeredTagline := fmt.Sprintf("%s%s", strings.Repeat(" ", padding), tagline)
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

// CleanupOnSignal releases locks and shows summary on interruption
func (a *App) CleanupOnSignal() {
	if a.Locker != nil {
		if err := a.Locker.Release(); err != nil {
			if a.Logger != nil {
				a.Logger.Error("Failed to release lock during cleanup: %v", err)
			} else {
				_, _ = fmt.Fprintf(a.Stderr, "❌ Failed to release lock during cleanup: %v\n", err)
			}
		}
	}

	if a.Gitbak != nil {
		a.Gitbak.PrintSummary()
	}
}
