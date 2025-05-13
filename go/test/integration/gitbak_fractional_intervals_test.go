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

func TestFractionalIntervals(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test; set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)

	gitbakBin := filepath.Join("..", "..", "build", "gitbak")
	if _, err := os.Stat(gitbakBin); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", gitbakBin, "../../cmd/gitbak")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build gitbak binary: %v", err)
		}
	}

	shortInterval := 0.1

	cmd := exec.Command(gitbakBin,
		"-interval", "0.1",
		"-branch", "fractional-test",
		"-prefix", "[test-fractional]",
		"-repo", repoPath,
		"-non-interactive")
	cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gitbak: %v", err)
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	testFile := filepath.Join(repoPath, "fractional-interval-test.txt")
	if err := os.WriteFile(testFile, []byte("Test content for fractional intervals"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait 12 seconds (should trigger approximately 2 commit cycles at 6-second intervals)
	time.Sleep(12 * time.Second)

	if err := os.WriteFile(testFile, []byte("Test content for fractional intervals\nSecond line"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait 12 more seconds (should trigger approximately 2 more commit cycles)
	time.Sleep(12 * time.Second)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	gitCmd := exec.Command("git", "-C", repoPath, "log", "--grep", "\\[test-fractional\\]")
	output, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	commitCount := strings.Count(string(output), "commit ")

	// We should have at least 2 commits with 24 seconds of runtime at 0.1 minute intervals
	if commitCount < 2 {
		t.Errorf("Expected at least 2 commits with %.1f minute intervals (6 seconds) over 24 seconds, got %d",
			shortInterval, commitCount)

		t.Logf("Git log output:\n%s", output)
	} else {
		t.Logf("Successfully created %d commits with %.1f minute intervals (6 seconds)",
			commitCount, shortInterval)
	}
}
