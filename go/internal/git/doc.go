// Package git provides Git operations for the gitbak application.
//
// This package abstracts Git commands and operations used by gitbak,
// handling repository interaction, branch management, commit creation, and error recovery.
// It implements automatic Git operations with built-in retry mechanisms and
// tracking for sequential commit numbering.
//
// # Core Components
//
// - Gitbak: Main type that manages a Git repository and performs automatic commits
// - CommandExecutor: Interface for executing Git commands
// - UserInteractor: Interface for user interaction during Git operations
//
// # Features
//
// - Automatic Git operations with configurable intervals
// - Sequential commit numbering with the ability to continue from a previous session
// - Branch creation and management
// - Error handling with configurable retry logic
// - Clean session termination with statistics
//
// # Usage
//
// Basic usage pattern:
//
//	config := git.GitbakConfig{
//	    RepoPath:        "/path/to/repo",
//	    IntervalMinutes: 5,
//	    BranchName:      "gitbak-session",
//	    CommitPrefix:    "[gitbak]",
//	    CreateBranch:    true,
//	    ContinueSession: false,
//	    Verbose:         true,
//	    ShowNoChanges:   false,
//	    MaxRetries:      3,
//	}
//
//	gitbak, err := git.NewGitbak(config, logger)
//	if err != nil {
//	    // Handle error
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	go func() {
//	    // Cancel context when termination is requested
//	    // (e.g., signal handling, user input, etc.)
//	}()
//
//	if err := gitbak.Run(ctx); err != nil {
//	    // Handle error
//	}
//
//	// Show session summary
//	gitbak.PrintSummary()
//
// # Error Handling
//
// The package implements sophisticated error recovery with configurable retry mechanisms:
//
// - Consecutive identical errors are counted and compared against MaxRetries
// - When errors change or successful operations occur, the error counter resets
// - Setting MaxRetries to 0 makes the system retry indefinitely
//
// # Implementation Notes
//
// The package uses the command-line Git executable rather than a Go Git library.
// This ensures compatibility with all Git features and repository configurations.
//
// # Concurrency Model
//
// The gitbak instance is not thread-safe and should be accessed from a single goroutine.
// Different gitbak instances can safely operate on different repositories concurrently,
// of course - allowing users to run multiple gitbak instances on different repositories
// without a concern.
//
// # Dependencies
//
// This package requires:
//
// - A functional Git installation in the system PATH
// - A valid Git repository at the configured path
// - Write permissions for the repository
package git
