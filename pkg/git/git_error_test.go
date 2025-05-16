package git

import (
	"context"
	"github.com/bashhack/gitbak/pkg/config"
	gitbakErrors "github.com/bashhack/gitbak/pkg/errors"
	"github.com/bashhack/gitbak/pkg/logger"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRetryLoop is a test helper that simulates the retry loop in the Gitbak struct
func (g *Gitbak) TestRetryLoop(ctx context.Context, iterations int) error {
	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	commitCounter := g.commitsCount + 1

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			opErr := g.tryOperation(ctx, &errorState, func() error {
				var commitWasCreated bool
				return g.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated)
			})

			// If the operation hit max retries, bubble up the fatal error
			if opErr != nil && errorState.consecutiveErrors > g.config.MaxRetries {
				return opErr
			}
		}
	}
	return nil
}

// TestGitErrorHandlingScenarios tests how Gitbak handles various error conditions
func TestGitErrorHandlingScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc    func(t *testing.T) (string, logger.Logger, *Gitbak)
		testMethod   func(t *testing.T, gb *Gitbak) error
		validateFunc func(t *testing.T, gb *Gitbak, err error)
	}{
		"CorruptedRepository": {
			setupFunc: func(t *testing.T) (string, logger.Logger, *Gitbak) {
				repoPath := setupTestRepo(t)
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-corrupted.log")
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

				testCtx := context.Background()
				_, err := gb.getCurrentBranch(testCtx)
				if err != nil {
					t.Fatalf("Failed to get current branch in healthy repo: %v", err)
				}

				// Deliberately damage the git repository by renaming the .git directory
				gitDir := filepath.Join(repoPath, ".git")
				renamedGitDir := filepath.Join(repoPath, ".git-renamed")
				if err := os.Rename(gitDir, renamedGitDir); err != nil {
					t.Fatalf("Failed to rename .git directory: %v", err)
				}

				t.Cleanup(func() {
					if err := os.Rename(renamedGitDir, gitDir); err != nil {
						t.Logf("Failed to restore .git directory: %v", err)
					}
				})

				return repoPath, log, gb
			},
			testMethod: func(t *testing.T, gb *Gitbak) error {
				testCtx := context.Background()
				_, err := gb.getCurrentBranch(testCtx)
				return err
			},
			validateFunc: func(t *testing.T, gb *Gitbak, err error) {
				if err == nil {
					t.Errorf("Expected getCurrentBranch to fail in corrupted repo, but it succeeded")
				}
			},
		},
		"InvalidCommandParameters": {
			setupFunc: func(t *testing.T) (string, logger.Logger, *Gitbak) {
				repoPath := setupTestRepo(t)
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-invalid-cmd.log")
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

				return repoPath, log, gb
			},
			testMethod: func(t *testing.T, gb *Gitbak) error {
				// Test with a command that's guaranteed to fail (non-existent ref)
				ctx := context.Background()
				_, err := gb.runGitCommandWithOutput(ctx, "show-ref", "--verify", "refs/heads/non-existent-branch-12345-67890")
				return err
			},
			validateFunc: func(t *testing.T, gb *Gitbak, err error) {
				if err == nil {
					t.Errorf("Expected show-ref for non-existent branch to fail, but it succeeded")
				}
			},
		},
		"ReadOnlyDirectoryError": {
			setupFunc: func(t *testing.T) (string, logger.Logger, *Gitbak) {
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-readonly.log")
				log := logger.New(true, tempLogFile, true)

				readOnlyDir, err := os.MkdirTemp("", "gitbak-readonly-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// This will block git init because it cannot create a directory with the same name
				gitFilePath := filepath.Join(readOnlyDir, ".git")
				if err := os.WriteFile(gitFilePath, []byte("blocking file"), 0400); err != nil {
					t.Fatalf("Failed to create blocking file: %v", err)
				}

				if err := os.Chmod(readOnlyDir, 0555); err != nil {
					t.Fatalf("Failed to make directory read-only: %v", err)
				}

				t.Cleanup(func() {
					if err := os.Chmod(readOnlyDir, 0755); err != nil {
						t.Logf("Warning: Failed to restore directory permissions: %v", err)
					}
					if err := os.Chmod(gitFilePath, 0644); err != nil {
						t.Logf("Warning: Failed to restore file permissions: %v", err)
					}
					if err := os.RemoveAll(readOnlyDir); err != nil {
						t.Logf("Failed to remove read-only dir: %v", err)
					}
				})

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

				return readOnlyDir, log, gb
			},
			testMethod: func(t *testing.T, gb *Gitbak) error {
				ctx := context.Background()
				err := gb.runGitCommand(ctx, "init")
				return err
			},
			validateFunc: func(t *testing.T, gb *Gitbak, err error) {
				if err == nil {
					t.Errorf("Expected git init in read-only directory to fail, but it succeeded")
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, _, gb := test.setupFunc(t)

			err := test.testMethod(t, gb)

			test.validateFunc(t, gb, err)
		})
	}
}

// TestGitbakConfigValidationScenarios tests the validation of GitbakConfig in various scenarios
func TestGitbakConfigValidationScenarios(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

// TestRetryLogicExitsAfterMaxRetries tests that the monitoring loop stops after hitting MaxRetries
func TestRetryLogicExitsAfterMaxRetries(t *testing.T) {
	t.Parallel()

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-test.log")
	log := logger.New(true, tempLogFile, true)

	mockErr := gitbakErrors.NewGitError("status", nil,
		gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed, "mock git failure for retry test"), "")

	mockExecutor := NewAdvancedMockRetryExecutor(10, mockErr) // More than our MaxRetries
	mockExecutor.NextFailCall = make(chan struct{}, 10)

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        "/mock/repo/path",
			IntervalMinutes: 1,
			BranchName:      "test-branch",
			CommitPrefix:    "[test]",
			MaxRetries:      config.DefaultMaxRetries,
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loopErr := gb.TestRetryLoop(ctx, 10)

	if loopErr == nil {
		t.Fatal("Expected monitoringLoop to exit with error after max retries")
	}

	if !gitbakErrors.Is(loopErr, gitbakErrors.ErrGitOperationFailed) {
		t.Errorf("Expected error to be ErrGitOperationFailed, got: %v", loopErr)
	}

	if !strings.Contains(loopErr.Error(), "maximum retries") {
		t.Errorf("Expected error to mention maximum retries, got: %v", loopErr)
	}

	if mockExecutor.CallCount < gb.config.MaxRetries {
		t.Errorf("Expected at least %d calls to executor, got %d",
			gb.config.MaxRetries, mockExecutor.CallCount)
	}
}

// TestRetryLogicResetOnDifferentError tests that the retry counter resets with different errors
func TestRetryLogicResetOnDifferentError(t *testing.T) {
	t.Parallel()

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-reset-test.log")
	log := logger.New(true, tempLogFile, true)

	mockErr := gitbakErrors.NewGitError("status", nil,
		gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed, "base error message that will vary"), "")

	mockExecutor := NewAdvancedMockRetryExecutor(10, mockErr) // More than our MaxRetries
	mockExecutor.NextFailCall = make(chan struct{}, 10)
	mockExecutor.ShouldResetErrorMsg = true // This will make every other error message different
	mockExecutor.ErrorVariant = 0

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        "/mock/repo/path",
			IntervalMinutes: 1,
			BranchName:      "test-branch",
			CommitPrefix:    "[test]",
			MaxRetries:      config.DefaultMaxRetries, // With default of 3, we should get 6+ calls
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testErr := gb.TestRetryLoop(ctx, 10)

	// We don't care about the specific error here, just that the test ran properly
	_ = testErr

	// Verify the executor was called enough times to demonstrate the counter was reset
	// We should see more calls than if it had failed immediately at MaxRetries
	if mockExecutor.CallCount < 4 {
		t.Errorf("Expected at least 4 calls to executor (indicating counter reset works), got %d",
			mockExecutor.CallCount)
	}
}

// TestRetryLogicResetOnSuccess tests that the retry counter resets after success
func TestRetryLogicResetOnSuccess(t *testing.T) {
	t.Parallel()

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-success-test.log")
	log := logger.New(true, tempLogFile, true)

	// Create mock error and channels
	mockErr := gitbakErrors.NewGitError("status", nil,
		gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed, "intermittent error for success reset test"), "")

	// This executor will fail MaxRetries-1 times, succeed once, then fail MaxRetries-1 times again
	// This pattern confirms the retry counter resets after success
	mockExecutor := NewAdvancedMockRetryExecutor(2, mockErr) // Fail first 2 times (MaxRetries-1)
	mockExecutor.NextFailCall = make(chan struct{}, 10)
	mockExecutor.NextSuccessCall = make(chan struct{}, 10)
	mockExecutor.PermanentFailAfter = 3 // After 3rd call (1 success), fail permanently

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        "/mock/repo/path",
			IntervalMinutes: 1,
			BranchName:      "test-branch",
			CommitPrefix:    "[test]",
			MaxRetries:      3,
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testErr := gb.TestRetryLoop(ctx, 10)

	if testErr == nil {
		t.Fatal("Expected error from monitoringLoop, got nil")
	}

	if mockExecutor.CallCount < 4 {
		t.Errorf("Expected at least 4 calls to executor, got %d",
			mockExecutor.CallCount)
	}
}
