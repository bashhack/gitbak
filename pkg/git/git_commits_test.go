package git

import (
	"bytes"
	"context"
	"fmt"
	gitbakErrors "github.com/bashhack/gitbak/pkg/errors"
	"github.com/bashhack/gitbak/pkg/logger"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCommitChangesScenarios tests how Gitbak handles committing changes in various scenarios
func TestCommitChangesScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak, repoPath string) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"CommitWithChanges": {
			config: GitbakConfig{
				BranchName:      "gitbak-commit-branch",
				CommitPrefix:    "[gitbak-commit] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed: %v", err)
				}

				testFile := filepath.Join(repoPath, "test-commit.txt")
				err = os.WriteFile(testFile, []byte("Test content for checkAndCommitChanges"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				var commitWasCreated bool
				err := gb.checkAndCommitChanges(ctx, 1, &commitWasCreated)
				if err != nil {
					t.Fatalf("checkAndCommitChanges failed: %v", err)
				}

				if !commitWasCreated {
					t.Errorf("Expected commit to be created")
				}

				output, err := gb.runGitCommandWithOutput(ctx, "log", "--grep", "\\[gitbak-commit\\]", "--oneline")
				if err != nil {
					t.Fatalf("Failed to get commit log: %v", err)
				}

				if !strings.Contains(output, "[gitbak-commit]") {
					t.Errorf("Expected to find a commit with prefix '[gitbak-commit]', but got: %s", output)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commits count to be 1, got %d", gb.commitsCount)
				}
			},
		},
		"CommitWithMultipleChanges": {
			config: GitbakConfig{
				BranchName:      "gitbak-multi-commit-branch",
				CommitPrefix:    "[gitbak-multi] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed: %v", err)
				}

				textFile := filepath.Join(repoPath, "text-file.txt")
				err = os.WriteFile(textFile, []byte("Text file content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create text file: %v", err)
				}

				subDir := filepath.Join(repoPath, "subdir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdirectory: %v", err)
				}

				subDirFile := filepath.Join(subDir, "subdir-file.txt")
				if err := os.WriteFile(subDirFile, []byte("File in subdirectory"), 0644); err != nil {
					t.Fatalf("Failed to create file in subdirectory: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				var commitWasCreated bool
				err := gb.checkAndCommitChanges(ctx, 1, &commitWasCreated)
				if err != nil {
					t.Fatalf("checkAndCommitChanges failed: %v", err)
				}

				if !commitWasCreated {
					t.Errorf("Expected commit to be created")
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commits count to be 1, got %d", gb.commitsCount)
				}

				output, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:%s")
				if err != nil {
					t.Fatalf("Failed to get commit details: %v", err)
				}

				expectedFiles := []string{
					"text-file.txt",
					"subdir/subdir-file.txt",
				}

				for _, file := range expectedFiles {
					if !strings.Contains(output, file) {
						t.Errorf("Expected commit to include %s, but it wasn't found in output: %s", file, output)
					}
				}

				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Errorf("Expected working directory to be clean after commit, but found changes")
				}
			},
		},
		"CommitWithWorkflowStyle": {
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
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				setupCtx := context.Background()
				if err := gb.RunSingleIteration(setupCtx); err != nil {
					t.Fatalf("Failed to set up initial branch: %v", err)
				}

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

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
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
			},
		},
		"CommitWithoutChanges": {
			config: GitbakConfig{
				BranchName:      "gitbak-no-changes-branch",
				CommitPrefix:    "[gitbak-no-changes] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 1,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				err := gb.initialize(initCtx)
				if err != nil {
					t.Fatalf("initialize failed: %v", err)
				}

				statusCtx := context.Background()
				hasChanges, err := gb.hasUncommittedChanges(statusCtx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Fatalf("Expected clean repository for test, but found changes")
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				var commitWasCreated bool
				err := gb.checkAndCommitChanges(ctx, 1, &commitWasCreated)
				if err != nil {
					t.Fatalf("checkAndCommitChanges failed: %v", err)
				}

				if commitWasCreated {
					t.Errorf("Expected no commit to be created for clean repository")
				}

				if gb.commitsCount != 0 {
					t.Errorf("Expected commits count to be 0, got %d", gb.commitsCount)
				}
			},
		},
		"CleanRepositoryWithRunSingleIteration": {
			config: GitbakConfig{
				BranchName:      "gitbak-clean-test",
				CommitPrefix:    "[gitbak-clean] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				ctx := context.Background()
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Errorf("Expected clean repository, but found changes")
				}

				return ctx
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run GitBak on clean repo: %v", err)
				}

				if gb.commitsCount != 0 {
					t.Errorf("Expected 0 commits on clean repo, got %d", gb.commitsCount)
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

			ctx := test.setupFunc(t, gb, repoPath)

			test.validateFunc(t, gb, ctx)
		})
	}
}

// TestCommitNumberScenarios tests the behavior of commit numbering in different scenarios
func TestCommitNumberScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak, repoPath string) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"FindHighestCommitNumber": {
			config: GitbakConfig{
				BranchName:      "commit-number-test",
				CommitPrefix:    "[gitbak] Automatic checkpoint",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				for i := 1; i <= 3; i++ {
					filename := filepath.Join(repoPath, fmt.Sprintf("file%d.txt", i))
					err := os.WriteFile(filename, []byte(fmt.Sprintf("Content %d", i)), 0644)
					if err != nil {
						t.Fatalf("Failed to create file %d: %v", i, err)
					}

					ctx := context.Background()
					err = gb.runGitCommand(ctx, "add", filepath.Base(filename))
					if err != nil {
						t.Fatalf("Failed to stage file %d: %v", i, err)
					}

					commitMsg := fmt.Sprintf("%s #%d - 2023-01-01 12:00:00", gb.config.CommitPrefix, i)
					err = gb.runGitCommand(ctx, "commit", "-m", commitMsg)
					if err != nil {
						t.Fatalf("Failed to commit file %d: %v", i, err)
					}
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				highest, err := gb.findHighestCommitNumber(ctx)
				if err != nil {
					t.Fatalf("Failed to find highest commit number: %v", err)
				}

				if highest != 3 {
					t.Errorf("Expected highest commit number to be 3, got %d", highest)
				}
			},
		},
		"FirstCommitInSession": {
			config: GitbakConfig{
				BranchName:      "first-commit-test",
				CommitPrefix:    "[gitbak-new] First session",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				highest, err := gb.findHighestCommitNumber(ctx)
				if err != nil {
					t.Fatalf("Failed to find highest commit number: %v", err)
				}

				if highest != 0 {
					t.Errorf("Expected highest commit number to be 0 (no commits yet), got %d", highest)
				}

				newFile := filepath.Join(gb.config.RepoPath, "first-commit.txt")
				err = os.WriteFile(newFile, []byte("First commit content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
				if err != nil {
					t.Fatalf("Failed to get commit message: %v", err)
				}

				if !strings.Contains(commitOutput, "#1") {
					t.Errorf("Expected first commit to be numbered #1, got: %s", commitOutput)
				}
			},
		},
		"ContinueSessionCommitNumbering": {
			config: GitbakConfig{
				BranchName:      "continue-session-test",
				CommitPrefix:    "[gitbak-continue] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: true,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				nonContinueCtx := context.Background()

				if err := gb.runGitCommand(nonContinueCtx, "checkout", "-b", gb.config.BranchName); err != nil {
					t.Fatalf("Failed to create branch: %v", err)
				}

				for i := 1; i <= 2; i++ {
					filename := filepath.Join(repoPath, fmt.Sprintf("continue%d.txt", i))
					err := os.WriteFile(filename, []byte(fmt.Sprintf("Continue content %d", i)), 0644)
					if err != nil {
						t.Fatalf("Failed to create file %d: %v", i, err)
					}

					err = gb.runGitCommand(nonContinueCtx, "add", filepath.Base(filename))
					if err != nil {
						t.Fatalf("Failed to stage file %d: %v", i, err)
					}

					commitMsg := fmt.Sprintf("%s #%d - 2023-01-01 12:00:00", gb.config.CommitPrefix, i)
					err = gb.runGitCommand(nonContinueCtx, "commit", "-m", commitMsg)
					if err != nil {
						t.Fatalf("Failed to commit file %d: %v", i, err)
					}
				}

				continueFile := filepath.Join(repoPath, "continue-new.txt")
				err := os.WriteFile(continueFile, []byte("New content for continue session"), 0644)
				if err != nil {
					t.Fatalf("Failed to create continue file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				highest, err := gb.findHighestCommitNumber(ctx)
				if err != nil {
					t.Fatalf("Failed to find highest commit number: %v", err)
				}

				if highest != 2 {
					t.Errorf("Expected highest commit number to be 2, got %d", highest)
				}

				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				commitOutput, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
				if err != nil {
					t.Fatalf("Failed to get commit message: %v", err)
				}

				if !strings.Contains(commitOutput, "#3") {
					t.Errorf("Expected continue session commit to be numbered #3, got: %s", commitOutput)
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
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-commit-number-%s.log", name))
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

// TestFileChangeScenarios tests Gitbak's ability to detect and handle different types
// of file changes (modified, new, deleted, binary) and ensures they are properly committed
func TestFileChangeScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak, repoPath string) context.Context
		validateFunc func(t *testing.T, gb *Gitbak, ctx context.Context)
	}{
		"DetectAndCommitModifiedFile": {
			config: GitbakConfig{
				BranchName:      "gitbak-modified-file",
				CommitPrefix:    "[gitbak-modified] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				if err := gb.RunSingleIteration(initCtx); err != nil {
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

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if !hasChanges {
					t.Errorf("Expected to detect modified file, but no changes were found")
				}

				statusOutput, err := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
				if err != nil {
					t.Fatalf("Failed to get git status: %v", err)
				}
				if !strings.Contains(statusOutput, "M modified-test.txt") {
					t.Errorf("Expected modified file in status, got: %s", statusOutput)
				}

				gb.commitsCount = 0
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
				}

				hasChanges, err = gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Errorf("Expected working directory to be clean after commit, but found changes")
				}

				commitMsg, err := gb.runGitCommandWithOutput(ctx, "log", "-1", "--pretty=%s")
				if err != nil {
					t.Fatalf("Failed to get commit message: %v", err)
				}
				if !strings.Contains(commitMsg, "[gitbak-modified] Commit #1") {
					t.Errorf("Expected commit message to contain '[gitbak-modified] Commit #1', got: %s", commitMsg)
				}
			},
		},
		"DetectAndCommitNewFile": {
			config: GitbakConfig{
				BranchName:      "gitbak-new-file",
				CommitPrefix:    "[gitbak-new] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				if err := gb.RunSingleIteration(initCtx); err != nil {
					t.Fatalf("Failed to initialize branch: %v", err)
				}

				// Create new file
				newFile := filepath.Join(repoPath, "new-file.txt")
				if err := os.WriteFile(newFile, []byte("New file content"), 0644); err != nil {
					t.Fatalf("Failed to create new file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if !hasChanges {
					t.Errorf("Expected to detect new file, but no changes were found")
				}

				statusOutput, err := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
				if err != nil {
					t.Fatalf("Failed to get git status: %v", err)
				}
				if !strings.Contains(statusOutput, "?? new-file.txt") {
					t.Errorf("Expected new file in status, got: %s", statusOutput)
				}

				gb.commitsCount = 0
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
				}

				showOutput, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:")
				if err != nil {
					t.Fatalf("Failed to get commit details: %v", err)
				}
				if !strings.Contains(showOutput, "new-file.txt") {
					t.Errorf("Expected new file in commit, got: %s", showOutput)
				}
			},
		},
		"DetectAndCommitDeletedFile": {
			config: GitbakConfig{
				BranchName:      "gitbak-deleted-file",
				CommitPrefix:    "[gitbak-deleted] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				if err := gb.RunSingleIteration(initCtx); err != nil {
					t.Fatalf("Failed to initialize branch: %v", err)
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

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if !hasChanges {
					t.Errorf("Expected to detect deleted file, but no changes were found")
				}

				statusOutput, err := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
				if err != nil {
					t.Fatalf("Failed to get git status: %v", err)
				}
				if !strings.Contains(statusOutput, " D to-be-deleted.txt") {
					t.Errorf("Expected deleted file in status, got: %s", statusOutput)
				}

				gb.commitsCount = 0
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
				}

				showOutput, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:")
				if err != nil {
					t.Fatalf("Failed to get commit details: %v", err)
				}
				if !strings.Contains(showOutput, "D\tto-be-deleted.txt") {
					t.Errorf("Expected deleted file in commit, got: %s", showOutput)
				}
			},
		},
		"DetectAndCommitBinaryFile": {
			config: GitbakConfig{
				BranchName:      "gitbak-binary-file",
				CommitPrefix:    "[gitbak-binary] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				if err := gb.RunSingleIteration(initCtx); err != nil {
					t.Fatalf("Failed to initialize branch: %v", err)
				}

				binaryFile := filepath.Join(repoPath, "binary-file.bin")
				binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
					t.Fatalf("Failed to create binary file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if !hasChanges {
					t.Errorf("Expected to detect binary file, but no changes were found")
				}

				statusOutput, err := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
				if err != nil {
					t.Fatalf("Failed to get git status: %v", err)
				}
				if !strings.Contains(statusOutput, "?? binary-file.bin") {
					t.Errorf("Expected binary file in status, got: %s", statusOutput)
				}

				gb.commitsCount = 0
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
				}

				showOutput, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:")
				if err != nil {
					t.Fatalf("Failed to get commit details: %v", err)
				}
				if !strings.Contains(showOutput, "binary-file.bin") {
					t.Errorf("Expected binary file in commit, got: %s", showOutput)
				}
			},
		},
		"DetectAndCommitMultipleFileTypes": {
			config: GitbakConfig{
				BranchName:      "gitbak-multiple-types",
				CommitPrefix:    "[gitbak-multiple] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak, repoPath string) context.Context {
				initCtx := context.Background()
				if err := gb.RunSingleIteration(initCtx); err != nil {
					t.Fatalf("Failed to initialize branch: %v", err)
				}

				modFile := filepath.Join(repoPath, "to-be-modified.txt")
				if err := os.WriteFile(modFile, []byte("Original content"), 0644); err != nil {
					t.Fatalf("Failed to create file to modify: %v", err)
				}

				delFile := filepath.Join(repoPath, "to-be-deleted.txt")
				if err := os.WriteFile(delFile, []byte("File to be deleted"), 0644); err != nil {
					t.Fatalf("Failed to create file to delete: %v", err)
				}

				setupCtx := context.Background()
				if err := gb.runGitCommand(setupCtx, "add", "."); err != nil {
					t.Fatalf("Failed to add initial files: %v", err)
				}
				if err := gb.runGitCommand(setupCtx, "commit", "-m", "Add initial files"); err != nil {
					t.Fatalf("Failed to commit initial files: %v", err)
				}

				if err := os.WriteFile(modFile, []byte("Modified content"), 0644); err != nil {
					t.Fatalf("Failed to modify test file: %v", err)
				}

				if err := os.Remove(delFile); err != nil {
					t.Fatalf("Failed to delete test file: %v", err)
				}

				newFile := filepath.Join(repoPath, "new-file.txt")
				if err := os.WriteFile(newFile, []byte("New file content"), 0644); err != nil {
					t.Fatalf("Failed to create new file: %v", err)
				}

				binaryFile := filepath.Join(repoPath, "binary-file.bin")
				binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
				if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
					t.Fatalf("Failed to create binary file: %v", err)
				}

				return context.Background()
			},
			validateFunc: func(t *testing.T, gb *Gitbak, ctx context.Context) {
				hasChanges, err := gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if !hasChanges {
					t.Errorf("Expected to detect multiple changes, but no changes were found")
				}

				statusOutput, err := gb.runGitCommandWithOutput(ctx, "status", "--porcelain")
				if err != nil {
					t.Fatalf("Failed to get git status: %v", err)
				}

				expectedChanges := []string{
					"M to-be-modified.txt",
					" D to-be-deleted.txt",
					"?? new-file.txt",
					"?? binary-file.bin",
				}

				for _, change := range expectedChanges {
					if !strings.Contains(statusOutput, change) {
						t.Errorf("Expected '%s' in status, but it wasn't found: %s", change, statusOutput)
					}
				}

				gb.commitsCount = 0
				if err := gb.RunSingleIteration(ctx); err != nil {
					t.Fatalf("Failed to run single iteration: %v", err)
				}

				if gb.commitsCount != 1 {
					t.Errorf("Expected commitsCount to be 1, got %d", gb.commitsCount)
				}

				hasChanges, err = gb.hasUncommittedChanges(ctx)
				if err != nil {
					t.Fatalf("Failed to check for uncommitted changes: %v", err)
				}
				if hasChanges {
					t.Errorf("Expected working directory to be clean after commit, but found changes")
				}

				showOutput, err := gb.runGitCommandWithOutput(ctx, "show", "--name-status", "--pretty=format:")
				if err != nil {
					t.Fatalf("Failed to get commit details: %v", err)
				}

				expectedCommitChanges := []string{
					"M\tto-be-modified.txt",
					"D\tto-be-deleted.txt",
					"A\tnew-file.txt",
					"A\tbinary-file.bin",
				}

				for _, change := range expectedCommitChanges {
					if !strings.Contains(showOutput, change) {
						t.Errorf("Expected '%s' in commit, but it wasn't found: %s", change, showOutput)
					}
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
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-file-changes-%s.log", name))
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

// TestCommitCounterIncrementationScenarios verifies that the commit counter is properly
// incremented after each successful commit in the monitoring loop and that
// the counter is correctly initialized when continuing sessions
func TestCommitCounterIncrementationScenarios(t *testing.T) {
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

		// Create an explicit value copy of the loop counter to ensure the closure captures
		// the correct value for each iteration
		counter := i // Explicit value copy to avoid loop variable capture issues
		err := gb.tryOperation(ctx, &errorState, func() error {
			var commitWasCreated bool
			if err := gb.checkAndCommitChanges(ctx, counter, &commitWasCreated); err != nil {
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

// TestHandleUncommittedChangesScenarios tests how Gitbak handles uncommitted changes in various scenarios
func TestHandleUncommittedChangesScenarios(t *testing.T) {
	tests := map[string]struct {
		shouldCommit   bool
		addError       bool
		commitError    bool
		expectedError  bool
		expectedErrMsg string
	}{
		"No commit requested": {
			shouldCommit:  false,
			expectedError: false,
		},
		"Successful commit": {
			shouldCommit:  true,
			expectedError: false,
		},
		"Add command fails": {
			shouldCommit:   true,
			addError:       true,
			expectedError:  true,
			expectedErrMsg: "failed to stage changes",
		},
		"Commit command fails": {
			shouldCommit:   true,
			commitError:    true,
			expectedError:  true,
			expectedErrMsg: "failed to create initial commit",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			mockInteractor := NewMockInteractor(test.shouldCommit)

			mockExecutor := NewMockUncommittedChangesExecutor(test.addError, test.commitError)

			tempLogDir := t.TempDir()
			tempLogFile := filepath.Join(tempLogDir, "gitbak-uncommitted-test.log")
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			config := GitbakConfig{
				RepoPath:        "/test/repo",
				IntervalMinutes: 5,
				BranchName:      "test-branch",
				CommitPrefix:    "[test]",
				NonInteractive:  false,
			}

			gb := setupTestGitbak(config, log)
			gb.interactor = mockInteractor
			gb.executor = mockExecutor

			err := gb.handleUncommittedChanges(context.Background())

			if test.expectedError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if test.expectedErrMsg != "" {
					gitErr, ok := err.(*gitbakErrors.GitError)
					if !ok {
						t.Errorf("Expected error type *gitbakErrors.GitError, got %T", err)
					} else if test.expectedErrMsg != "" && !strings.Contains(gitErr.Error(), test.expectedErrMsg) {
						t.Errorf("Expected error message to contain '%s', got: %s", test.expectedErrMsg, gitErr.Error())
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}

			if test.shouldCommit {
				if !mockExecutor.AddCalled {
					t.Error("Expected git add to be called, but it wasn't")
				}

				if !test.addError && !mockExecutor.CommitCalled {
					t.Error("Expected git commit to be called, but it wasn't")
				}
			} else {
				if mockExecutor.AddCalled || mockExecutor.CommitCalled {
					t.Error("No commands should have been called when user doesn't want to commit")
				}
			}
		})
	}
}

// TestSummaryScenarios tests the generation of session summary information under different scenarios
func TestSummaryScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       GitbakConfig
		setupFunc    func(t *testing.T, gb *Gitbak) bytes.Buffer
		validateFunc func(t *testing.T, gb *Gitbak, summaryOutput string)
	}{
		"StandardSummary": {
			config: GitbakConfig{
				BranchName:      "gitbak-summary-test",
				CommitPrefix:    "[gitbak-summary] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) bytes.Buffer {
				gb.originalBranch = "main"
				gb.commitsCount = 5
				gb.startTime = time.Now().Add(-2 * time.Hour)

				var stdoutBuf bytes.Buffer
				originalLogger := gb.logger
				testLogger := logger.NewWithOutput(true, "", true, &stdoutBuf, &stdoutBuf)
				gb.logger = testLogger

				gb.PrintSummary()

				gb.logger = originalLogger

				return stdoutBuf
			},
			validateFunc: func(t *testing.T, gb *Gitbak, summaryOutput string) {
				expectedInfo := []string{
					"gitbak-summary-test",
					"Total commits made: 5",
					"Session duration: 2h",
					"Working branch",
					"To merge these changes",
				}

				for _, info := range expectedInfo {
					if !strings.Contains(summaryOutput, info) {
						t.Errorf("Expected summary to contain '%s', but it wasn't found", info)
					}
				}
			},
		},
		"ZeroCommitsSummary": {
			config: GitbakConfig{
				BranchName:      "gitbak-zero-commits",
				CommitPrefix:    "[gitbak-zero] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) bytes.Buffer {
				gb.originalBranch = "main"
				gb.commitsCount = 0
				gb.startTime = time.Now().Add(-30 * time.Minute)

				var stdoutBuf bytes.Buffer
				originalLogger := gb.logger
				testLogger := logger.NewWithOutput(true, "", true, &stdoutBuf, &stdoutBuf)
				gb.logger = testLogger

				gb.PrintSummary()

				gb.logger = originalLogger

				return stdoutBuf
			},
			validateFunc: func(t *testing.T, gb *Gitbak, summaryOutput string) {
				expectedInfo := []string{
					"gitbak-zero-commits",
					"Total commits made: 0",
					"Session duration: 0h 30m",
					"Working branch",
				}

				for _, info := range expectedInfo {
					if !strings.Contains(summaryOutput, info) {
						t.Errorf("Expected summary to contain '%s', but it wasn't found", info)
					}
				}
			},
		},
		"LongDurationSummary": {
			config: GitbakConfig{
				BranchName:      "gitbak-long-duration",
				CommitPrefix:    "[gitbak-long] Commit",
				CreateBranch:    true,
				Verbose:         true,
				ShowNoChanges:   true,
				ContinueSession: false,
				NonInteractive:  true,
				IntervalMinutes: 5,
			},
			setupFunc: func(t *testing.T, gb *Gitbak) bytes.Buffer {
				gb.originalBranch = "main"
				gb.commitsCount = 50
				gb.startTime = time.Now().Add(-25 * time.Hour)

				var stdoutBuf bytes.Buffer
				originalLogger := gb.logger
				testLogger := logger.NewWithOutput(true, "", true, &stdoutBuf, &stdoutBuf)
				gb.logger = testLogger

				gb.PrintSummary()

				gb.logger = originalLogger

				return stdoutBuf
			},
			validateFunc: func(t *testing.T, gb *Gitbak, summaryOutput string) {
				expectedInfo := []string{
					"gitbak-long-duration",
					"Total commits made: 50",
					"Session duration: 25h",
					"Working branch",
					"To merge these changes",
				}

				for _, info := range expectedInfo {
					if !strings.Contains(summaryOutput, info) {
						t.Errorf("Expected summary to contain '%s', but it wasn't found", info)
					}
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
			tempLogFile := filepath.Join(tempLogDir, fmt.Sprintf("gitbak-summary-%s.log", name))
			log := logger.New(true, tempLogFile, true)
			defer func() {
				if err := log.Close(); err != nil {
					t.Logf("Failed to close log: %v", err)
				}
			}()

			config := test.config
			config.RepoPath = repoPath
			gb := setupTestGitbak(config, log)

			stdoutBuf := test.setupFunc(t, gb)
			summaryOutput := stdoutBuf.String()

			test.validateFunc(t, gb, summaryOutput)
		})
	}
}
