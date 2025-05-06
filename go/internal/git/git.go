package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bashhack/gitbak/internal/common"
	"github.com/bashhack/gitbak/internal/errors"
)

// GitbakConfig contains configuration for a gitbak instance
type GitbakConfig struct {
	// Repository path
	RepoPath string

	// Commit settings
	IntervalMinutes int
	BranchName      string
	CommitPrefix    string
	CreateBranch    bool
	ContinueSession bool

	// Output configuration
	Verbose       bool
	ShowNoChanges bool

	// When true, disables prompts and uses defaults
	NonInteractive bool
}

// Gitbak monitors and auto-commits changes to a git repository
type Gitbak struct {
	config         GitbakConfig
	logger         Logger
	executor       CommandExecutor
	interactor     UserInteractor
	commitsCount   int
	startTime      time.Time
	originalBranch string
}

// Logger alias to common.Logger
type Logger = common.Logger

// NewGitbak creates a new gitbak instance with default dependencies
func NewGitbak(config GitbakConfig, logger Logger) *Gitbak {
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
	logger Logger,
	executor CommandExecutor,
	interactor UserInteractor,
) *Gitbak {
	return &Gitbak{
		config:       config,
		logger:       logger,
		executor:     executor,
		interactor:   interactor,
		commitsCount: 0,
		startTime:    time.Now(),
	}
}

// IsRepository checks if the given path is a git repository
func IsRepository(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	executor := NewExecExecutor()
	return executor.Execute(cmd) == nil
}

// Run starts the gitbak process with the given context for cancellation
func (g *Gitbak) Run(ctx context.Context) error {
	g.startTime = time.Now()

	if err := g.initialize(); err != nil {
		return err
	}

	return g.monitoringLoop(ctx)
}

// initialize prepares the gitbak session by detecting the original branch
// and configuring the appropriate session mode
func (g *Gitbak) initialize() error {
	var err error

	g.originalBranch, err = g.getCurrentBranch()
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		// Check if it's already a GitError
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.Wrap(errors.ErrGitOperationFailed, "failed to get current branch")
	}
	g.logger.Info("Starting gitbak on branch: %s", g.originalBranch)

	if g.config.ContinueSession {
		if err := g.setupContinueSession(); err != nil {
			return err
		}
	} else if g.config.CreateBranch {
		if err := g.setupNewBranchSession(); err != nil {
			return err
		}
	} else {
		g.setupCurrentBranchSession()
	}

	g.displayStartupInfo()
	return nil
}

// setupContinueSession configures gitbak for continuing a previous session
func (g *Gitbak) setupContinueSession() error {
	g.config.CreateBranch = false
	g.logger.StatusMessage("🔄 Continuing gitbak session on branch: %s", g.originalBranch)

	highestNum, err := g.findHighestCommitNumber()
	if err != nil {
		g.logger.Warning("Failed to find highest commit number: %v", err)
		g.logger.InfoToUser("No previous commits found with prefix '%s' - starting from commit #1", g.config.CommitPrefix)
	} else if highestNum > 0 {
		g.logger.InfoToUser("Found previous commits - starting from commit #%d", highestNum+1)
	} else {
		g.logger.InfoToUser("No previous commits found with prefix '%s' - starting from commit #1", g.config.CommitPrefix)
	}

	return nil
}

// setupNewBranchSession creates and switches to a new branch
func (g *Gitbak) setupNewBranchSession() error {
	hasChanges, err := g.hasUncommittedChanges()
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.NewGitError("status", nil, errors.Wrap(errors.ErrGitOperationFailed, "failed to check for uncommitted changes"), "")
	}

	if hasChanges {
		g.logger.WarningToUser("You have uncommitted changes.")
		if err := g.handleUncommittedChanges(); err != nil {
			return err
		}
	}

	if err := g.handleBranchName(); err != nil {
		return err
	}

	if err := g.createAndCheckoutBranch(); err != nil {
		return err
	}

	return nil
}

// setupCurrentBranchSession uses the current branch for commits
func (g *Gitbak) setupCurrentBranchSession() {
	currentBranch, err := g.getCurrentBranch()
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		g.logger.StatusMessage("🌿 Using current branch: unknown")
		return
	}
	g.logger.StatusMessage("🌿 Using current branch: %s", currentBranch)
}

// handleUncommittedChanges prompts the user about uncommitted changes
func (g *Gitbak) handleUncommittedChanges() error {
	shouldCommit := g.promptForCommit()

	if shouldCommit {
		addArgs := []string{"."}
		if err := g.runGitCommand("add", "."); err != nil {
			return errors.NewGitError("add", addArgs, err, "failed to stage changes")
		}

		commitMsg := "Manual commit before starting gitbak session"
		commitArgs := []string{"-m", commitMsg}
		if err := g.runGitCommand("commit", "-m", commitMsg); err != nil {
			return errors.NewGitError("commit", commitArgs, err, "failed to create initial commit")
		}

		g.logger.Success("Created initial commit")
	}

	return nil
}

// handleBranchName manages branch name conflicts
func (g *Gitbak) handleBranchName() error {
	branchExists, err := g.branchExists(g.config.BranchName)
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.NewGitError("show-ref", []string{g.config.BranchName},
			errors.Wrap(errors.ErrGitOperationFailed, "failed to check if branch exists"), "")
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
func (g *Gitbak) createAndCheckoutBranch() error {
	args := []string{"-b", g.config.BranchName}
	err := g.runGitCommand("checkout", "-b", g.config.BranchName)
	if err != nil {
		return errors.NewGitError("checkout", args, err, "failed to create new branch")
	}

	g.logger.StatusMessage("🌿 Created and switched to new branch: %s", g.config.BranchName)
	return nil
}

// displayStartupInfo outputs the active configuration to the user
func (g *Gitbak) displayStartupInfo() {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	g.logger.StatusMessage("🔄 gitbak started at %s", timestamp)
	g.logger.StatusMessage("📂 Repository: %s", g.config.RepoPath)
	g.logger.StatusMessage("⏱️  Interval: %d minutes", g.config.IntervalMinutes)
	g.logger.StatusMessage("📝 Commit prefix: %s", g.config.CommitPrefix)
	g.logger.StatusMessage("🔊 Verbose mode: %t", g.config.Verbose)
	g.logger.StatusMessage("🔔 Show no-changes messages: %t", g.config.ShowNoChanges)
	g.logger.StatusMessage("❓ Press Ctrl+C to stop and view session summary")
}

// monitoringLoop periodically checks for changes and creates commits.
// It runs until the context is canceled or an unrecoverable error occurs.
func (g *Gitbak) monitoringLoop(ctx context.Context) error {
	commitCounter := 1

	if g.config.ContinueSession {
		highestNum, err := g.findHighestCommitNumber()
		if err == nil && highestNum > 0 {
			commitCounter = highestNum + 1
		}
	}

	ticker := time.NewTicker(time.Duration(g.config.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Received cancellation signal, shutting down gracefully...")
			return ctx.Err()

		case <-ticker.C:
			if err := g.checkAndCommitChanges(commitCounter); err != nil {
				g.logger.Error("Error in commit cycle: %v", err)
				g.logger.WarningToUser("Error occurred: %v", err)
				g.logger.StatusMessage("Will retry in %d minutes.", g.config.IntervalMinutes)
			} else if g.commitsCount > 0 {
				commitCounter++
			}
		}
	}
}

// checkAndCommitChanges checks for uncommitted changes and creates a commit if found.
func (g *Gitbak) checkAndCommitChanges(commitCounter int) error {
	hasChanges, err := g.hasUncommittedChanges()
	if err != nil {
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.NewGitError("status", nil,
			errors.Wrap(errors.ErrGitOperationFailed, "failed to check git status"), "")
	}

	if hasChanges {
		return g.createCommit(commitCounter)
	} else if g.config.ShowNoChanges && g.config.Verbose {
		g.logger.InfoToUser("No changes to commit at %s", time.Now().Format("15:04:05"))
		g.logger.Info("No changes to commit detected")
	}

	return nil
}

// createCommit stages all changes and creates a commit with the configured prefix.
func (g *Gitbak) createCommit(commitCounter int) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	addArgs := []string{"."}
	err := g.runGitCommand("add", ".")
	if err != nil {
		g.logger.Warning("Failed to stage changes: %v", err)
		g.logger.WarningToUser("Failed to stage changes: %v", err)
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.NewGitError("add", addArgs,
			errors.Wrap(errors.ErrGitOperationFailed, "failed to stage changes"), "")
	}

	commitMsg := fmt.Sprintf("%s #%d - %s", g.config.CommitPrefix, commitCounter, timestamp)
	commitArgs := []string{"-m", commitMsg}
	err = g.runGitCommand("commit", "-m", commitMsg)
	if err != nil {
		g.logger.Warning("Failed to create commit: %v", err)
		g.logger.WarningToUser("Failed to create commit: %v", err)
		// If it's already a GitError or already has ErrGitOperationFailed, just return it
		if errors.Is(err, errors.ErrGitOperationFailed) {
			return err
		}
		return errors.NewGitError("commit", commitArgs,
			errors.Wrap(errors.ErrGitOperationFailed, "failed to create commit"), "")
	}

	g.logger.Success("Commit #%d created at %s", commitCounter, timestamp)
	g.logger.Info("Successfully created commit #%d", commitCounter)
	g.commitsCount++

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
	output, err := g.runGitCommandWithOutput("log", "--graph", "--oneline", "--decorate", "--all", "--color=always", "-n", "10")
	if err == nil && output != "" {
		g.logger.StatusMessage("")
		g.logger.StatusMessage("🔍 Branch visualization (last 10 commits):")
		g.logger.StatusMessage("---------------------------------------------")
		g.logger.StatusMessage("%s", output)
	}
}

// Git operations

// getCurrentBranch returns the name of the current git branch.
func (g *Gitbak) getCurrentBranch() (string, error) {
	output, err := g.runGitCommandWithOutput("branch", "--show-current")
	if err != nil {
		return "unknown", err
	}
	return strings.TrimSpace(output), nil
}

// hasUncommittedChanges returns true if the repository contains changes
// that have not been committed yet.
func (g *Gitbak) hasUncommittedChanges() (bool, error) {
	output, err := g.runGitCommandWithOutput("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// branchExists checks if a branch with the given name exists.
func (g *Gitbak) branchExists(branchName string) (bool, error) {
	_, err := g.runGitCommandWithOutput("show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err != nil {
		// Command returns non-zero if branch doesn't exist
		return false, nil
	}
	return true, nil
}

// findHighestCommitNumber parses git log to find the highest sequential commit
// number used with the configured commit prefix.
func (g *Gitbak) findHighestCommitNumber() (int, error) {
	escapedPrefix := regexp.QuoteMeta(g.config.CommitPrefix)

	output, err := g.runGitCommandWithOutput("log", "--pretty=format:%s")
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

// runGitCommand executes a git command in the repository directory.
func (g *Gitbak) runGitCommand(args ...string) error {
	baseArgs := []string{"-C", g.config.RepoPath}
	cmd := exec.Command("git", append(baseArgs, args...)...)
	cmd.Dir = g.config.RepoPath
	return g.executor.Execute(cmd)
}

// runGitCommandWithOutput executes a git command and returns its output.
func (g *Gitbak) runGitCommandWithOutput(args ...string) (string, error) {
	baseArgs := []string{"-C", g.config.RepoPath}
	cmd := exec.Command("git", append(baseArgs, args...)...)
	cmd.Dir = g.config.RepoPath
	return g.executor.ExecuteWithOutput(cmd)
}

// User interaction

// promptForCommit asks if the user wants to commit changes before starting.
func (g *Gitbak) promptForCommit() bool {
	return g.interactor.PromptYesNo("Would you like to commit them before creating the gitbak branch?")
}

// promptYesNo presents a yes/no question to the user and returns their response.
func (g *Gitbak) promptYesNo(question string) bool {
	return g.interactor.PromptYesNo(question)
}
