//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCustomCommitPrefix tests using a custom commit prefix with -prefix flag
func TestCustomCommitPrefix(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	gitbakBin := buildGitbak(t)

	customPrefix := "[custom-test] Backup"

	cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath,
		"-prefix", customPrefix, "-non-interactive")
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

	testFile := filepath.Join(repoPath, "custom-prefix-test.txt")
	err := os.WriteFile(testFile, []byte("Custom prefix test content\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change for custom prefix test at %s", time.Now().Format("15:04:05"))

	time.Sleep(10 * time.Second)

	gitCmd := exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s")
	logOutput, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	commitMsg := strings.TrimSpace(string(logOutput))
	expectedPrefixPattern := regexp.MustCompile(`^\[custom-test\] Backup #\d+`)
	if !expectedPrefixPattern.MatchString(commitMsg) {
		t.Errorf("Expected commit message with custom prefix, got: %s", commitMsg)
	} else {
		t.Logf("Custom prefix correctly used in commit: %s", commitMsg)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	t.Run("Continue mode with custom prefix", func(t *testing.T) {
		gitCmd := exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s")
		output, err := gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get git log: %v", err)
		}

		commitMsg := string(output)
		commitNumRegex := regexp.MustCompile(`#(\d+)`)
		match := commitNumRegex.FindStringSubmatch(commitMsg)
		if len(match) != 2 {
			t.Fatalf("Failed to extract commit number from log: %s", commitMsg)
		}

		previousNum, _ := strconv.Atoi(match[1])
		expectedNextNum := previousNum + 1

		contCmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath,
			"-prefix", customPrefix, "-continue", "-non-interactive")
		contCmd.Env = append(os.Environ(), "GITBAK_TESTING=1")
		if err := contCmd.Start(); err != nil {
			t.Fatalf("Failed to start gitbak in continue mode: %v", err)
		}

		defer func() {
			if contCmd.Process != nil {
				contCmd.Process.Kill()
			}
		}()

		time.Sleep(3 * time.Second)

		testFile := filepath.Join(repoPath, "custom-prefix-continue-test.txt")
		err = os.WriteFile(testFile, []byte("Custom prefix continue test content\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to write to test file: %v", err)
		}
		t.Logf("Made change for custom prefix continue test at %s", time.Now().Format("15:04:05"))

		time.Sleep(10 * time.Second)

		gitCmd = exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s")
		output, err = gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get git log: %v", err)
		}

		commitMsg = strings.TrimSpace(string(output))
		match = commitNumRegex.FindStringSubmatch(commitMsg)
		if len(match) != 2 {
			t.Fatalf("Failed to extract commit number from log: %s", commitMsg)
		}

		actualNum, _ := strconv.Atoi(match[1])
		if actualNum != expectedNextNum {
			t.Errorf("Expected commit #%d, got #%d", expectedNextNum, actualNum)
		} else {
			t.Logf("Continue mode correctly numbered commit: %s", commitMsg)
		}

		if !expectedPrefixPattern.MatchString(commitMsg) {
			t.Errorf("Expected commit message with custom prefix in continue mode, got: %s", commitMsg)
		}
	})
}
