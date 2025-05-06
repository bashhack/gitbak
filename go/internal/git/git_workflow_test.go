package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestCompleteGitbakWorkflow tests the full end-to-end workflow
func TestCompleteGitbakWorkflow(t *testing.T) {
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-workflow-test.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("Phase 1: Initialize with new branch", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-workflow-test",
				CommitPrefix:    "[gitbak-workflow] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err := runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to initialize GitBak: %v", err)
		}

		currentBranch, err := gb.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		if currentBranch != "gitbak-workflow-test" {
			t.Errorf("Expected to be on branch 'gitbak-workflow-test', got '%s'", currentBranch)
		}

		if gb.commitsCount != 0 {
			t.Errorf("Expected 0 commits, got %d", gb.commitsCount)
		}
	})

	t.Run("Phase 2: Add various types of changes", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-workflow-test",
				CommitPrefix:    "[gitbak-workflow] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		newFile := filepath.Join(repoPath, "new-workflow-file.txt")
		if err := os.WriteFile(newFile, []byte("New workflow file content"), 0644); err != nil {
			t.Fatalf("Failed to create new file: %v", err)
		}

		subDir := filepath.Join(repoPath, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		subDirFile := filepath.Join(subDir, "subdir-file.txt")
		if err := os.WriteFile(subDirFile, []byte("File in subdirectory"), 0644); err != nil {
			t.Fatalf("Failed to create file in subdirectory: %v", err)
		}

		initialFile := filepath.Join(repoPath, "initial.txt")
		if err := os.WriteFile(initialFile, []byte("Modified initial content"), 0644); err != nil {
			t.Fatalf("Failed to modify initial file: %v", err)
		}

		err := runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run GitBak with changes: %v", err)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected 1 commit, got %d", gb.commitsCount)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected working directory to be clean after commit, but found changes")
		}

		output, err := gb.runGitCommandWithOutput("show", "--name-status", "--pretty=format:%s")
		if err != nil {
			t.Fatalf("Failed to get commit details: %v", err)
		}

		filesChanged := []string{
			"new-workflow-file.txt",
			"subdir/subdir-file.txt",
			"initial.txt",
		}

		for _, file := range filesChanged {
			if !strings.Contains(output, file) {
				t.Errorf("Expected commit to include %s, but it wasn't found in output: %s", file, output)
			}
		}
	})

	t.Run("Phase 3: Continue session with existing commits", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-workflow-test",
				CommitPrefix:    "[gitbak-workflow] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: true,
				NonInteractive:  true,
			},
			log,
		)

		nextFile := filepath.Join(repoPath, "continue-session-file.txt")
		if err := os.WriteFile(nextFile, []byte("File added during continue session"), 0644); err != nil {
			t.Fatalf("Failed to create continue session file: %v", err)
		}

		err := runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run GitBak in continue mode: %v", err)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected 1 additional commit, got %d", gb.commitsCount)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected working directory to be clean after commit, but found changes")
		}

		output, err := gb.runGitCommandWithOutput("log", "-1", "--pretty=%s")
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}

		if !strings.Contains(output, "#2") {
			t.Errorf("Expected commit message to contain '#2', got: %s", output)
		}
	})

	// Tests that gitbak can handle a scenario where its target branch has been deleted
	// by creating a new branch when the specified branch no longer exists.
	// It verifies that gitbak:
	// 1. Successfully creates a new branch with the specified name
	// 2. Can continue operation normally after branch recreation
	t.Run("Phase 4: Recovery from deleted branches", func(t *testing.T) {
		gitCmd := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "",
				CommitPrefix:    "",
				CreateBranch:    false,
				Verbose:         false,
				ShowNoChanges:   false,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)
		originalBranch, err := gitCmd.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get original branch: %v", err)
		}

		tempBranch := "temp-workflow-branch"
		err = gitCmd.runGitCommand("checkout", "-b", tempBranch)
		if err != nil {
			t.Fatalf("Failed to create temporary branch: %v", err)
		}

		deletedBranchName := "branch-to-be-deleted"
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      deletedBranchName,
				CommitPrefix:    "[gitbak-recovery] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to create branch for deletion test: %v", err)
		}

		recoveryFile := filepath.Join(repoPath, "recovery-file.txt")
		if err := os.WriteFile(recoveryFile, []byte("Recovery test content"), 0644); err != nil {
			t.Fatalf("Failed to create recovery file: %v", err)
		}

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to commit recovery file: %v", err)
		}

		err = gitCmd.runGitCommand("checkout", originalBranch)
		if err != nil {
			t.Fatalf("Failed to switch back to original branch: %v", err)
		}

		err = gitCmd.runGitCommand("branch", "-D", deletedBranchName)
		if err != nil {
			t.Fatalf("Failed to delete branch: %v", err)
		}

		exists, err := gitCmd.branchExists(deletedBranchName)
		if err != nil {
			t.Fatalf("Failed to check if branch exists: %v", err)
		}
		if exists {
			t.Errorf("Branch should not exist after deletion")
		}

		// Now run GitBak again with the deleted branch name - it should recreate
		gb2 := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      deletedBranchName,
				CommitPrefix:    "[gitbak-recovery] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		err = runSingleIteration(gb2)
		if err != nil {
			t.Fatalf("Failed to recover from deleted branch: %v", err)
		}

		currentBranch, err := gb2.getCurrentBranch()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}

		if !strings.HasPrefix(currentBranch, deletedBranchName) {
			t.Errorf("Expected to be on a branch starting with '%s', got '%s'", deletedBranchName, currentBranch)
		}
	})

	t.Run("Phase 5: Summary information", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-summary-test",
				CommitPrefix:    "[gitbak-summary] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		gb.originalBranch = "main"
		gb.config.BranchName = "gitbak-summary-test"
		gb.commitsCount = 5
		gb.startTime = time.Now().Add(-2 * time.Hour)

		gb.PrintSummary()
		// No real assertions here, just making sure it executes without errors
	})
}

// TestGitbakWithDifferentRepoStates tests how GitBak handles various repository
// states that can occur in real-world usage
func TestGitbakWithDifferentRepoStates(t *testing.T) {
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-state-test.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("Clean repository", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-clean-test",
				CommitPrefix:    "[gitbak-clean] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected clean repository, but found changes")
		}

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run GitBak on clean repo: %v", err)
		}

		if gb.commitsCount != 0 {
			t.Errorf("Expected 0 commits on clean repo, got %d", gb.commitsCount)
		}
	})

	t.Run("Repository with untracked files", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-untracked-test",
				CommitPrefix:    "[gitbak-untracked] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		for i := 1; i <= 3; i++ {
			untrackedFile := filepath.Join(repoPath, fmt.Sprintf("untracked-file-%d.txt", i))
			if err := os.WriteFile(untrackedFile, []byte(fmt.Sprintf("Untracked file %d", i)), 0644); err != nil {
				t.Fatalf("Failed to create untracked file: %v", err)
			}
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if !hasChanges {
			t.Errorf("Expected changes from untracked files, but found none")
		}

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run GitBak with untracked files: %v", err)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected 1 commit for untracked files, got %d", gb.commitsCount)
		}

		hasChanges, err = gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected working directory to be clean after commit, but found changes")
		}
	})

	t.Run("Repository with mixed changes", func(t *testing.T) {
		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 5,
				BranchName:      "gitbak-mixed-test",
				CommitPrefix:    "[gitbak-mixed] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		baselineFile := filepath.Join(repoPath, "baseline.txt")
		if err := os.WriteFile(baselineFile, []byte("Baseline content"), 0644); err != nil {
			t.Fatalf("Failed to create baseline file: %v", err)
		}

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to commit baseline file: %v", err)
		}

		if err := os.WriteFile(baselineFile, []byte("Modified baseline content"), 0644); err != nil {
			t.Fatalf("Failed to modify baseline file: %v", err)
		}

		newFile := filepath.Join(repoPath, "new-mixed.txt")
		if err := os.WriteFile(newFile, []byte("New file in mixed test"), 0644); err != nil {
			t.Fatalf("Failed to create new file: %v", err)
		}

		toDeleteFile := filepath.Join(repoPath, "to-delete.txt")
		if err := os.WriteFile(toDeleteFile, []byte("File to be deleted"), 0644); err != nil {
			t.Fatalf("Failed to create file for deletion: %v", err)
		}

		err = gb.runGitCommand("add", toDeleteFile)
		if err != nil {
			t.Fatalf("Failed to stage file for deletion: %v", err)
		}
		err = gb.runGitCommand("commit", "-m", "Add file that will be deleted")
		if err != nil {
			t.Fatalf("Failed to commit file for deletion: %v", err)
		}

		if err := os.Remove(toDeleteFile); err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		gb.commitsCount = 0

		statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}

		if !strings.Contains(statusOutput, "M ") ||
			!strings.Contains(statusOutput, "?? ") ||
			!strings.Contains(statusOutput, " D ") {
			t.Errorf("Expected mix of modified, added, and deleted files, got: %s", statusOutput)
		}

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run GitBak with mixed changes: %v", err)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected 1 commit for mixed changes, got %d", gb.commitsCount)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected working directory to be clean after commit, but found changes")
		}
	})
}

// TestGitChangeDetectionWorkflow tests the complete workflow of detecting
// different types of file changes and handling them appropriately
func TestGitChangeDetectionWorkflow(t *testing.T) {
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test.log")
	log := logger.New(true, tempLogFile, true)

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 5,
			BranchName:      "gitbak-test-branch",
			CommitPrefix:    "[gitbak-test] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	t.Run("Modified file", func(t *testing.T) {
		initialFile := filepath.Join(repoPath, "modified-test.txt")
		if err := os.WriteFile(initialFile, []byte("Initial content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if err := gb.runGitCommand("add", "modified-test.txt"); err != nil {
			t.Fatalf("Failed to add test file: %v", err)
		}
		if err := gb.runGitCommand("commit", "-m", "Initial commit for modified file test"); err != nil {
			t.Fatalf("Failed to commit test file: %v", err)
		}

		if err := os.WriteFile(initialFile, []byte("Modified content"), 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if !hasChanges {
			t.Errorf("Expected to detect modified file, but no changes were found")
		}

		statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		if !strings.Contains(statusOutput, "M modified-test.txt") {
			t.Errorf("Expected modified file in status, got: %s", statusOutput)
		}
	})

	t.Run("New file", func(t *testing.T) {
		newFile := filepath.Join(repoPath, "new-file.txt")
		if err := os.WriteFile(newFile, []byte("New file content"), 0644); err != nil {
			t.Fatalf("Failed to create new file: %v", err)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if !hasChanges {
			t.Errorf("Expected to detect new file, but no changes were found")
		}

		statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		if !strings.Contains(statusOutput, "?? new-file.txt") {
			t.Errorf("Expected new file in status, got: %s", statusOutput)
		}
	})

	t.Run("Deleted file", func(t *testing.T) {
		deleteFile := filepath.Join(repoPath, "to-be-deleted.txt")
		if err := os.WriteFile(deleteFile, []byte("File to be deleted"), 0644); err != nil {
			t.Fatalf("Failed to create file to delete: %v", err)
		}

		if err := gb.runGitCommand("add", "to-be-deleted.txt"); err != nil {
			t.Fatalf("Failed to add file to delete: %v", err)
		}
		if err := gb.runGitCommand("commit", "-m", "Add file that will be deleted"); err != nil {
			t.Fatalf("Failed to commit file to delete: %v", err)
		}

		if err := os.Remove(deleteFile); err != nil {
			t.Fatalf("Failed to delete test file: %v", err)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if !hasChanges {
			t.Errorf("Expected to detect deleted file, but no changes were found")
		}

		statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		if !strings.Contains(statusOutput, " D to-be-deleted.txt") {
			t.Errorf("Expected deleted file in status, got: %s", statusOutput)
		}
	})

	t.Run("Binary file", func(t *testing.T) {
		binaryFile := filepath.Join(repoPath, "binary-file.bin")
		binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
			t.Fatalf("Failed to create binary file: %v", err)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if !hasChanges {
			t.Errorf("Expected to detect binary file, but no changes were found")
		}

		statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		if !strings.Contains(statusOutput, "?? binary-file.bin") {
			t.Errorf("Expected binary file in status, got: %s", statusOutput)
		}
	})

	t.Run("Create commit", func(t *testing.T) {
		gb.commitsCount = 0

		err := runSingleIteration(gb)
		if err != nil {
			t.Fatalf("Failed to run single iteration: %v", err)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
		}

		hasChanges, err := gb.hasUncommittedChanges()
		if err != nil {
			t.Fatalf("Failed to check for uncommitted changes: %v", err)
		}
		if hasChanges {
			t.Errorf("Expected working directory to be clean after commit, but found changes")

			statusOutput, err := gb.runGitCommandWithOutput("status", "--porcelain")
			if err != nil {
				t.Fatalf("Failed to get git status: %v", err)
			}
			t.Logf("Unexpected status after commit: %s", statusOutput)
		}

		commitMsg, err := gb.runGitCommandWithOutput("log", "-1", "--pretty=%s")
		if err != nil {
			t.Fatalf("Failed to get commit message: %v", err)
		}
		if !strings.Contains(commitMsg, "[gitbak-test] Commit #1") {
			t.Errorf("Expected commit message to contain '[gitbak-test] Commit #1', got: %s", commitMsg)
		}
	})
}
