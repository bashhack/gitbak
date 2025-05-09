package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/logger"
)

// setupTestRepo initializes a test git repository
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	cmd := exec.Command("git", "init", tempDir)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("Initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "add", "initial.txt")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to add initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

func TestIsRepository(t *testing.T) {
	repoPath := setupTestRepo(t)

	isRepo, err := IsRepository(repoPath)
	if err != nil {
		t.Fatalf("IsRepository returned unexpected error: %v", err)
	}
	if !isRepo {
		t.Errorf("Expected %s to be recognized as a git repository", repoPath)
	}

	tempDir := t.TempDir()

	isRepo, err = IsRepository(tempDir)
	// For a non-git repo, we expect isRepo to be false and err to be nil
	if err != nil {
		t.Fatalf("IsRepository returned unexpected error for non-repo: %v", err)
	}
	if isRepo {
		t.Errorf("Expected %s to not be recognized as a git repository", tempDir)
	}
}

func TestGitbakNewAndBasicMethods(t *testing.T) {
	ctx := context.Background()
	repoPath := setupTestRepo(t)

	// Place the log file outside the repo directory to avoid affecting git status
	tempLogDir := t.TempDir()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test.log")
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
			BranchName:      "test-branch",
			CommitPrefix:    "[test] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	branch, err := gb.getCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch to be main or master, got %s", branch)
	}

	exists, err := gb.branchExists(ctx, branch)
	if err != nil {
		t.Fatalf("Failed to check if current branch exists: %v", err)
	}
	if !exists {
		t.Errorf("Current branch %s should exist but branchExists returned false", branch)
	}

	exists, err = gb.branchExists(ctx, "non-existent-branch-name-12345")
	if err != nil {
		t.Fatalf("Failed to check if non-existent branch exists: %v", err)
	}
	if exists {
		t.Errorf("Non-existent branch should not exist but branchExists returned true")
	}

	cmd := exec.Command("git", "-C", repoPath, "branch", "test-new-branch")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to create test branch: %v", err)
	}

	exists, err = gb.branchExists(ctx, "test-new-branch")
	if err != nil {
		t.Fatalf("Failed to check if test branch exists: %v", err)
	}
	if !exists {
		t.Errorf("Test branch 'test-new-branch' should exist but branchExists returned false")
	}

	hasChanges, err := gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}

	if hasChanges {
		statusOutput, statusErr := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
		if statusErr != nil {
			t.Fatalf("git status failed unexpectedly: %v", statusErr)
		} else {
			t.Logf("git status reported uncommitted changes: %q", statusOutput)
		}
		t.Error("Expected no uncommitted changes in fresh repo")
	}

	newFile := filepath.Join(repoPath, "new-file.txt")
	err = os.WriteFile(newFile, []byte("New content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	hasChanges, err = gb.hasUncommittedChanges(ctx)
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}

	if !hasChanges {
		t.Error("Expected uncommitted changes after creating new file")
	}
}

func TestFindHighestCommitNumber(t *testing.T) {
	ctx := context.Background()
	repoPath := setupTestRepo(t)

	prefix := "[gitbak] Automatic checkpoint"

	for i := 1; i <= 3; i++ {
		filename := filepath.Join(repoPath, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(filename, []byte(fmt.Sprintf("Content %d", i)), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}

		cmd := exec.Command("git", "-C", repoPath, "add", filepath.Base(filename))
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to stage file %d: %v", i, err)
		}

		commitMsg := fmt.Sprintf("%s #%d - 2023-01-01 12:00:00", prefix, i)
		cmd = exec.Command("git", "-C", repoPath, "commit", "-m", commitMsg)
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to commit file %d: %v", i, err)
		}
	}

	// Place the log file outside the repo directory to avoid affecting git status
	tempLogDir := t.TempDir()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test.log")
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
			BranchName:      "test-branch",
			CommitPrefix:    prefix,
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	highest, err := gb.findHighestCommitNumber(ctx)
	if err != nil {
		t.Fatalf("Failed to find highest commit number: %v", err)
	}

	if highest != 3 {
		t.Errorf("Expected highest commit number to be 3, got %d", highest)
	}
}

// TestRunDirectly tests the logic of the Run method using the exported RunSingleIteration function
func TestRunDirectly(t *testing.T) {
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-direct.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	testFile := filepath.Join(repoPath, "test-run-direct.txt")
	if err := os.WriteFile(testFile, []byte("Test content for Run method direct"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-direct-branch",
			CommitPrefix:    "[gitbak-direct] Commit",
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
		t.Fatalf("RunSingleIteration failed: %v", err)
	}

	currentBranch, err := gb.getCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if currentBranch != "gitbak-direct-branch" {
		t.Errorf("Expected to be on branch 'gitbak-direct-branch', but got '%s'", currentBranch)
	}

	commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", regexp.QuoteMeta("[gitbak-direct]"), "--oneline")
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	if !strings.Contains(commitOutput, "[gitbak-direct]") {
		t.Errorf("Expected to find a commit with prefix '[gitbak-direct]', but got: %s", commitOutput)
	}
}

// TestRunBasic tests the Run method functionality
func TestRunBasic(t *testing.T) {
	t.Run("Non-interactive mode with branch creation", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-branch.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		testFile := filepath.Join(repoPath, "test-run.txt")
		err := os.WriteFile(testFile, []byte("Test content for Run method"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
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

		ctx := context.Background()
		err = gb.RunSingleIteration(ctx)
		if err != nil {
			t.Fatalf("RunSingleIteration failed: %v", err)
		}

		exists, err := gb.branchExists(ctx, "gitbak-test-branch")
		if err != nil {
			t.Fatalf("Failed to check if branch exists: %v", err)
		}
		if !exists {
			t.Errorf("Expected branch 'gitbak-test-branch' to be created")
		}

		grepPattern := regexp.QuoteMeta("[gitbak-test]")
		commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", grepPattern, "--oneline")
		if err != nil {
			t.Fatalf("Failed to get commit log: %v", err)
		}

		if !strings.Contains(commitOutput, "[gitbak-test]") {
			t.Errorf("Expected to find a commit with prefix '[gitbak-test]', but got: %s", commitOutput)
		}

		if gb.commitsCount != 1 {
			t.Errorf("Expected commits count to be 1, got %d", gb.commitsCount)
		}
	})

	t.Run("Continue session mode", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-continue.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		setupBranch := "gitbak-continue-branch"
		commitPrefix := "[gitbak-continue] Commit"

		cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", setupBranch)
		err := cmd.Run()
		if err != nil {
			t.Fatalf("Failed to create continue test branch: %v", err)
		}

		continueFile := filepath.Join(repoPath, "continue-test.txt")
		err = os.WriteFile(continueFile, []byte("Continue test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create continue test file: %v", err)
		}

		cmd = exec.Command("git", "-C", repoPath, "add", "continue-test.txt")
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to add continue test file: %v", err)
		}

		initialCommitMsg := fmt.Sprintf("%s #1 - 2023-01-01 12:00:00", commitPrefix)
		cmd = exec.Command("git", "-C", repoPath, "commit", "-m", initialCommitMsg)
		err = cmd.Run()
		if err != nil {
			t.Fatalf("Failed to commit continue test file: %v", err)
		}

		continueFile2 := filepath.Join(repoPath, "continue-test2.txt")
		err = os.WriteFile(continueFile2, []byte("More continue test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create second continue test file: %v", err)
		}

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      setupBranch,
				CommitPrefix:    commitPrefix,
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: true,
				NonInteractive:  true,
			},
			log,
		)

		ctx := context.Background()
		err = gb.RunSingleIteration(ctx)
		if err != nil {
			t.Fatalf("RunSingleIteration failed in continue mode: %v", err)
		}

		grepPattern := regexp.QuoteMeta(commitPrefix)
		commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", grepPattern, "--oneline")
		if err != nil {
			t.Fatalf("Failed to get commit log: %v", err)
		}

		// Count the number of lines in the output (each line is a commit)
		commitLines := strings.Split(strings.TrimSpace(commitOutput), "\n")
		if len(commitLines) != 2 {
			t.Errorf("Expected 2 commits with prefix '%s', got %d: %s",
				commitPrefix, len(commitLines), commitOutput)
		}

		if !strings.Contains(commitOutput, "#2") {
			t.Errorf("Expected to find commit #2, but output was: %s", commitOutput)
		}
	})

	t.Run("PrintSummary", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-summary.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      "gitbak-summary-branch",
				CommitPrefix:    "[gitbak-summary]",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		gb.originalBranch = "main"
		gb.config.BranchName = "gitbak-summary-branch"
		gb.commitsCount = 5
		gb.startTime = time.Now().Add(-1 * time.Hour)

		// PrintSummary doesn't return anything, but we want to make sure it executes without panicking
		gb.PrintSummary()
	})
}

// TestSetupCurrentBranchSession tests the setupCurrentBranchSession method
func TestSetupCurrentBranchSession(t *testing.T) {
	t.Run("setupCurrentBranchSession normal case", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-current.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        repoPath,
				IntervalMinutes: 1,
				BranchName:      "main",
				CommitPrefix:    "[gitbak-current] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		gb.originalBranch = "main"

		sessionCtx := context.Background()
		gb.setupCurrentBranchSession(sessionCtx)
		// This method doesn't return anything and has no observable side effects
		// other than the log message, so just ensuring it doesn't panic
	})

	t.Run("setupCurrentBranchSession with error getting current branch", func(t *testing.T) {
		t.Parallel()
		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-current-error.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		// Create a gitbak instance with an invalid repository path,
		// this will cause getCurrentBranch to fail
		nonExistentPath := filepath.Join(os.TempDir(), "non-existent-repo")

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        nonExistentPath,
				IntervalMinutes: 1,
				BranchName:      "main",
				CommitPrefix:    "[gitbak-current-error] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			log,
		)

		sessionCtx := context.Background()
		gb.setupCurrentBranchSession(sessionCtx)
		// This should handle the error gracefully and not panic
	})
}

// TestGitbakConfigValidate tests the GitbakConfig.Validate method
func TestGitbakConfigValidate(t *testing.T) {
	tests := map[string]struct {
		config      GitbakConfig
		expectError bool
		errorMsg    string
	}{
		"valid config": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				MaxRetries:      3,
			},
			expectError: false,
		},
		"empty repo path": {
			config: GitbakConfig{
				RepoPath:        "",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				MaxRetries:      3,
			},
			expectError: true,
			errorMsg:    "RepoPath must not be empty",
		},
		"zero interval minutes": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 0,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				MaxRetries:      3,
			},
			expectError: true,
			errorMsg:    "IntervalMinutes must be > 0 (got 0.00)",
		},
		"negative interval minutes": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: -5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				MaxRetries:      3,
			},
			expectError: true,
			errorMsg:    "IntervalMinutes must be > 0 (got -5.00)",
		},
		"empty branch name": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "",
				CommitPrefix:    "[test] ",
				MaxRetries:      3,
			},
			expectError: true,
			errorMsg:    "BranchName must not be empty",
		},
		"empty commit prefix": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "",
				MaxRetries:      3,
			},
			expectError: true,
			errorMsg:    "CommitPrefix must not be empty",
		},
		"negative max retries": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				MaxRetries:      -1,
			},
			expectError: true,
			errorMsg:    "MaxRetries cannot be negative (got -1)",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			err := test.config.Validate()

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestNonInteractivePromptYesNo tests the promptYesNo method in non-interactive mode
func TestNonInteractivePromptYesNo(t *testing.T) {
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-prompt.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-prompt-branch",
			CommitPrefix:    "[gitbak-prompt] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	result := gb.promptYesNo("Test question?")

	if result != false {
		t.Errorf("Expected promptYesNo to return false in non-interactive mode, got %v", result)
	}
}

// TestGitErrorHandlingWithRealCommands tests error handling by using a real repo but
// simulating error conditions through repository state
func TestGitErrorHandlingWithRealCommands(t *testing.T) {
	t.Run("Corrupted repository error handling", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-corrupted.log")
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

		testCtx := context.Background()
		branch, err := gb.getCurrentBranch(testCtx)
		if err != nil {
			t.Fatalf("Failed to get current branch in healthy repo: %v", err)
		}
		if branch == "" {
			t.Fatalf("Expected branch name, got empty string")
		}

		// Deliberately damage the git repository by renaming the .git directory
		gitDir := filepath.Join(repoPath, ".git")
		renamedGitDir := filepath.Join(repoPath, ".git-renamed")
		if err := os.Rename(gitDir, renamedGitDir); err != nil {
			t.Fatalf("Failed to rename .git directory: %v", err)
		}

		defer func() {
			// Restore original .git directory
			if err := os.Rename(renamedGitDir, gitDir); err != nil {
				t.Logf("Failed to restore .git directory: %v", err)
			}
		}()

		_, err = gb.getCurrentBranch(testCtx)
		if err == nil {
			t.Errorf("Expected getCurrentBranch to fail in corrupted repo, but it succeeded")
		}

		_, err = gb.hasUncommittedChanges(testCtx)
		if err == nil {
			t.Errorf("Expected hasUncommittedChanges to fail in corrupted repo, but it succeeded")
		}
	})

	t.Run("Invalid command parameters", func(t *testing.T) {
		t.Parallel()
		repoPath := setupTestRepo(t)

		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-invalid-cmd.log")
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

		// Test with a command that's guaranteed to fail (non-existent ref)
		ctx := context.Background()
		output, err := gb.runGitCommandWithOutput(ctx, "show-ref", "--verify", "refs/heads/non-existent-branch-12345-67890")
		if err == nil {
			t.Errorf("Expected show-ref for non-existent branch to fail, but it succeeded with output: %s", output)
		}
	})

	t.Run("Permissions error handling", func(t *testing.T) {
		t.Parallel()
		tempLogDir := t.TempDir()
		tempLogFile := filepath.Join(tempLogDir, "gitbak-test-permissions.log")
		log := logger.New(true, tempLogFile, true)
		defer func() {
			if err := log.Close(); err != nil {
				t.Logf("Failed to close log: %v", err)
			}
		}()

		readOnlyDir, err := os.MkdirTemp("", "gitbak-readonly-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() {
			// Make it writable again for cleanup
			if err := os.Chmod(readOnlyDir, 0755); err != nil {
				t.Logf("Warning: Failed to restore directory permissions: %v", err)
			}
			if err := os.RemoveAll(readOnlyDir); err != nil {
				t.Logf("Failed to remove read-only dir: %v", err)
			}
		}()

		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to make directory read-only: %v", err)
		}

		gb := setupTestGitbak(
			GitbakConfig{
				RepoPath:        readOnlyDir,
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

		ctx := context.Background()
		err = gb.runGitCommand(ctx, "init")
		if err == nil {
			t.Errorf("Expected git init in read-only directory to fail, but it succeeded")
		}
	})
}
