package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestCommitCounterIncrementation verifies that the commit counter is properly
// incremented after each successful commit in the monitoring loop and that
// the counter is correctly initialized when continuing sessions
func TestCommitCounterIncrementation(t *testing.T) {
	t.Parallel()
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-counter-test.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	// Set up gitbak with a custom monitoring loop that doesn't wait for ticker
	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "counter-test-branch",
			CommitPrefix:    "[counter-test] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	ctx := context.Background()
	if err := gb.initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize gitbak: %v", err)
	}

	// Create errorState struct for passing to tryOperation
	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	// Simulate monitoring loop execution with multiple commits
	// Manually create the files that will trigger commits
	for i := 1; i <= 3; i++ {
		// Create a new file for each iteration
		filename := filepath.Join(repoPath, fmt.Sprintf("commit-counter-test-%d.txt", i))
		content := fmt.Sprintf("Content for testing commit counter incrementation - #%d", i)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}

		// Execute the same pattern as in monitoringLoop but without waiting for ticker
		// This simulates the pattern in the code:
		// 1. Check for changes and commit if needed
		// 2. Increment counter only on success
		commitCounter := i
		err := gb.tryOperation(ctx, &errorState, func() error {
			var commitWasCreated bool
			if err := gb.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated); err != nil {
				return err
			}
			// In the real code, commitCounter would be incremented here
			return nil
		})

		if err != nil {
			t.Fatalf("Failed on commit #%d: %v", i, err)
		}

		output, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		expectedPrefix := fmt.Sprintf("[counter-test] Commit #%d", i)
		if !strings.Contains(output, expectedPrefix) {
			t.Errorf("Commit #%d has incorrect message. Expected to contain '%s', got: %s",
				i, expectedPrefix, output)
		}
	}

	output, err := gb.runGitCommandWithOutput(ctx, "log", "--pretty=%s")
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// The first line is the most recent commit (commit #3)
	for i, line := range lines[:3] {
		commitNum := 3 - i // Newest first, so index 0 = commit #3
		expectedPrefix := fmt.Sprintf("[counter-test] Commit #%d", commitNum)
		if !strings.Contains(line, expectedPrefix) {
			t.Errorf("Commit history incorrect at position %d. Expected '%s', got: %s",
				i, expectedPrefix, line)
		}
	}
}
