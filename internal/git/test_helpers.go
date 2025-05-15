package git

import (
	"context"
	"github.com/bashhack/gitbak/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

// RunSingleIteration is an exported version of the internal gitbak logic for testing.
// It runs a single iteration of the gitbak process without the infinite loop.
// This function is only available in test builds.
func (g *Gitbak) RunSingleIteration(ctx context.Context) error {
	var err error
	g.originalBranch, err = g.getCurrentBranch(ctx)
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		return err
	}

	if g.config.ContinueSession {
		if err := g.setupContinueSession(ctx); err != nil {
			return err
		}
	} else if g.config.CreateBranch {
		if err := g.setupNewBranchSession(ctx); err != nil {
			return err
		}
	} else {
		g.setupCurrentBranchSession(ctx)
	}

	g.displayStartupInfo()

	// Initialize commit counter based on commit count, exactly as in monitoringLoop
	// g.commitsCount is already set appropriately by setupContinueSession if we're in continue mode
	commitCounter := g.commitsCount + 1

	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	// NOTE: We don't need to increment g.commitsCount here because
	// checkAndCommitChanges already increments it when creating a commit
	err = g.tryOperation(ctx, &errorState, func() error {
		var commitWasCreated bool
		return g.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated)
	})

	return err
}

// setupTestGitbak creates a Gitbak instance for testing with default mocks
// In test context, we panic on validation errors since tests should be providing valid configs
func setupTestGitbak(config GitbakConfig, logger logger.Logger) *Gitbak {
	executor := NewExecExecutor()

	var interactor UserInteractor
	if config.NonInteractive {
		interactor = NewNonInteractiveInteractor()
	} else {
		interactor = NewDefaultInteractor(logger)
	}

	gb, err := NewGitbakWithDeps(config, logger, executor, interactor)
	if err != nil {
		panic("Test setup failed: " + err.Error())
	}
	return gb
}
