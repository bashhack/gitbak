package git

import (
	"testing"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestNewGitbak tests the NewGitbak function to ensure it correctly sets up dependencies
func TestNewGitbak(t *testing.T) {
	t.Parallel()
	log := logger.New(true, "", true)

	t.Run("NewGitbak with NonInteractive=true", func(t *testing.T) {
		t.Parallel()
		config := GitbakConfig{
			RepoPath:        "/test/repo",
			IntervalMinutes: 5,
			BranchName:      "test-branch",
			CommitPrefix:    "[test] ",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		}

		gb, err := NewGitbak(config, log)
		if err != nil {
			t.Fatalf("NewGitbak returned unexpected error: %v", err)
		}

		if gb == nil {
			t.Fatal("NewGitbak returned nil")
		}

		if gb.config != config {
			t.Errorf("Expected config to match, but was different")
		}

		if gb.logger != log {
			t.Errorf("Expected logger to be set correctly")
		}

		if gb.executor == nil {
			t.Errorf("Expected executor to be set, got nil")
		}

		// Since NonInteractive is true, the interactor should be a NonInteractiveInteractor
		_, isNonInteractive := gb.interactor.(*NonInteractiveInteractor)
		if !isNonInteractive {
			t.Errorf("Expected interactor to be NonInteractiveInteractor when NonInteractive=true")
		}
	})

	t.Run("NewGitbak with NonInteractive=false", func(t *testing.T) {
		t.Parallel()
		config := GitbakConfig{
			RepoPath:        "/test/repo",
			IntervalMinutes: 5,
			BranchName:      "test-branch",
			CommitPrefix:    "[test] ",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  false,
		}

		gb, err := NewGitbak(config, log)
		if err != nil {
			t.Fatalf("NewGitbak returned unexpected error: %v", err)
		}

		if gb == nil {
			t.Fatal("NewGitbak returned nil")
		}

		// Since NonInteractive is false, the interactor should be a DefaultInteractor
		_, isDefaultInteractor := gb.interactor.(*DefaultInteractor)
		if !isDefaultInteractor {
			t.Errorf("Expected interactor to be DefaultInteractor when NonInteractive=false")
		}
	})
}
