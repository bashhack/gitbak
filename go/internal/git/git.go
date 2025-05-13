package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
	"github.com/bashhack/gitbak/internal/logger"
)

// GitbakConfig contains configuration for a gitbak instance.
// This struct holds all the settings that control gitbak's behavior,
// including repository location, commit preferences, output options,
// and error handling settings.
type GitbakConfig struct {
	// RepoPath specifies the filesystem path to the Git repository.
	// Can be absolute or relative path. If empty, validation will fail.
	RepoPath string

	// IntervalMinutes defines how often (in minutes) gitbak checks for changes.
	// This can be a fractional value (e.g. 0.5 for 30 seconds).
	// Must be greater than 0.
	IntervalMinutes float64

	// BranchName specifies the Git branch to use for checkpoint commits.
	// If CreateBranch is true, this branch will be created.
	// If CreateBranch is false, this branch must already exist.
	// If ContinueSession is true, this should be an existing gitbak branch.
	BranchName string

	// CommitPrefix is prepended to all commit messages.
	// Used to identify gitbak commits and extract commit numbers.
	CommitPrefix string

	// CreateBranch determines whether to create a new branch or use existing one.
	// If true, a new branch named BranchName will be created.
	// If false, gitbak will use the existing branch specified by BranchName.
	// ContinueSession implicitly sets this to false.
	CreateBranch bool

	// ContinueSession enables continuation mode for resuming a previous session.
	// When true, gitbak finds the last commit number and continues numbering from there.
	// Requires that previous gitbak commits exist on the specified branch.
	ContinueSession bool

	// Verbose controls the amount of informational output.
	// When true, gitbak provides detailed status updates.
	// When false, only essential messages are shown.
	Verbose bool

	// ShowNoChanges determines whether to report when no changes are detected.
	// When true, gitbak logs a message at each interval even if nothing changed.
	// When false, these messages are suppressed.
	ShowNoChanges bool

	// NonInteractive disables any prompts and uses default responses.
	// Useful for running gitbak in automated environments.
	NonInteractive bool

	// MaxRetries defines how many consecutive identical errors are allowed before exiting.
	// If zero, gitbak will retry indefinitely.
	// Errors of different types or successful operations reset this counter.
	MaxRetries int
}

// Validate sanity-checks the config and returns an error if something is wrong.
// It ensures all required fields have valid values before gitbak starts running.
// This helps prevent runtime errors by catching configuration issues early.
//
// The following validations are performed:
//   - RepoPath must not be empty
//   - IntervalMinutes must be greater than 0
//   - BranchName must not be empty
//   - CommitPrefix must not be empty
//   - MaxRetries must not be negative
//
// Returns nil if the configuration is valid, or an error describing the issue.
func (c *GitbakConfig) Validate() error {
	if c.RepoPath == "" {
		return fmt.Errorf("RepoPath must not be empty")
	}
	if c.IntervalMinutes <= 0 {
		return fmt.Errorf("IntervalMinutes must be > 0 (got %.2f)", c.IntervalMinutes)
	}
	if c.BranchName == "" {
		return fmt.Errorf("BranchName must not be empty")
	}
	if c.CommitPrefix == "" {
		return fmt.Errorf("CommitPrefix must not be empty")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries cannot be negative (got %d)", c.MaxRetries)
	}
	return nil
}

// Gitbak monitors and auto-commits changes to a git repository.
// It provides the core functionality for automatically committing changes at
// regular intervals, with features for branch management, error recovery,
// and session continuation.
type Gitbak struct {
	// config holds all the settings for this gitbak instance
	config GitbakConfig

	// logger handles all output messages with appropriate formatting
	logger logger.Logger

	// executor runs Git commands and captures their output
	executor CommandExecutor

	// interactor handles any user interaction needed during operation
	interactor UserInteractor

	// commitsCount tracks the total number of commits made in this session
	commitsCount int

	// startTime records when this gitbak instance began running
	startTime time.Time

	// originalBranch stores the branch name that was active when gitbak started
	originalBranch string
}

// NewGitbak creates a new gitbak instance with default dependencies.
// This is the primary constructor for creating a gitbak instance with standard
// components. It validates the configuration and sets up all required dependencies.
//
// Parameters:
//   - config: The configuration for this gitbak instance
//   - logger: The logger to use for output messages
//
// Returns:
//   - A configured gitbak instance ready to run
//   - An error if the configuration is invalid or initialization fails
//
// Example:
//
//	cfg := GitbakConfig{
//	    RepoPath: "/path/to/repo",
//	    IntervalMinutes: 5,
//	    BranchName: "gitbak-session",
//	    CommitPrefix: "[gitbak]",
//	    CreateBranch: true,
//	}
//	gitbak, err := NewGitbak(cfg, logger)
func NewGitbak(config GitbakConfig, logger logger.Logger) (*Gitbak, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid gitbak configuration: %w", err)
	}

	executor := NewExecExecutor()

	var interactor UserInteractor
	if config.NonInteractive {
		interactor = NewNonInteractiveInteractor()
	} else {
		interactor = NewDefaultInteractor(logger)
	}

	return NewGitbakWithDeps(config, logger, executor, interactor)
}

// NewGitbakWithDeps creates a new gitbak instance with custom dependencies
func NewGitbakWithDeps(
	config GitbakConfig,
	logger logger.Logger,
	executor CommandExecutor,
	interactor UserInteractor,
) (*Gitbak, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid gitbak configuration: %w", err)
	}

	return &Gitbak{
		config:       config,
		logger:       logger,
		executor:     executor,
		interactor:   interactor,
		commitsCount: 0,
		startTime:    time.Now(),
	}, nil
}

// IsRepository checks if the given path is a git repository
// Returns true if it is a repository, false otherwise.
// If path is not a repository due to git exit code 128, returns (false, nil).
// For other errors (git not found, permission issues, etc), returns (false, err).
func IsRepository(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	executor := NewExecExecutor()
	// Use background context since this is a utility function
	ctx := context.Background()
	if err := executor.Execute(ctx, cmd); err != nil {
		// Exit code 128 is git's generic fatal error code - for this command,
		// it typically means the directory is not part of a git repository,
		// but...it could also indicate other repository-related issues.
		//
		// While I'm treating this as a "not a repository" case, I am knowingly
		// grouping together what could be a number of different issues. For the purposes
		// of this function, I think it's reasonable to treat them all the same -
		// as almost any issue with the repository will be fatal to gitbak.

		var exitErr *exec.ExitError
		if gitbakErrors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return false, nil
		}

		// Unexpected failure (git binary missing, permissions, etc)
		return false, err
	}
	return true, nil
}

// Run starts the gitbak process with the given context for cancellation
func (g *Gitbak) Run(ctx context.Context) error {
	g.startTime = time.Now()

	if err := g.initialize(ctx); err != nil {
		return err
	}

	return g.monitoringLoop(ctx)
}

// initialize prepares the gitbak session by detecting the original branch
// and configuring the appropriate session mode
func (g *Gitbak) initialize(ctx context.Context) error {
	var err error

	g.originalBranch, err = g.getCurrentBranch(ctx)
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		// Check if it's already a GitError
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.Wrap(err, "failed to get current branch")
	}
	g.logger.Info("Starting gitbak on branch: %s", g.originalBranch)

	if g.config.ContinueSession {
		if err := g.setupContinueSession(ctx); err != nil {
			return err
		}
	} else if g.config.CreateBranch {
		if err := g.setupNewBranchSession(ctx); err != nil {
			return err
		}
	} else {
		g.setupCurrentBranchSession(ctx)
	}

	g.displayStartupInfo()
	return nil
}

// setupContinueSession configures gitbak for continuing a previous session
func (g *Gitbak) setupContinueSession(ctx context.Context) error {
	g.config.CreateBranch = false
	g.logger.StatusMessage("🔄 Continuing gitbak session on branch: %s", g.originalBranch)

	highestNum, err := g.findHighestCommitNumber(ctx)
	if err != nil {
		g.logger.Warning("Failed to find highest commit number: %v", err)
		g.logger.InfoToUser("No previous commits found with prefix '%s' - starting from commit #1", g.config.CommitPrefix)
	} else if highestNum > 0 {
		// Initialize our commits count to the highest number we found
		// This ensures the commit counter in monitoringLoop starts at the right value
		g.commitsCount = highestNum
		g.logger.InfoToUser("Found previous commits - starting from commit #%d", highestNum+1)
	} else {
		g.logger.InfoToUser("No previous commits found with prefix '%s' - starting from commit #1", g.config.CommitPrefix)
	}

	return nil
}

// setupNewBranchSession creates and switches to a new branch
func (g *Gitbak) setupNewBranchSession(ctx context.Context) error {
	hasChanges, err := g.hasUncommittedChanges(ctx)
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.NewGitError("status", nil, gitbakErrors.Wrap(err, "failed to check for uncommitted changes"), "")
	}

	if hasChanges {
		g.logger.WarningToUser("You have uncommitted changes.")
		if err := g.handleUncommittedChanges(ctx); err != nil {
			return err
		}
	}

	if err := g.handleBranchName(ctx); err != nil {
		return err
	}

	if err := g.createAndCheckoutBranch(ctx); err != nil {
		return err
	}

	return nil
}

// setupCurrentBranchSession uses the current branch for commits
func (g *Gitbak) setupCurrentBranchSession(ctx context.Context) {
	currentBranch, err := g.getCurrentBranch(ctx)
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		g.logger.StatusMessage("🌿 Using current branch: unknown")
		return
	}
	g.logger.StatusMessage("🌿 Using current branch: %s", currentBranch)
}

// handleUncommittedChanges prompts the user about uncommitted changes
func (g *Gitbak) handleUncommittedChanges(ctx context.Context) error {
	shouldCommit := g.promptForCommit()

	if shouldCommit {
		addArgs := []string{"."}
		if err := g.runGitCommand(ctx, "add", "."); err != nil {
			return gitbakErrors.NewGitError("add", addArgs, err, "failed to stage changes")
		}

		commitMsg := "Manual commit before starting gitbak session"
		commitArgs := []string{"-m", commitMsg}
		if err := g.runGitCommand(ctx, "commit", "-m", commitMsg); err != nil {
			return gitbakErrors.NewGitError("commit", commitArgs, err, "failed to create initial commit")
		}

		g.logger.Success("Created initial commit")
	}

	return nil
}

// handleBranchName manages branch name conflicts
func (g *Gitbak) handleBranchName(ctx context.Context) error {
	branchExists, err := g.branchExists(ctx, g.config.BranchName)
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.NewGitError("show-ref", []string{g.config.BranchName},
			gitbakErrors.Wrap(err, "failed to check if branch exists"), "")
	}

	if branchExists {
		g.logger.WarningToUser("Branch '%s' already exists.", g.config.BranchName)

		var shouldChangeBranch bool
		if g.config.NonInteractive {
			g.logger.Info("Non-interactive mode: automatically using a different branch name")
			shouldChangeBranch = true
		} else {
			shouldChangeBranch = g.promptYesNo("Would you like to use a different branch name?")
		}

		if shouldChangeBranch {
			g.config.BranchName = fmt.Sprintf("%s-%s", g.config.BranchName, time.Now().Format("150405"))
			g.logger.StatusMessage("🌿 Using new branch name: %s", g.config.BranchName)
		}
	}

	return nil
}

// createAndCheckoutBranch creates and checks out a new branch
func (g *Gitbak) createAndCheckoutBranch(ctx context.Context) error {
	args := []string{"-b", g.config.BranchName}
	err := g.runGitCommand(ctx, "checkout", "-b", g.config.BranchName)
	if err != nil {
		return gitbakErrors.NewGitError("checkout", args, err, "failed to create new branch")
	}

	g.logger.StatusMessage("🌿 Created and switched to new branch: %s", g.config.BranchName)
	return nil
}

// displayStartupInfo outputs the active configuration to the user
func (g *Gitbak) displayStartupInfo() {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	g.logger.StatusMessage("🔄 gitbak started at %s", timestamp)
	g.logger.StatusMessage("📂 Repository: %s", g.config.RepoPath)
	g.logger.StatusMessage("⏱️ Interval: %.2f minutes", g.config.IntervalMinutes)
	g.logger.StatusMessage("📝 Commit prefix: %s", g.config.CommitPrefix)
	g.logger.StatusMessage("🔊 Verbose mode: %t", g.config.Verbose)
	g.logger.StatusMessage("🔔 Show no-changes messages: %t", g.config.ShowNoChanges)
	g.logger.StatusMessage("❓ Press Ctrl+C to stop and view session summary")
}

// tryOperation executes the provided operation function, tracks errors, and implements retry logic.
// Returns error if operation fails too many times based on MaxRetries.
// NOTE: This is exported for testing purposes but should not be used directly by clients!
func (g *Gitbak) tryOperation(
	ctx context.Context,
	errorState *struct {
		consecutiveErrors int
		lastErrorMsg      string
	},
	operation func() error,
) error {
	err := operation()
	if err != nil {
		g.logger.Error("Error in operation: %v", err)
		g.logger.WarningToUser("Error occurred: %v", err)

		currentErrorMsg := err.Error()
		if currentErrorMsg == errorState.lastErrorMsg {
			errorState.consecutiveErrors++
		} else {
			errorState.consecutiveErrors = 1
			errorState.lastErrorMsg = currentErrorMsg
		}

		// Using '>' instead of '>=' to ensure MaxRetries = 1 allows one retry attempt
		if g.config.MaxRetries > 0 && errorState.consecutiveErrors > g.config.MaxRetries {
			g.logger.Error("Reached maximum number of consecutive errors (%d). Stopping gitbak.", g.config.MaxRetries)
			g.logger.WarningToUser("Too many consecutive errors (same error %d times in a row). Stopping gitbak.", errorState.consecutiveErrors)
			return gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed,
				fmt.Sprintf("maximum retries (%d) exceeded with error: %v", g.config.MaxRetries, err))
		}
		return err
	}

	// Reset consecutive errors on success
	errorState.consecutiveErrors = 0
	errorState.lastErrorMsg = ""
	return nil
}

// monitoringLoop periodically checks for changes and creates commits.
// It runs until the context is canceled or an unrecoverable error occurs.
func (g *Gitbak) monitoringLoop(ctx context.Context) error {
	// Initialize commit counter based on commits count
	// If we're in continue mode, g.commitsCount was already set in setupContinueSession
	commitCounter := g.commitsCount + 1

	// Convert interval minutes (float) to duration for more precise control
	interval := time.Duration(g.config.IntervalMinutes*60*1000) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track consecutive errors for potential bail-out
	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Received cancellation signal, shutting down gracefully...")
			return ctx.Err()

		case <-ticker.C:
			opErr := g.tryOperation(ctx, &errorState, func() error {
				commitWasCreated := false

				if err := g.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated); err != nil {
					return err
				}

				if commitWasCreated {
					commitCounter++
				}

				return nil
			})

			// If the operation hit max retries, bubble up the fatal error
			if opErr != nil && errorState.consecutiveErrors > g.config.MaxRetries {
				return opErr
			}
		}
	}
}

// checkAndCommitChanges checks for uncommitted changes and creates a commit if found.
func (g *Gitbak) checkAndCommitChanges(ctx context.Context, commitCounter int, commitWasCreated *bool) error {
	hasChanges, err := g.hasUncommittedChanges(ctx)
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.NewGitError("status", nil,
			gitbakErrors.Wrap(err, "failed to check git status"), "")
	}

	if hasChanges {
		*commitWasCreated = true
		return g.createCommit(ctx, commitCounter)
	} else {
		*commitWasCreated = false
		if g.config.ShowNoChanges && g.config.Verbose {
			g.logger.InfoToUser("No changes to commit at %s", time.Now().Format("15:04:05"))
			g.logger.Info("No changes to commit detected")
		}
	}

	return nil
}

// createCommit stages all changes and creates a commit with the configured prefix.
func (g *Gitbak) createCommit(ctx context.Context, commitCounter int) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	addArgs := []string{"."}
	err := g.runGitCommand(ctx, "add", ".")
	if err != nil {
		g.logger.Warning("Failed to stage changes: %v", err)
		g.logger.WarningToUser("Failed to stage changes: %v", err)
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.NewGitError("add", addArgs,
			gitbakErrors.Wrap(err, "failed to stage changes"), "")
	}

	commitMsg := fmt.Sprintf("%s #%d - %s", g.config.CommitPrefix, commitCounter, timestamp)
	commitArgs := []string{"-m", commitMsg}
	err = g.runGitCommand(ctx, "commit", "-m", commitMsg)
	if err != nil {
		g.logger.Warning("Failed to create commit: %v", err)
		g.logger.WarningToUser("Failed to create commit: %v", err)
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if gitbakErrors.Is(err, gitbakErrors.ErrGitOperationFailed) {
			return err
		}
		return gitbakErrors.NewGitError("commit", commitArgs,
			gitbakErrors.Wrap(err, "failed to create commit"), "")
	}

	g.logger.Success("Commit #%d created at %s", commitCounter, timestamp)
	g.logger.Info("Successfully created commit #%d", commitCounter)

	g.commitsCount = commitCounter

	return nil
}

// PrintSummary prints a summary of the gitbak session
func (g *Gitbak) PrintSummary() {
	duration := time.Since(g.startTime)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	g.logger.StatusMessage("")
	g.logger.StatusMessage("---------------------------------------------")
	g.logger.StatusMessage("📊 gitbak Session Summary")
	g.logger.StatusMessage("---------------------------------------------")
	g.logger.StatusMessage("✅ Total commits made: %d", g.commitsCount)
	g.logger.StatusMessage("⏱️  Session duration: %dh %dm %ds", hours, minutes, seconds)

	if g.config.CreateBranch {
		g.logger.StatusMessage("🌿 Working branch: %s", g.config.BranchName)
		g.logger.StatusMessage("")
		g.logger.StatusMessage("To merge these changes to your original branch:")
		g.logger.StatusMessage("  git checkout %s", g.originalBranch)
		g.logger.StatusMessage("  git merge %s", g.config.BranchName)
		g.logger.StatusMessage("")
		g.logger.StatusMessage("To squash all commits into one:")
		g.logger.StatusMessage("  git checkout %s", g.originalBranch)
		g.logger.StatusMessage("  git merge --squash %s", g.config.BranchName)
		g.logger.StatusMessage("  git commit -m \"Merged gitbak session\"")
	} else {
		g.logger.StatusMessage("🌿 Working branch: %s (unchanged)", g.originalBranch)
	}

	g.showBranchVisualization()

	g.logger.StatusMessage("---------------------------------------------")
	g.logger.StatusMessage("🛑 gitbak terminated at %s", time.Now().Format("2006-01-02 15:04:05"))
}

// showBranchVisualization displays a visual representation of the branch structure
func (g *Gitbak) showBranchVisualization() {
	// Using a background context since this is just for display and not tied to the main app lifecycle
	ctx := context.Background()
	output, err := g.runGitCommandWithOutput(ctx, "log", "--graph", "--oneline", "--decorate", "--all", "--color=always", "-n", "10")
	if err == nil && output != "" {
		g.logger.StatusMessage("")
		g.logger.StatusMessage("🔍 Branch visualization (last 10 commits):")
		g.logger.StatusMessage("---------------------------------------------")
		g.logger.StatusMessage("%s", output)
	}
}

// Git operations

// getCurrentBranch returns the name of the current git branch.
func (g *Gitbak) getCurrentBranch(ctx context.Context) (string, error) {
	output, err := g.runGitCommandWithOutput(ctx, "branch", "--show-current")
	if err != nil {
		return "unknown", err
	}
	return strings.TrimSpace(output), nil
}

// hasUncommittedChanges returns true if the repository contains changes
// that have not been committed yet.
func (g *Gitbak) hasUncommittedChanges(ctx context.Context) (bool, error) {
	output, err := g.runGitCommandWithOutput(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// branchExists checks if a branch with the given name exists.
func (g *Gitbak) branchExists(ctx context.Context, branchName string) (bool, error) {
	_, err := g.runGitCommandWithOutput(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if gitbakErrors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Exit code 1 is the expected "branch not found" case
		return false, nil
	}
	// Unexpected error – bubble up.
	return false, err
}

// findHighestCommitNumber parses git log to find the highest sequential commit
// number used with the configured commit prefix.
func (g *Gitbak) findHighestCommitNumber(ctx context.Context) (int, error) {
	escapedPrefix := regexp.QuoteMeta(g.config.CommitPrefix)

	output, err := g.runGitCommandWithOutput(ctx, "log", "--pretty=format:%s")
	if err != nil {
		return 0, err
	}

	pattern := fmt.Sprintf("%s #([0-9]+)", escapedPrefix)
	re := regexp.MustCompile(pattern)

	highestNum := 0
	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			num, err := strconv.Atoi(matches[1])
			if err == nil && num > highestNum {
				highestNum = num
			}
		}
	}

	return highestNum, nil
}

// runGitCommand executes a git command in the repository directory with context.
func (g *Gitbak) runGitCommand(ctx context.Context, args ...string) error {
	allArgs := append([]string{"-C", g.config.RepoPath}, args...)
	return g.executor.ExecuteWithContext(ctx, "git", allArgs...)
}

// runGitCommandWithOutput executes a git command and returns its output with context.
func (g *Gitbak) runGitCommandWithOutput(ctx context.Context, args ...string) (string, error) {
	allArgs := append([]string{"-C", g.config.RepoPath}, args...)
	return g.executor.ExecuteWithContextAndOutput(ctx, "git", allArgs...)
}

// promptForCommit asks if the user wants to commit changes before starting.
func (g *Gitbak) promptForCommit() bool {
	return g.interactor.PromptYesNo("Would you like to commit them before creating the gitbak branch?")
}

// promptYesNo presents a yes/no question to the user and returns their response.
func (g *Gitbak) promptYesNo(question string) bool {
	return g.interactor.PromptYesNo(question)
}
