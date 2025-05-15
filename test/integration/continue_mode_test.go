//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestContinueModeWithoutBranch tests that -continue flag works without specifying a branch
func TestContinueModeWithoutBranch(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	gitbakBin := buildGitbak(t)

	// First, run gitbak normally to create some commits with a minimal interval
	cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-non-interactive")
	cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gitbak: %v", err)
	}

	time.Sleep(3 * time.Second)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("Initial test content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}

	time.Sleep(10 * time.Second)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	gitCmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	branchOutput, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	currentBranch := strings.TrimSpace(string(branchOutput))
	t.Logf("Current branch is: %s", currentBranch)

	// Now run gitbak with -continue but without -branch, but with a short timeout
	// This prevents gitbak from running indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	continueCmd := exec.CommandContext(ctx, gitbakBin, "-interval", "0.1", "-continue", "-debug", "-repo", repoPath, "-non-interactive")
	continueCmd.Env = append(os.Environ(), "GITBAK_TESTING=1")

	continueOutput, err := continueCmd.CombinedOutput()

	// We expect the command to be killed by the timeout, so ignore the context deadline error
	if err != nil && !strings.Contains(err.Error(), "signal: killed") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("Failed to run gitbak in continue mode: %v", err)
	}

	output := string(continueOutput)
	expectedMsg := "🔄 Continuing gitbak session on branch: " + currentBranch
	if !strings.Contains(output, expectedMsg) {
		t.Errorf("Expected message '%s' in output but not found. Output was:\n%s", expectedMsg, output)
	} else {
		t.Logf("Found expected branch message: %s", expectedMsg)
	}

	// Now test the interactive running scenario - start gitbak and make a new commit
	contRunCmd := exec.Command(gitbakBin, "-interval", "0.1", "-continue", "-debug", "-repo", repoPath, "-non-interactive")
	contRunCmd.Env = append(os.Environ(), "GITBAK_TESTING=1")

	if err := contRunCmd.Start(); err != nil {
		t.Fatalf("Failed to start gitbak in continue mode: %v", err)
	}

	defer func() {
		if contRunCmd.Process != nil {
			contRunCmd.Process.Kill()
		}
	}()

	time.Sleep(3 * time.Second)

	continueFile := filepath.Join(repoPath, "continue-detected.txt")
	err = os.WriteFile(continueFile, []byte("Continue mode with auto-detected branch test\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}

	time.Sleep(10 * time.Second)

	// Check that a commit was made (confirming the branch was correctly detected)
	gitLogCmd := exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s")
	logOutput, err := gitLogCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	commitMsg := strings.TrimSpace(string(logOutput))
	if !strings.Contains(commitMsg, "[gitbak] Automatic checkpoint #") {
		t.Errorf("Expected a gitbak commit message, got: %s", commitMsg)
	} else {
		t.Logf("Continue mode with auto-detected branch successfully created commit: %s", commitMsg)
	}
}
