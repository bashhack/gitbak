package git

import (
	"context"
	"github.com/bashhack/gitbak/internal/errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestRunWithContext tests the Run method with context cancellation
func TestRunWithContext(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	tempLogDir, err := os.MkdirTemp("", "gitbak-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for logs: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempLogDir); err != nil {
			t.Logf("Failed to remove temporary log directory: %v", err)
		}
	}()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-context.log")
	log := logger.New(true, tempLogFile, true)

	// Create a test file to ensure we have changes to commit
	testFile := filepath.Join(repoPath, "test-run-context.txt")
	err = os.WriteFile(testFile, []byte("Test content for Run method with context"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("Context cancellation", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1, // 1 minute interval (will be quickly canceled)
				BranchName:      "gitbak-context-branch",
				CommitPrefix:    "[gitbak-context] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		// Create a context that will be canceled after a short delay
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// Run the gitbak instance in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- gb.Run(ctx)
		}()

		// Wait for the context to be canceled and gitbak to return
		select {
		case err := <-errChan:
			// Verify that the error is a context error (deadline exceeded or canceled)
			if !errors.Is(err, ctx.Err()) {
				t.Errorf("Expected context error %v, got %v", ctx.Err(), err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("gitbak.Run did not return after context cancellation within 5 seconds")
		}

		// Verify that a new branch was created
		branchExists, err := gb.branchExists("gitbak-context-branch")
		if err != nil {
			t.Fatalf("Failed to check if branch exists: %v", err)
		}
		if !branchExists {
			t.Errorf("Expected branch 'gitbak-context-branch' to be created")
		}

		currentBranch, err := gb.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		if currentBranch != "gitbak-context-branch" {
			t.Errorf("Expected to be on branch 'gitbak-context-branch', but got '%s'", currentBranch)
		}

		// At minimum, initialize should have completed with context cancellation
		// If the interval is small enough and cancellation slows enough, a commit might have occurred.
		// We'll check if changes were staged at least
		output, err := gb.runGitCommandWithOutput("diff", "--cached", "--name-only")
		if err != nil {
			t.Fatalf("Failed to get staged changes: %v", err)
		}
		if !strings.Contains(output, "test-run-context.txt") {
			t.Logf("Note: No changes were staged before context cancellation")
		}
	})
}

// TestInitializeAndSetupBranch tests the initialize and branch setup methods
func TestInitializeAndSetupBranch(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	tempLogDir, err := os.MkdirTemp("", "gitbak-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for logs: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempLogDir); err != nil {
			t.Logf("Failed to remove temporary log directory: %v", err)
		}
	}()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-init.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("initialize with create branch", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      "gitbak-init-branch",
				CommitPrefix:    "[gitbak-init] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := gb.initialize()
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		currentBranch, err := gb.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		if currentBranch != "gitbak-init-branch" {
			t.Errorf("Expected to be on branch 'gitbak-init-branch', but got '%s'", currentBranch)
		}

		if gb.originalBranch == "" {
			t.Errorf("Expected originalBranch to be set, but it was empty")
		}
	})

	t.Run("initialize with existing branch", func(t *testing.T) {
		existingBranchName := "gitbak-existing-branch"
		gb1 := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      existingBranchName,
				CommitPrefix:    "[gitbak-existing] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := gb1.initialize()
		if err != nil {
			t.Fatalf("initialize failed for setup: %v", err)
		}

		// Now create a second gitbak instance that will use the same branch name
		// This will trigger the branch name conflict handling
		gb2 := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      existingBranchName,
				CommitPrefix:    "[gitbak-existing] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err = gb2.initialize()
		if err != nil {
			t.Fatalf("initialize failed for conflict test: %v", err)
		}

		currentBranch, err := gb2.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}

		if currentBranch == existingBranchName {
			t.Errorf("Expected to be on a branch different from '%s', but got the same branch", existingBranchName)
		}
		if !strings.HasPrefix(currentBranch, existingBranchName+"-") {
			t.Errorf("Expected to be on a branch with prefix '%s-', but got '%s'", existingBranchName, currentBranch)
		}
	})

	t.Run("initialize with continue session", func(t *testing.T) {
		continuePrefix := "[gitbak-continue] Commit"
		continueBranch := "gitbak-continue-branch"

		gb1 := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      continueBranch,
				CommitPrefix:    continuePrefix,
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := gb1.initialize()
		if err != nil {
			t.Fatalf("initialize failed for setup: %v", err)
		}

		testFile := filepath.Join(repoPath, "test-continue.txt")
		err = os.WriteFile(testFile, []byte("Test content for continue mode"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = gb1.runGitCommand("add", "test-continue.txt")
		if err != nil {
			t.Fatalf("Failed to stage test file: %v", err)
		}

		initialCommitMsg := continuePrefix + " #1 - 2023-01-01 12:00:00"
		err = gb1.runGitCommand("commit", "-m", initialCommitMsg)
		if err != nil {
			t.Fatalf("Failed to commit test file: %v", err)
		}

		gb2 := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      continueBranch,
				CommitPrefix:    continuePrefix,
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: true,
				NonInteractive:  true,
			},
			log,
		)

		err = gb2.initialize()
		if err != nil {
			t.Fatalf("initialize failed for continue test: %v", err)
		}

		if gb2.config.CreateBranch {
			t.Errorf("Expected CreateBranch to be false in continue mode")
		}
	})
}

// TestMonitoringLoopWithContext tests the monitoring loop with context cancellation
func TestMonitoringLoopWithContext(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	tempLogDir, err := os.MkdirTemp("", "gitbak-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for logs: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempLogDir); err != nil {
			t.Logf("Failed to remove temporary log directory: %v", err)
		}
	}()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-monitor.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("monitoringLoop with context cancellation", func(t *testing.T) {
		// Create a gitbak instance with a very short interval
		// We want the interval to be short enough that the ticker will trigger before the context is canceled
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1, // 1 minute, but we'll use a shorter context timeout
				BranchName:      "gitbak-monitor-branch",
				CommitPrefix:    "[gitbak-monitor] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		// Initialize first (normally done by Run)
		err := gb.initialize()
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		// Create a test file to ensure we have changes to commit
		testFile := filepath.Join(repoPath, "test-monitor.txt")
		err = os.WriteFile(testFile, []byte("Test content for monitoring loop"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a context with a short timeout to cancel the loop
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		// Start the monitoring loop in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- gb.monitoringLoop(ctx)
		}()

		// Wait for the context to be canceled and monitoringLoop to return
		select {
		case err := <-errChan:
			// Verify that the error is a context error (deadline exceeded or canceled)
			if !errors.Is(err, ctx.Err()) {
				t.Errorf("Expected context error %v, got %v", ctx.Err(), err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("monitoringLoop did not return after context cancellation within 5 seconds")
		}
	})
}

// TestCheckAndCommitChanges tests the checkAndCommitChanges and createCommit methods
func TestCheckAndCommitChanges(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	tempLogDir, err := os.MkdirTemp("", "gitbak-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for logs: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempLogDir); err != nil {
			t.Logf("Failed to remove temporary log directory: %v", err)
		}
	}()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-commit.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("checkAndCommitChanges with changes", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      "gitbak-commit-branch",
				CommitPrefix:    "[gitbak-commit] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := gb.initialize()
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		testFile := filepath.Join(repoPath, "test-commit.txt")
		err = os.WriteFile(testFile, []byte("Test content for checkAndCommitChanges"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = gb.checkAndCommitChanges(1)
		if err != nil {
			t.Fatalf("checkAndCommitChanges failed: %v", err)
		}

		output, err := gb.runGitCommandWithOutput("log", "--grep", "\\[gitbak-commit\\]", "--oneline")
		if err != nil {
			t.Fatalf("Failed to get commit log: %v", err)
		}

		if !strings.Contains(output, "[gitbak-commit]") {
			t.Errorf("Expected to find a commit with prefix '[gitbak-commit]', but got: %s", output)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected commits count to be 1, got %d", gb.commitsCount)
		}
	})

	t.Run("checkAndCommitChanges without changes", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      "gitbak-no-changes-branch",
				CommitPrefix:    "[gitbak-no-changes] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := gb.initialize()
		if err != nil {
			t.Fatalf("initialize failed: %v", err)
		}

		err = gb.checkAndCommitChanges(1)
		if err != nil {
			t.Fatalf("checkAndCommitChanges failed: %v", err)
		}

		if gb.commitsCount != 0 {
			t.Errorf("Expected commits count to be 0, got %d", gb.commitsCount)
		}
	})
}
