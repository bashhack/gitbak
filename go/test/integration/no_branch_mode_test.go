//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNoBranchMode tests the -no-branch flag functionality
func TestNoBranchMode(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	gitbakBin := buildGitbak(t)

	gitCmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	originalBranch := strings.TrimSpace(string(branchOutput))
	t.Logf("Original branch is: %s", originalBranch)

	cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-no-branch", "-non-interactive")
	cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gitbak: %v", err)
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	time.Sleep(3 * time.Second)

	gitCmd = exec.Command("git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err = gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch during gitbak run: %v", err)
	}
	currentBranch := strings.TrimSpace(string(branchOutput))

	if currentBranch != originalBranch {
		t.Errorf("Expected to remain on branch %s, but switched to %s", originalBranch, currentBranch)
	} else {
		t.Logf("Correctly remained on branch %s with -no-branch flag", currentBranch)
	}

	testFile := filepath.Join(repoPath, "no-branch-test.txt")
	err = os.WriteFile(testFile, []byte("No-branch mode test content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change for no-branch test at %s", time.Now().Format("15:04:05"))

	time.Sleep(10 * time.Second)

	gitCmd = exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s")
	logOutput, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	commitMsg := strings.TrimSpace(string(logOutput))
	if !strings.Contains(commitMsg, "[gitbak] Automatic checkpoint #") {
		t.Errorf("Expected a gitbak commit message, got: %s", commitMsg)
	} else {
		t.Logf("No-branch mode successfully created commit: %s", commitMsg)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	gitCmd = exec.Command("git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err = gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch after gitbak: %v", err)
	}
	finalBranch := strings.TrimSpace(string(branchOutput))

	if finalBranch != originalBranch {
		t.Errorf("Expected to remain on branch %s, but ended on %s", originalBranch, finalBranch)
	}
}
