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

// setupTestRepo creates a test git repository
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gitbak-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init", tempDir)
	err = cmd.Run()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com")
	err = cmd.Run()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User")
	err = cmd.Run()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("Initial content"), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "add", "initial.txt")
	err = cmd.Run()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to add initial file: %v", err)
	}

	cmd = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit")
	err = cmd.Run()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tempDir
}

// TestBasicFunctionality tests the basic functionality of gitbak
// This is equivalent to the shell script basic_functionality.sh
func TestBasicFunctionality(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	gitbakBin := filepath.Join("..", "..", "build", "gitbak")
	if _, err := os.Stat(gitbakBin); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", gitbakBin, "../../cmd/gitbak")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build gitbak binary: %v", err)
		}
	}

	cmd := exec.Command(gitbakBin, "-interval", "1", "-debug", "-repo", repoPath, "-non-interactive")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gitbak: %v", err)
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	time.Sleep(10 * time.Second)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("Change 1\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change 1 at %s", time.Now().Format("15:04:05"))

	time.Sleep(120 * time.Second)

	err = os.WriteFile(testFile, []byte("Change 1\nChange 2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change 2 at %s", time.Now().Format("15:04:05"))

	time.Sleep(120 * time.Second)

	err = os.WriteFile(testFile, []byte("Change 1\nChange 2\nChange 3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change 3 at %s", time.Now().Format("15:04:05"))

	time.Sleep(120 * time.Second)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	gitCmd := exec.Command("git", "-C", repoPath, "log", "--grep", "\\[gitbak\\]")
	output, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	commitCount := strings.Count(string(output), "commit ")
	if commitCount < 2 {
		t.Errorf("Expected at least 2 gitbak commits, got %d", commitCount)
	} else {
		t.Logf("Basic functionality test passed: %d commits created", commitCount)
	}
}

// TestLockFile tests the lock file functionality
// This is equivalent to the shell script lock_file.sh
func TestLockFile(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	defer os.RemoveAll(repoPath)

	gitbakBin := filepath.Join("..", "..", "build", "gitbak")
	if _, err := os.Stat(gitbakBin); os.IsNotExist(err) {
		// Try to build it
		buildCmd := exec.Command("go", "build", "-o", gitbakBin, "../../cmd/gitbak")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build gitbak binary: %v", err)
		}
	}

	cmd1 := exec.Command(gitbakBin, "-interval", "60", "-debug", "-repo", repoPath, "-non-interactive")
	if err := cmd1.Start(); err != nil {
		t.Fatalf("Failed to start first gitbak instance: %v", err)
	}

	defer func() {
		if cmd1.Process != nil {
			cmd1.Process.Kill()
		}
	}()

	time.Sleep(2 * time.Second)

	cmd2 := exec.Command(gitbakBin, "-interval", "60", "-debug", "-repo", repoPath, "-non-interactive")
	output, err := cmd2.CombinedOutput()

	if err == nil {
		t.Errorf("Expected second gitbak instance to fail, but it succeeded")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "another gitbak instance is already running") {
		t.Errorf("Expected lock error message, got: %s", outputStr)
	} else {
		t.Logf("Lock file test passed: Second instance correctly detected lock")
	}

	if cmd1.Process != nil {
		cmd1.Process.Kill()
	}
}
