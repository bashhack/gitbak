package git

import (
	"context"
	"fmt"
	"github.com/bashhack/gitbak/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsRepository(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupPath    func(t *testing.T) string
		expectedRepo bool
		expectError  bool
	}{
		"Valid Git Repository": {
			setupPath: func(t *testing.T) string {
				return setupTestRepo(t)
			},
			expectedRepo: true,
			expectError:  false,
		},
		"Non-Git Directory": {
			setupPath: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedRepo: false,
			expectError:  false,
		},
		"Non-Existent Path": {
			setupPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "non-existent-subdirectory")
			},
			expectedRepo: false,
			expectError:  false, // IsRepository should handle non-existent paths gracefully
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := test.setupPath(t)

			isRepo, err := IsRepository(path)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("IsRepository returned unexpected error: %v", err)
				}
			}

			if isRepo != test.expectedRepo {
				t.Errorf("Expected IsRepository to return %v for %s, but got %v",
					test.expectedRepo, path, isRepo)
			}
		})
	}
}

// TestGitMethodScenarios tests various git operations and methods in a table-driven way
func TestGitMethodScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc    func(t *testing.T, repoPath string) (*Gitbak, context.Context)
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"GetCurrentBranch": {
			setupFunc: func(t *testing.T, repoPath string) (*Gitbak, context.Context) {
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-current-branch.log")
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

				return gb, context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				branch, err := gb.getCurrentBranch(ctx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				if branch != "main" && branch != "master" {
					t.Errorf("Expected branch to be main or master, got %s", branch)
				}
			},
		},
		"BranchExists": {
			setupFunc: func(t *testing.T, repoPath string) (*Gitbak, context.Context) {
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-branch-exists.log")
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

				setupCtx := context.Background()

				cmd := exec.Command("git", "-C", repoPath, "branch", "test-new-branch")
				err := cmd.Run()
				if err != nil {
					t.Fatalf("Failed to create test branch: %v", err)
				}

				return gb, setupCtx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				currentBranch, err := gb.getCurrentBranch(ctx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				exists, err := gb.branchExists(ctx, currentBranch)
				if err != nil {
					t.Fatalf("Failed to check if current branch exists: %v", err)
				}
				if !exists {
					t.Errorf("Current branch %s should exist but branchExists returned false", currentBranch)
				}

				exists, err = gb.branchExists(ctx, "test-new-branch")
				if err != nil {
					t.Fatalf("Failed to check if test branch exists: %v", err)
				}
				if !exists {
					t.Errorf("Test branch 'test-new-branch' should exist but branchExists returned false")
				}

				exists, err = gb.branchExists(ctx, "non-existent-branch-name-12345")
				if err != nil {
					t.Fatalf("Failed to check if non-existent branch exists: %v", err)
				}
				if exists {
					t.Errorf("Non-existent branch should not exist but branchExists returned true")
				}
			},
		},
		"DetectUncommittedChanges": {
			setupFunc: func(t *testing.T, repoPath string) (*Gitbak, context.Context) {
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-uncommitted.log")
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

				return gb, context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
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

				newFile := filepath.Join(gb.config.RepoPath, "new-file.txt")
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
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repoPath := setupTestRepo(t)
			gb, ctx := test.setupFunc(t, repoPath)

			defer func() {
				if err := gb.logger.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			test.validateFunc(t, gb, ctx)
		})
	}
}

// TestInitializationScenarios tests the initialization of Gitbak instances under various configurations
func TestInitializationScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		validateFunc func(t *testing.T, gb *Gitbak)
	}{
		"NonInteractiveMode": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
			},
			validateFunc: func(t *testing.T, gb *Gitbak) {
				if gb == nil {
					t.Fatal("NewGitbak returned nil")
				}

				if gb.config.RepoPath != "/test/repo" ||
					gb.config.IntervalMinutes != 5 ||
					gb.config.BranchName != "test-branch" ||
					gb.config.CommitPrefix != "[test] " ||
					!gb.config.CreateBranch ||
					!gb.config.Verbose ||
					!gb.config.ShowNoChanges ||
					gb.config.ContinueSession ||
					!gb.config.NonInteractive {
					t.Errorf("Expected config to match, but was different")
				}

				if gb.logger == nil {
					t.Errorf("Expected logger to be set")
				}

				if gb.executor == nil {
					t.Errorf("Expected executor to be set, got nil")
				}

				_, isNonInteractive := gb.interactor.(*NonInteractiveInteractor)
				if !isNonInteractive {
					t.Errorf("Expected interactor to be NonInteractiveInteractor when NonInteractive=true")
				}
			},
		},
		"InteractiveMode": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  false,
			},
			validateFunc: func(t *testing.T, gb *Gitbak) {
				if gb == nil {
					t.Fatal("NewGitbak returned nil")
				}

				if gb.config.RepoPath != "/test/repo" ||
					gb.config.BranchName != "test-branch" ||
					gb.config.NonInteractive != false {
					t.Errorf("Expected config to match, but was different")
				}

				if gb.logger == nil {
					t.Errorf("Expected logger to be set")
				}

				_, isDefaultInteractor := gb.interactor.(*DefaultInteractor)
				if !isDefaultInteractor {
					t.Errorf("Expected interactor to be DefaultInteractor when NonInteractive=false")
				}
			},
		},
		"CustomRetryConfiguration": {
			config: GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test] ",
				CreateBranch:    true,
				Verbose:         false,
				ShowNoChanges:   false,
				ContinueSession: false,
				NonInteractive:  true,
				MaxRetries:      10,
			},
			validateFunc: func(t *testing.T, gb *Gitbak) {
				if gb == nil {
					t.Fatal("NewGitbak returned nil")
				}

				if gb.config.MaxRetries != 10 {
					t.Errorf("Expected MaxRetries to be 10, got %d", gb.config.MaxRetries)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logPath := filepath.Join(t.TempDir(), "gitbak-init.log")
			log := logger.New(true, logPath, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close logger: %v", err)
				}
			}()

			gb, err := NewGitbak(test.config, log)
			if err != nil {
				t.Fatalf("NewGitbak returned unexpected error: %v", err)
			}

			test.validateFunc(t, gb)
		})
	}
}

// TestBranchSessionScenarios tests the branch session initialization in different scenarios
func TestBranchSessionScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"SuccessfulCurrentBranchSession": {
			config: GitbakConfig{
				BranchName:      "main",
				CommitPrefix:    "[gitbak-current] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) context.Context {
				ctx := context.Background()

				var err error
				gb.originalBranch, err = gb.getCurrentBranch(ctx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				return ctx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				gb.setupCurrentBranchSession(ctx)

				// There's no direct observable state to verify since this method
				// primarily logs information, but we can verify it doesn't panic
				// and that the originalBranch is still set
				if gb.originalBranch == "" {
					t.Errorf("Expected originalBranch to remain set")
				}
			},
		},
		"ErrorGettingCurrentBranch": {
			config: GitbakConfig{
				RepoPath:        filepath.Join(os.TempDir(), "non-existent-repo"),
				BranchName:      "main",
				CommitPrefix:    "[gitbak-current-error] Commit",
				CreateBranch:    false,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) context.Context {
				gb.originalBranch = "test-original-branch"
				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				gb.setupCurrentBranchSession(ctx)

				if gb.originalBranch != "test-original-branch" {
					t.Errorf("Expected originalBranch to remain unchanged, got %s", gb.originalBranch)
				}
			},
		},
		"SetupNewBranchSession": {
			config: GitbakConfig{
				BranchName:      "gitbak-new-branch-test",
				CommitPrefix:    "[gitbak-new] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) context.Context {
				ctx := context.Background()

				var err error
				gb.originalBranch, err = gb.getCurrentBranch(ctx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				return ctx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				err := gb.setupNewBranchSession(ctx)
				if err != nil {
					t.Fatalf("setupNewBranchSession failed: %v", err)
				}

				currentBranch, err := gb.getCurrentBranch(ctx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				if currentBranch != "gitbak-new-branch-test" {
					t.Errorf("Expected to be on branch 'gitbak-new-branch-test', got '%s'", currentBranch)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var repoPath string
			if test.config.RepoPath == "" {
				repoPath = setupTestRepo(t)
				test.config.RepoPath = repoPath
			}

			tempLogDir := t.TempDir()
			tempLogFile := filepath.Join(tempLogDir, "gitbak-session-test.log")
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			gb := setupTestGitbak(test.config, log)
			ctx := test.setupFunc(t, gb)
			test.validateFunc(t, gb, ctx)
		})
	}
}

// TestBranchInitializationScenarios tests the branch initialization scenarios in a table-driven manner
func TestBranchInitializationScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"CreateNewBranch": {
			config: GitbakConfig{
				BranchName:      "gitbak-init-branch",
				CommitPrefix:    "[gitbak-init] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) context.Context {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed: %v", err)
				}
				return initCtx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				checkCtx := context.Background()
				currentBranch, err := gb.getCurrentBranch(checkCtx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}
				if currentBranch != "gitbak-init-branch" {
					t.Errorf("Expected to be on branch 'gitbak-init-branch', but got '%s'", currentBranch)
				}

				if gb.originalBranch == "" {
					t.Errorf("Expected originalBranch to be set, but it was empty")
				}
			},
		},
		"CreateUsingRunSingleIteration": {
			config: GitbakConfig{
				BranchName:      "gitbak-workflow-test",
				CommitPrefix:    "[gitbak-workflow] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) context.Context {
				ctx := context.Background()
				err := gb.RunSingleIteration(ctx)
				if err != nil {
					t.Fatalf("Failed to initialize GitBak: %v", err)
				}
				return ctx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
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
			},
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

			config := test.config
			config.RepoPath = repoPath
			gb := setupTestGitbak(config, log)

			ctx := test.setupFunc(t, gb)

			test.validateFunc(t, gb, ctx)
		})
	}
}

// TestBranchEdgeCasesScenarios tests various edge cases in branch handling
func TestBranchEdgeCasesScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc    func(t *testing.T, repoPath string, log logger.Logger) *Gitbak
		testFunc     func(t *testing.T, gb *Gitbak, repoPath string) (context.Context, error)
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"InitializeWithExistingBranch": {
			setupFunc: func(t *testing.T, repoPath string, log logger.Logger) *Gitbak {
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

				initCtx := context.Background()
				err := gb1.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed for setup: %v", err)
				}

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

				return gb2
			},
			testFunc: func(t *testing.T, gb *Gitbak, repoPath string) (context.Context, error) {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed for conflict test: %v", err)
				}
				return initCtx, nil
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				existingBranchName := "gitbak-existing-branch" // Use the original branch name

				checkCtx := context.Background()
				currentBranch, err := gb.getCurrentBranch(checkCtx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				// When a branch already exists, we expect to be on a branch with
				// the original name as a prefix followed by a timestamp
				if currentBranch == existingBranchName {
					t.Errorf("Expected to be on a branch different from '%s', but got the same branch", existingBranchName)
				}
				if !strings.HasPrefix(currentBranch, existingBranchName) {
					t.Errorf("Expected to be on a branch starting with '%s', got '%s'", existingBranchName, currentBranch)
				}
			},
		},
		"RecoverFromDeletedBranch": {
			setupFunc: func(t *testing.T, repoPath string, log logger.Logger) *Gitbak {
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

				return gb2
			},
			testFunc: func(t *testing.T, gb *Gitbak, repoPath string) (context.Context, error) {
				recoveryCtx := context.Background()
				err := gb.RunSingleIteration(recoveryCtx)
				if err != nil {
					t.Fatalf("Failed to recover from deleted branch: %v", err)
				}
				return recoveryCtx, nil
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				deletedBranchName := gb.config.BranchName

				checkCtx := context.Background()
				currentBranch, err := gb.getCurrentBranch(checkCtx)
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}

				if !strings.HasPrefix(currentBranch, deletedBranchName) {
					t.Errorf("Expected to be on a branch starting with '%s', got '%s'", deletedBranchName, currentBranch)
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
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-test-%s.log", name))
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			gb := test.setupFunc(t, repoPath, log)

			ctx, err := test.testFunc(t, gb, repoPath)
			if err != nil {
				t.Fatalf("Test failed: %v", err)
			}

			test.validateFunc(t, gb, ctx)
		})
	}
}

// TestSessionInitializationScenarios tests initialization with continue session vs new branch scenarios
func TestSessionInitializationScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupRepo    func(t *testing.T) (string, logger.Logger, string, string)
		config       func(repoPath, branchName, commitPrefix string) GitbakConfig
		validateFunc func(t *testing.T, gb *Gitbak)
	}{
		"ContinueSessionMode": {
			setupRepo: func(t *testing.T) (string, logger.Logger, string, string) {
				repoPath := setupTestRepo(t)
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-continue.log")
				log := logger.New(true, tempLogFile, true)

				continueBranch := "gitbak-continue-branch"
				continuePrefix := "[gitbak-continue] Commit"

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

				initCtx := context.Background()
				err := gb1.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed for setup: %v", err)
				}

				testFile := filepath.Join(repoPath, "test-continue.txt")
				err = os.WriteFile(testFile, []byte("Test content for continue mode"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				cmdCtx := context.Background()
				err = gb1.runGitCommand(cmdCtx, "add", "test-continue.txt")
				if err != nil {
					t.Fatalf("Failed to stage test file: %v", err)
				}

				initialCommitMsg := continuePrefix + " #1 - 2023-01-01 12:00:00"
				err = gb1.runGitCommand(cmdCtx, "commit", "-m", initialCommitMsg)
				if err != nil {
					t.Fatalf("Failed to commit test file: %v", err)
				}

				return repoPath, log, continueBranch, continuePrefix
			},
			config: func(repoPath, branchName, commitPrefix string) GitbakConfig {
				return GitbakConfig{
					RepoPath:        repoPath,
					IntervalMinutes: 1,
					BranchName:      branchName,
					CommitPrefix:    commitPrefix,
					CreateBranch:    false,
					Verbose:         true,
					ShowNoChanges:   true,
					ContinueSession: true,
					NonInteractive:  true,
				}
			},
			validateFunc: func(t *testing.T, gb *Gitbak) {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed for continue test: %v", err)
				}

				if gb.config.CreateBranch {
					t.Errorf("Expected CreateBranch to be false in continue mode")
				}

				nextFile := filepath.Join(gb.config.RepoPath, "continue-session-file.txt")
				err = os.WriteFile(nextFile, []byte("File added during continue session"), 0644)
				if err != nil {
					t.Fatalf("Failed to create continue session file: %v", err)
				}

				ctx := context.Background()
				var commitWasCreated bool
				err = gb.checkAndCommitChanges(ctx, gb.commitsCount+1, &commitWasCreated)
				if err != nil {
					t.Fatalf("Failed to commit changes: %v", err)
				}

				if !commitWasCreated {
					t.Errorf("Expected a commit to be created")
				}

				output, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
				if err != nil {
					t.Fatalf("Failed to get commit message: %v", err)
				}

				if !strings.Contains(output, "#2") {
					t.Errorf("Expected commit message to contain '#2', got: %s", output)
				}
			},
		},
		"ContinueSessionWithRunSingleIteration": {
			setupRepo: func(t *testing.T) (string, logger.Logger, string, string) {
				repoPath := setupTestRepo(t)
				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-continue-run.log")
				log := logger.New(true, tempLogFile, true)

				continueBranch := "gitbak-continue-iteration-branch"
				continuePrefix := "[gitbak-workflow] Commit"

				setupGb := setupTestGitbak(
					GitbakConfig{
						RepoPath:        repoPath,
						IntervalMinutes: 5,
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

				return repoPath, log, continueBranch, continuePrefix
			},
			config: func(repoPath, branchName, commitPrefix string) GitbakConfig {
				return GitbakConfig{
					RepoPath:        repoPath,
					IntervalMinutes: 5,
					BranchName:      branchName,
					CommitPrefix:    commitPrefix,
					CreateBranch:    false,
					Verbose:         true,
					ShowNoChanges:   true,
					ContinueSession: true,
					NonInteractive:  true,
				}
			},
			validateFunc: func(t *testing.T, gb *Gitbak) {
				nextFile := filepath.Join(gb.config.RepoPath, "continue-session-file.txt")
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
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repoPath, log, branchName, commitPrefix := test.setupRepo(t)
			config := test.config(repoPath, branchName, commitPrefix)
			gb := setupTestGitbak(config, log)
			t.Cleanup(func() { _ = log.Close() })

			test.validateFunc(t, gb)
		})
	}
}
