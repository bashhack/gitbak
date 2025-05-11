package git

import (
	"context"
	"fmt"
	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
	"github.com/bashhack/gitbak/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestExecuteWithOutput(t *testing.T) {
	ctx := context.Background()
	executor := NewExecExecutor()

	cmd := exec.Command("echo", "test output")
	output, err := executor.ExecuteWithOutput(ctx, cmd)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedOutput := "test output"
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, trimmedOutput)
	}

	cmd = exec.Command("false")
	_, err = executor.ExecuteWithOutput(ctx, cmd)

	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}

	if !strings.Contains(err.Error(), "exit status 1") {
		t.Errorf("Unexpected error message: %v", err)
	}

	cmd = exec.Command("non_existent_command_12345")
	_, err = executor.ExecuteWithOutput(ctx, cmd)

	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

func TestExecutorWithContextMethods(t *testing.T) {
	executor := NewExecExecutor()
	ctx := context.Background()

	err := executor.ExecuteWithContext(ctx, "echo", "test context")

	if err != nil {
		t.Errorf("Expected no error for ExecuteWithContext, got: %v", err)
	}

	output, err := executor.ExecuteWithContextAndOutput(ctx, "echo", "test output")

	if err != nil {
		t.Errorf("Expected no error for ExecuteWithContextAndOutput, got: %v", err)
	}

	expectedOutput := "test output"
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, trimmedOutput)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = executor.ExecuteWithContext(canceledCtx, "sleep", "5")
	if err == nil {
		t.Error("Expected error with canceled context, got nil")
	}

	if !gitbakErrors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestContextCancellationScenarios tests different gitbak methods with context cancellation
func TestContextCancellationScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc      func(t *testing.T, repoPath string, log logger.Logger) *Gitbak
		testMethod     func(t *testing.T, gb *Gitbak) error
		timeoutMs      int
		validateFunc   func(t *testing.T, gb *Gitbak, err error)
		maxWaitSeconds int
	}{
		"Run with context": {
			setupFunc: func(t *testing.T, repoPath string, log logger.Logger) *Gitbak {
				testFile := filepath.Join(repoPath, "test-run-context.txt")
				if err := os.WriteFile(testFile, []byte("Test content for Run method with context"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return setupTestGitbak(
					GitbakConfig{
						RepoPath:        repoPath,
						IntervalMinutes: 1,
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
			},
			testMethod: func(t *testing.T, gb *Gitbak) error {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				defer cancel()

				errChan := make(chan error, 1)
				go func() {
					errChan <- gb.Run(ctx)
				}()

				select {
				case err := <-errChan:
					return err
				case <-time.After(5 * time.Second):
					t.Fatal("Run did not return after context cancellation within 5 seconds")
					return nil // Unreachable but needed for the compiler
				}
			},
			timeoutMs: 500,
			validateFunc: func(t *testing.T, gb *Gitbak, err error) {
				if err != nil && !gitbakErrors.Is(err, context.DeadlineExceeded) && !gitbakErrors.Is(err, context.Canceled) {
					t.Errorf("Expected context cancellation error, got: %v", err)
				}

				checkCtx := context.Background()
				branchExists, err := gb.branchExists(checkCtx, "gitbak-context-branch")
				if err != nil {
					t.Fatalf("Failed to check if branch exists: %v", err)
				}
				if !branchExists {
					t.Errorf("Expected branch 'gitbak-context-branch' to be created")
				}

				currentBranch, err := gb.getCurrentBranch(checkCtx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}
				if currentBranch != "gitbak-context-branch" {
					t.Errorf("Expected to be on branch 'gitbak-context-branch', but got '%s'", currentBranch)
				}

				output, err := gb.runGitCommandWithOutput(checkCtx, "diff", "--cached", "--name-only")
				if err != nil {
					t.Fatalf("Failed to get staged changes: %v", err)
				}
				if !os.IsNotExist(err) && !filepath.IsLocal(output) {
					t.Logf("Staged changes after context cancellation: %s", output)
				}
			},
			maxWaitSeconds: 5,
		},
		"MonitoringLoop with context": {
			setupFunc: func(t *testing.T, repoPath string, log logger.Logger) *Gitbak {
				gb := setupTestGitbak(
					GitbakConfig{
						RepoPath:        repoPath,
						IntervalMinutes: 1,
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

				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed: %v", err)
				}

				testFile := filepath.Join(repoPath, "test-monitor.txt")
				err = os.WriteFile(testFile, []byte("Test content for monitoring loop"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return gb
			},
			testMethod: func(t *testing.T, gb *Gitbak) error {
				ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
				defer cancel()

				errChan := make(chan error, 1)
				go func() {
					errChan <- gb.monitoringLoop(ctx)
				}()

				select {
				case err := <-errChan:
					return err
				case <-time.After(5 * time.Second):
					t.Fatal("monitoringLoop did not return after context cancellation within 5 seconds")
					return nil // Unreachable but needed for the compiler
				}
			},
			timeoutMs: 300,
			validateFunc: func(t *testing.T, gb *Gitbak, err error) {
				if err != nil && !gitbakErrors.Is(err, context.DeadlineExceeded) && !gitbakErrors.Is(err, context.Canceled) {
					t.Errorf("Expected context cancellation error, got: %v", err)
				}
			},
			maxWaitSeconds: 5,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repoPath := setupTestRepo(t)
			tempLogDir := t.TempDir()
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-test-%s.log", name))
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			gb := test.setupFunc(t, repoPath, log)

			err := test.testMethod(t, gb)

			test.validateFunc(t, gb, err)
		})
	}
}

// TestRunBehaviorScenarios tests the RunSingleIteration method under different branch and commit scenarios
func TestRunBehaviorScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak, repoPath string) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"NewBranchCreation": {
			config: GitbakConfig{
				BranchName:      "gitbak-test-branch",
				CommitPrefix:    "[gitbak-test] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				testFile := filepath.Join(repoPath, "test-run.txt")
				err := os.WriteFile(testFile, []byte("Test content for Run method"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				err := gb.RunSingleIteration(ctx)
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
			},
		},
		"ContinueSessionMode": {
			config: GitbakConfig{
				BranchName:      "gitbak-continue-branch",
				CommitPrefix:    "[gitbak-continue] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: true,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				setupCtx := context.Background()
				err := gb.runGitCommand(setupCtx, "checkout", "-b", gb.config.BranchName)
				if err != nil {
					t.Fatalf("Failed to create continue test branch: %v", err)
				}

				continueFile := filepath.Join(repoPath, "continue-test.txt")
				err = os.WriteFile(continueFile, []byte("Continue test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create continue test file: %v", err)
				}

				err = gb.runGitCommand(setupCtx, "add", "continue-test.txt")
				if err != nil {
					t.Fatalf("Failed to add continue test file: %v", err)
				}

				initialCommitMsg := fmt.Sprintf("%s #1 - 2023-01-01 12:00:00", gb.config.CommitPrefix)
				err = gb.runGitCommand(setupCtx, "commit", "-m", initialCommitMsg)
				if err != nil {
					t.Fatalf("Failed to commit continue test file: %v", err)
				}

				continueFile2 := filepath.Join(repoPath, "continue-test2.txt")
				err = os.WriteFile(continueFile2, []byte("More continue test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create second continue test file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				err := gb.RunSingleIteration(ctx)
				if err != nil {
					t.Fatalf("RunSingleIteration failed in continue mode: %v", err)
				}

				grepPattern := regexp.QuoteMeta(gb.config.CommitPrefix)
				commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", grepPattern, "--oneline")
				if err != nil {
					t.Fatalf("Failed to get commit log: %v", err)
				}

				commitLines := strings.Split(strings.TrimSpace(commitOutput), "\n")
				if len(commitLines) != 2 {
					t.Errorf("Expected 2 commits with prefix '%s', got %d: %s",
						gb.config.CommitPrefix, len(commitLines), commitOutput)
				}

				if !strings.Contains(commitOutput, "#2") {
					t.Errorf("Expected to find commit #2, but output was: %s", commitOutput)
				}
			},
		},
		"NoChangesScenario": {
			config: GitbakConfig{
				BranchName:      "gitbak-no-changes",
				CommitPrefix:    "[gitbak-no-changes] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				setupCtx := context.Background()
				err := gb.runGitCommand(setupCtx, "checkout", "-b", gb.config.BranchName)
				if err != nil {
					t.Fatalf("Failed to create no-changes test branch: %v", err)
				}

				hasChanges, err := gb.hasUncommittedChanges(setupCtx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Fatalf("Expected clean repository for test, but found changes")
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				initialCount := gb.commitsCount

				err := gb.RunSingleIteration(ctx)
				if err != nil {
					t.Fatalf("RunSingleIteration failed in no-changes scenario: %v", err)
				}

				if gb.commitsCount != initialCount {
					t.Errorf("Expected commit count to remain at %d, but got %d",
						initialCount, gb.commitsCount)
				}

				grepPattern := regexp.QuoteMeta(gb.config.CommitPrefix)
				commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", grepPattern, "--oneline")
				if err != nil {
					t.Fatalf("Failed to get commit log: %v", err)
				}

				if strings.TrimSpace(commitOutput) != "" {
					t.Errorf("Expected no commits with prefix '%s', but found: %s",
						gb.config.CommitPrefix, commitOutput)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repoPath := setupTestRepo(t)
			tempLogDir := t.TempDir()
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-run-test-%s.log", name))
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			config := test.config
			config.RepoPath = repoPath
			gb := setupTestGitbak(config, log)

			ctx := test.setupFunc(t, gb, repoPath)

			test.validateFunc(t, gb, ctx)
		})
	}
}
