package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestGitbakInitializeBranch tests initializing Gitbak with a new branch
func TestGitbakInitializeBranch(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-init-branch.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

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

	ctx := context.Background()
	err := gb.RunSingleIteration(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize GitBak: %v", err)
	}

	currentBranch, err := gb.getCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if currentBranch != "gitbak-workflow-test" {
		t.Errorf("Expected to be on branch 'gitbak-workflow-test', got '%s'", currentBranch)
	}

	if gb.commitsCount != 0 {
		t.Errorf("Expected 0 commits, got %d", gb.commitsCount)
	}
}

// TestGitbakHandleChanges tests Gitbak's ability to detect and commit different types of changes
func TestGitbakHandleChanges(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-handle-changes.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	setupGb := setupTestGitbak(
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

	setupCtx := context.Background()
	if err := setupGb.RunSingleIteration(setupCtx); err != nil {
		t.Fatalf("Failed to set up initial branch: %v", err)
	}

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

	ctx := context.Background()
	err := gb.RunSingleIteration(ctx)
	if err != nil {
		t.Fatalf("Failed to run GitBak with changes: %v", err)
	}

	if gb.commitsCount != 1 {
		t.Errorf("Expected 1 commit, got %d", gb.commitsCount)
	}

	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected working directory to be clean after commit, but found changes")
	}

	output, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:%s")
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
}

// TestGitbakContinueSession tests Gitbak's continue mode which resumes with correct commit numbering
func TestGitbakContinueSession(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-continue.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	branch := "gitbak-continue-branch"
	prefix := "[gitbak-workflow] Commit"

	setupGb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 5,
			BranchName:      branch,
			CommitPrefix:    prefix,
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	setupCtx := context.Background()
	if err := setupGb.RunSingleIteration(setupCtx); err != nil {
		t.Fatalf("Failed to set up branch: %v", err)
	}

	firstFile := filepath.Join(repoPath, "first-file.txt")
	if err := os.WriteFile(firstFile, []byte("First file content"), 0644); err != nil {
		t.Fatalf("Failed to create first file: %v", err)
	}

	if err := setupGb.RunSingleIteration(setupCtx); err != nil {
		t.Fatalf("Failed to make first commit: %v", err)
	}

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 5,
			BranchName:      branch,
			CommitPrefix:    prefix,
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

	ctx := context.Background()
	err := gb.RunSingleIteration(ctx)
	if err != nil {
		t.Fatalf("Failed to run GitBak in continue mode: %v", err)
	}

	// When in continue mode, setupContinueSession initializes g.commitsCount to the highest found commit number
	// After creating a commit, g.commitsCount is incremented
	// At this point, we expect commitsCount to reflect that we've processed 2 commits total
	if gb.commitsCount != 2 {
		t.Errorf("Expected 2 for commitsCount in continue session (1 from setupContinueSession + 1 new), got %d", gb.commitsCount)
	}

	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected working directory to be clean after commit, but found changes")
	}

	output, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}

	if !strings.Contains(output, "#2") {
		t.Errorf("Expected commit message to contain '#2', got: %s", output)
	}
}

// TestGitbakRecoverFromDeletedBranch tests Gitbak's ability to recover from a deleted target branch
// by creating a new branch when the specified branch no longer exists
func TestGitbakRecoverFromDeletedBranch(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-recover-branch.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	gitCmd := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 5,
			BranchName:      "temp-command-branch",
			CommitPrefix:    "[temp] ",
			CreateBranch:    false,
			Verbose:         false,
			ShowNoChanges:   false,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)
	cmdCtx := context.Background()
	originalBranch, err := gitCmd.getCurrentBranch(cmdCtx)
	if err != nil {
		t.Fatalf("Failed to get original branch: %v", err)
	}

	tempBranch := "temp-workflow-branch"
	err = gitCmd.runGitCommand(cmdCtx, "checkout", "-b", tempBranch)
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

	ctx := context.Background()
	if err := gb.RunSingleIteration(ctx); err != nil {
		t.Fatalf("Failed to create branch '%s' for deletion test: %v", deletedBranchName, err)
	}

	recoveryFile := filepath.Join(repoPath, "recovery-file.txt")
	if err := os.WriteFile(recoveryFile, []byte("Recovery test content"), 0644); err != nil {
		t.Fatalf("Failed to create recovery file: %v", err)
	}

	if err := gb.RunSingleIteration(ctx); err != nil {
		t.Fatalf("Failed to commit recovery file: %v", err)
	}

	err = gitCmd.runGitCommand(cmdCtx, "checkout", originalBranch)
	if err != nil {
		t.Fatalf("Failed to switch back to original branch: %v", err)
	}

	err = gitCmd.runGitCommand(cmdCtx, "branch", "-D", deletedBranchName)
	if err != nil {
		t.Fatalf("Failed to delete branch: %v", err)
	}

	exists, err := gitCmd.branchExists(cmdCtx, deletedBranchName)
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

	recoveryCtx := context.Background()
	err = gb2.RunSingleIteration(recoveryCtx)
	if err != nil {
		t.Fatalf("Failed to recover from deleted branch: %v", err)
	}

	checkCtx := context.Background()
	currentBranch, err := gb2.getCurrentBranch(checkCtx)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if !strings.HasPrefix(currentBranch, deletedBranchName) {
		t.Errorf("Expected to be on a branch starting with '%s', got '%s'", deletedBranchName, currentBranch)
	}
}

// TestGitbakPrintSummary tests the generation of session summary information
func TestGitbakPrintSummary(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-summary.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

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

	var stdoutBuf bytes.Buffer
	originalLogger := gb.logger
	testLogger := logger.NewWithOutput(true, "", true, &stdoutBuf, &stdoutBuf)
	gb.logger = testLogger

	gb.PrintSummary()

	gb.logger = originalLogger

	summaryOutput := stdoutBuf.String()
	expectedInfo := []string{
		"gitbak-summary-test",    // Branch name
		"Total commits made: 5",  // Commit count
		"Session duration: 2h",   // Duration
		"Working branch",         // Branch info
		"To merge these changes", // Merge instructions
	}

	for _, info := range expectedInfo {
		if !strings.Contains(summaryOutput, info) {
			t.Errorf("Expected summary to contain '%s', but it wasn't found", info)
		}
	}
}

// TestGitbakCleanRepository tests how GitBak handles a clean repository with no changes
func TestGitbakCleanRepository(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-clean.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

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

	ctx := context.Background()
	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected clean repository, but found changes")
	}

	if err := gb.RunSingleIteration(ctx); err != nil {
		t.Fatalf("Failed to run GitBak on clean repo: %v", err)
	}

	if gb.commitsCount != 0 {
		t.Errorf("Expected 0 commits on clean repo, got %d", gb.commitsCount)
	}
}

// TestGitbakUntrackedFiles tests how GitBak handles a repository with untracked files
func TestGitbakUntrackedFiles(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-untracked.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

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

	ctx := context.Background()
	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected changes from untracked files, but found none")
	}

	if err := gb.RunSingleIteration(ctx); err != nil {
		t.Fatalf("Failed to run GitBak with untracked files: %v", err)
	}

	if gb.commitsCount != 1 {
		t.Errorf("Expected 1 commit for untracked files, got %d", gb.commitsCount)
	}

	hasChanges, err = gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected working directory to be clean after commit, but found changes")
	}
}

// TestGitbakMixedChanges tests how GitBak handles a repository with mixed changes
// (modified, new, and deleted files)
func TestGitbakMixedChanges(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-mixed.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

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

	ctx := context.Background()
	if err := gb.RunSingleIteration(ctx); err != nil {
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

	addCtx := context.Background()
	if err := gb.runGitCommand(addCtx, "add", toDeleteFile); err != nil {
		t.Fatalf("Failed to stage file for deletion: %v", err)
	}

	statusOutput, statusErr := gb.runGitCommandWithOutput(addCtx, "status", "--porcelain")
	if statusErr != nil {
		t.Fatalf("Failed to check git status after staging: %v", statusErr)
	}

	// Format can be either "A " or "A  " depending on git version, so check for contains...
	fileName := filepath.Base(toDeleteFile)
	if !strings.Contains(statusOutput, "A ") || !strings.Contains(statusOutput, fileName) {
		t.Fatalf("File not properly staged, status: %s", statusOutput)
	}

	if err := gb.runGitCommand(addCtx, "commit", "-m", "Add file that will be deleted"); err != nil {
		t.Fatalf("Failed to commit file for deletion: %v", err)
	}

	if err := os.Remove(toDeleteFile); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	gb.commitsCount = 0

	statusCtx := context.Background()
	statusOutput, err := gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}

	if !strings.Contains(statusOutput, "M ") ||
		!strings.Contains(statusOutput, "?? ") ||
		!strings.Contains(statusOutput, " D ") {
		t.Errorf("Expected mix of modified, added, and deleted files, got: %s", statusOutput)
	}

	mixedCtx := context.Background()
	if err := gb.RunSingleIteration(mixedCtx); err != nil {
		t.Fatalf("Failed to run GitBak with mixed changes: %v", err)
	}

	if gb.commitsCount != 1 {
		t.Errorf("Expected 1 commit for mixed changes, got %d", gb.commitsCount)
	}

	cleanCtx := context.Background()
	hasChanges, err := gb.hasUncommittedChanges(cleanCtx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected working directory to be clean after commit, but found changes")
	}
}

// TestGitbakFileChanges tests Gitbak's ability to detect and handle different types
// of file changes (modified, new, deleted, and binary)
func TestGitbakFileChanges(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-file-changes.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 5,
			BranchName:      "gitbak-file-changes",
			CommitPrefix:    "[gitbak-test] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	if err := gb.RunSingleIteration(context.Background()); err != nil {
		t.Fatalf("Failed to initialize branch: %v", err)
	}

	initialFile := filepath.Join(repoPath, "modified-test.txt")
	if err := os.WriteFile(initialFile, []byte("Initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	addCtx := context.Background()
	if err := gb.runGitCommand(addCtx, "add", "modified-test.txt"); err != nil {
		t.Fatalf("Failed to add test file: %v", err)
	}
	if err := gb.runGitCommand(addCtx, "commit", "-m", "Initial commit for modified file test"); err != nil {
		t.Fatalf("Failed to commit test file: %v", err)
	}

	if err := os.WriteFile(initialFile, []byte("Modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	ctx := context.Background()
	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected to detect modified file, but no changes were found")
	}

	statusCtx := context.Background()
	statusOutput, err := gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !strings.Contains(statusOutput, "M modified-test.txt") {
		t.Errorf("Expected modified file in status, got: %s", statusOutput)
	}

	newFile := filepath.Join(repoPath, "new-file.txt")
	if err := os.WriteFile(newFile, []byte("New file content"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	hasChanges, err = gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected to detect new file, but no changes were found")
	}

	statusOutput, err = gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !strings.Contains(statusOutput, "?? new-file.txt") {
		t.Errorf("Expected new file in status, got: %s", statusOutput)
	}

	deleteFile := filepath.Join(repoPath, "to-be-deleted.txt")
	if err := os.WriteFile(deleteFile, []byte("File to be deleted"), 0644); err != nil {
		t.Fatalf("Failed to create file to delete: %v", err)
	}

	delCtx := context.Background()
	if err := gb.runGitCommand(delCtx, "add", "to-be-deleted.txt"); err != nil {
		t.Fatalf("Failed to add file to delete: %v", err)
	}
	if err := gb.runGitCommand(delCtx, "commit", "-m", "Add file that will be deleted"); err != nil {
		t.Fatalf("Failed to commit file to delete: %v", err)
	}

	if err := os.Remove(deleteFile); err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	hasChanges, err = gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected to detect deleted file, but no changes were found")
	}

	statusOutput, err = gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !strings.Contains(statusOutput, " D to-be-deleted.txt") {
		t.Errorf("Expected deleted file in status, got: %s", statusOutput)
	}

	binaryFile := filepath.Join(repoPath, "binary-file.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	hasChanges, err = gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if !hasChanges {
		t.Errorf("Expected to detect binary file, but no changes were found")
	}

	statusOutput, err = gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !strings.Contains(statusOutput, "?? binary-file.bin") {
		t.Errorf("Expected binary file in status, got: %s", statusOutput)
	}

	gb.commitsCount = 0

	commitCtx := context.Background()
	err = gb.RunSingleIteration(commitCtx)
	if err != nil {
		t.Fatalf("Failed to run single iteration: %v", err)
	}

	if gb.commitsCount != 1 {
		t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
	}

	cleanCtx := context.Background()
	hasChanges, err = gb.hasUncommittedChanges(cleanCtx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}
	if hasChanges {
		t.Errorf("Expected working directory to be clean after commit, but found changes")

		statusOutput, err = gb.runGitCommandWithOutput(statusCtx, "status", "--porcelain")
		if err != nil {
			t.Fatalf("Failed to get git status: %v", err)
		}
		t.Logf("Unexpected status after commit: %s", statusOutput)
	}

	commitMsgCtx := context.Background()
	commitMsg, err := gb.runGitCommandWithOutput(commitMsgCtx, "log", "-1", "--pretty=%s")
	if err != nil {
		t.Fatalf("Failed to get commit message: %v", err)
	}
	if !strings.Contains(commitMsg, "[gitbak-test] Commit #1") {
		t.Errorf("Expected commit message to contain '[gitbak-test] Commit #1', got: %s", commitMsg)
	}
}
