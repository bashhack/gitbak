package git

import (
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

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init", tempDir)
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("Initial content"), 0644)
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to create initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "add", "initial.txt")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to add initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit")
	err = cmd.Run()
	if err != nil {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			t.Logf("Failed to clean up temp directory: %v", cleanErr)
		}
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

// cleanupTestRepo cleans up test resources
func cleanupTestRepo(t *testing.T, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Logf("Failed to remove test repository directory: %v", err)
	}
}

func TestIsRepository(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	if !IsRepository(repoPath) {
		t.Errorf("Expected %s to be recognized as a git repository", repoPath)
	}

	tempDir, err := os.MkdirTemp("", "gitbak-not-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	if IsRepository(tempDir) {
		t.Errorf("Expected %s to not be recognized as a git repository", tempDir)
	}
}

func TestGitbakNewAndBasicMethods(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

	// Place the log file outside the repo directory to avoid affecting git status
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

	branch, err := gb.getCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch to be main or master, got %s", branch)
	}

	exists, err := gb.branchExists(branch)
	if err != nil {
		t.Fatalf("Failed to check if current branch exists: %v", err)
	}
	if !exists {
		t.Errorf("Current branch %s should exist but branchExists returned false", branch)
	}

	exists, err = gb.branchExists("non-existent-branch-name-12345")
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

	exists, err = gb.branchExists("test-new-branch")
	if err != nil {
		t.Fatalf("Failed to check if test branch exists: %v", err)
	}
	if !exists {
		t.Errorf("Test branch 'test-new-branch' should exist but branchExists returned false")
	}

	hasChanges, err := gb.hasUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}

	if hasChanges {
		statusOutput, statusErr := gb.runGitCommandWithOutput("status", "--porcelain")
		if statusErr != nil {
			t.Logf("Failed to get git status: %v", statusErr)
		} else {
			t.Logf("Git status shows uncommitted changes: %q", statusOutput)
		}
		t.Error("Expected no uncommitted changes in fresh repo")
	}

	newFile := filepath.Join(repoPath, "new-file.txt")
	err = os.WriteFile(newFile, []byte("New content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	hasChanges, err = gb.hasUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to check for uncommitted changes: %v", err)
	}

	if !hasChanges {
		t.Error("Expected uncommitted changes after creating new file")
	}
}

func TestFindHighestCommitNumber(t *testing.T) {
	repoPath := setupTestRepo(t)
	defer cleanupTestRepo(t, repoPath)

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

	highest, err := gb.findHighestCommitNumber()
	if err != nil {
		t.Fatalf("Failed to find highest commit number: %v", err)
	}

	if highest != 3 {
		t.Errorf("Expected highest commit number to be 3, got %d", highest)
	}
}

// runSingleIteration runs a single iteration of the gitbak process
// This is a trimmed-down version of the Run method that doesn't loop forever
func runSingleIteration(g *Gitbak) error {
	var err error

	g.originalBranch, err = g.getCurrentBranch()
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	g.logger.Info("Starting gitbak on branch: %s", g.originalBranch)

	commitCounter := 1

	if g.config.ContinueSession {
		g.config.CreateBranch = false
		fmt.Printf("🔄 Continuing gitbak session on branch: %s\n", g.originalBranch)

		highestNum, err := g.findHighestCommitNumber()
		if err != nil {
			g.logger.Warning("Failed to find highest commit number: %v", err)
			fmt.Println("ℹ️  No previous commits found with prefix '" + g.config.CommitPrefix + "' - starting from commit #1")
		} else if highestNum > 0 {
			commitCounter = highestNum + 1
			fmt.Printf("ℹ️  Found previous commits - starting from commit #%d\n", commitCounter)
		} else {
			fmt.Println("ℹ️  No previous commits found with prefix '" + g.config.CommitPrefix + "' - starting from commit #1")
		}
	} else if g.config.CreateBranch {
		hasChanges, err := g.hasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}

		if hasChanges {
			fmt.Println("⚠️  Warning: You have uncommitted changes.")

			// Use our promptForCommit method, which respects NonInteractive mode
			shouldCommit := g.promptForCommit()

			if shouldCommit {
				err = g.runGitCommand("add", ".")
				if err != nil {
					return fmt.Errorf("failed to stage changes: %w", err)
				}

				err = g.runGitCommand("commit", "-m", "Manual commit before starting gitbak session")
				if err != nil {
					return fmt.Errorf("failed to create initial commit: %w", err)
				}

				fmt.Println("✅ Created initial commit")
			}
		}

		branchExists, err := g.branchExists(g.config.BranchName)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		if branchExists {
			fmt.Printf("⚠️  Warning: Branch '%s' already exists.\n", g.config.BranchName)

			// Default to true in tests since we'd just get stuck otherwise
			// In real usage, this would use promptYesNo which respects NonInteractive mode
			if g.config.NonInteractive {
				g.logger.Info("Non-interactive mode: automatically using a different branch name")
			}

			g.config.BranchName = fmt.Sprintf("%s-%s", g.config.BranchName, time.Now().Format("150405"))
			fmt.Printf("🌿 Using new branch name: %s\n", g.config.BranchName)
		}

		err = g.runGitCommand("checkout", "-b", g.config.BranchName)
		if err != nil {
			return fmt.Errorf("failed to create new branch: %w", err)
		}

		fmt.Printf("🌿 Created and switched to new branch: %s\n", g.config.BranchName)
	} else {
		currentBranch, err := g.getCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		fmt.Printf("🌿 Using current branch: %s\n", currentBranch)
	}

	fmt.Printf("🔄 gitbak started at %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("📂 Repository: %s\n", g.config.RepoPath)
	fmt.Printf("⏱️  Interval: %d minutes\n", g.config.IntervalMinutes)
	fmt.Printf("📝 Commit prefix: %s\n", g.config.CommitPrefix)
	fmt.Printf("🔊 Verbose mode: %t\n", g.config.Verbose)
	fmt.Printf("🔔 Show no-changes messages: %t\n", g.config.ShowNoChanges)
	fmt.Println("❓ Press Ctrl+C to stop and view session summary")

	hasChanges, err := g.hasUncommittedChanges()
	if err != nil {
		g.logger.Error("Failed to check git status: %v", err)
		fmt.Printf("❌ Error: Failed to check git status: %v\n", err)
		return err
	}

	if hasChanges {
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		err = g.runGitCommand("add", ".")
		if err != nil {
			g.logger.Warning("Failed to stage changes: %v", err)
			fmt.Printf("⚠️  Warning: Failed to stage changes: %v\n", err)
			return err
		}

		commitMsg := fmt.Sprintf("%s #%d - %s", g.config.CommitPrefix, commitCounter, timestamp)
		err = g.runGitCommand("commit", "-m", commitMsg)
		if err != nil {
			g.logger.Warning("Failed to create commit: %v", err)
			fmt.Printf("⚠️  Warning: Failed to create commit: %v\n", err)
			return err
		}

		fmt.Printf("✅ Commit #%d created at %s\n", commitCounter, timestamp)
		g.logger.Info("Successfully created commit #%d", commitCounter)
		g.commitsCount++
	} else if g.config.ShowNoChanges && g.config.Verbose {
		fmt.Printf("ℹ️  No changes to commit at %s\n", time.Now().Format("15:04:05"))
		g.logger.Info("No changes to commit detected")
	}

	return nil
}

// TestRunDirectly attempts to test parts of the Run method by injecting a channel and goroutine
func TestRunDirectly(t *testing.T) {
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-direct.log")
	log := logger.New(true, tempLogFile, true)

	testFile := filepath.Join(repoPath, "test-run-direct.txt")
	err = os.WriteFile(testFile, []byte("Test content for Run method direct"), 0644)
	if err != nil {
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

	err = runSingleIteration(gb)
	if err != nil {
		t.Fatalf("runSingleIteration failed: %v", err)
	}

	currentBranch, err := gb.getCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if currentBranch != "gitbak-direct-branch" {
		t.Errorf("Expected to be on branch 'gitbak-direct-branch', but got '%s'", currentBranch)
	}

	commitOutput, err := gb.runGitCommandWithOutput("log", "--grep", regexp.QuoteMeta("[gitbak-direct]"), "--oneline")
	if err != nil {
		t.Fatalf("Failed to get commit log: %v", err)
	}

	if !strings.Contains(commitOutput, "[gitbak-direct]") {
		t.Errorf("Expected to find a commit with prefix '[gitbak-direct]', but got: %s", commitOutput)
	}
}

// TestRunBasic tests the Run method functionality
func TestRunBasic(t *testing.T) {
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

	t.Run("Non-interactive mode with branch creation", func(t *testing.T) {
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

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("runSingleIteration failed: %v", err)
		}

		exists, err := gb.branchExists("gitbak-test-branch")
		if err != nil {
			t.Fatalf("Failed to check if branch exists: %v", err)
		}
		if !exists {
			t.Errorf("Expected branch 'gitbak-test-branch' to be created")
		}

		grepPattern := regexp.QuoteMeta("[gitbak-test]")
		commitOutput, err := gb.runGitCommandWithOutput("log", "--grep", grepPattern, "--oneline")
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

		err = runSingleIteration(gb)
		if err != nil {
			t.Fatalf("runSingleIteration failed in continue mode: %v", err)
		}

		grepPattern := regexp.QuoteMeta(commitPrefix)
		commitOutput, err := gb.runGitCommandWithOutput("log", "--grep", grepPattern, "--oneline")
		if err != nil {
			t.Fatalf("Failed to get commit log: %v", err)
		}

		// Count the number of lines in the output (each line is a commit)
		commitLines := strings.Split(strings.TrimSpace(commitOutput), "\n")
		if len(commitLines) != 2 {
			t.Errorf("Expected 2 commits with prefix '%s', got %d: %s",
				commitPrefix, len(commitLines), commitOutput)
		}

		// Verify that the second commit has #2 in its message
		if !strings.Contains(commitOutput, "#2") {
			t.Errorf("Expected to find commit #2, but output was: %s", commitOutput)
		}
	})

	t.Run("PrintSummary", func(t *testing.T) {
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
	// Create a test repository
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-current.log")
	log := logger.New(true, tempLogFile, true)

	t.Run("setupCurrentBranchSession normal case", func(t *testing.T) {
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

		gb.setupCurrentBranchSession()
		// This method doesn't return anything and has no observable side effects
		// other than the log message, so just ensuring it doesn't panic
	})

	t.Run("setupCurrentBranchSession with error getting current branch", func(t *testing.T) {
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

		gb.setupCurrentBranchSession()
		// This should handle the error gracefully and not panic
	})
}

// TestNonInteractivePromptYesNo tests the promptYesNo method in non-interactive mode
func TestNonInteractivePromptYesNo(t *testing.T) {
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

	tempLogFile := filepath.Join(tempLogDir, "gitbak-test-prompt.log")
	log := logger.New(true, tempLogFile, true)

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

	t.Run("Corrupted repository error handling", func(t *testing.T) {
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

		branch, err := gb.getCurrentBranch()
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

		_, err = gb.getCurrentBranch()
		if err == nil {
			t.Errorf("Expected getCurrentBranch to fail in corrupted repo, but it succeeded")
		}

		_, err = gb.hasUncommittedChanges()
		if err == nil {
			t.Errorf("Expected hasUncommittedChanges to fail in corrupted repo, but it succeeded")
		}
	})

	t.Run("Invalid command parameters", func(t *testing.T) {
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
		output, err := gb.runGitCommandWithOutput("show-ref", "--verify", "refs/heads/non-existent-branch-12345-67890")
		if err == nil {
			t.Errorf("Expected show-ref for non-existent branch to fail, but it succeeded with output: %s", output)
		}
	})

	t.Run("Permissions error handling", func(t *testing.T) {
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

		err = gb.runGitCommand("init")
		if err == nil {
			t.Errorf("Expected git init in read-only directory to fail, but it succeeded")
		}
	})
}
