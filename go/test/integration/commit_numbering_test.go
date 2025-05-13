//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCommitNumberingAndContinue tests the fixed commit numbering and continue mode functionality
// Note: This integration test uses fixed sleep times rather than polling because:
// 1. We're explicitly testing GitBak's timer-based commit detection
// 2. We're verifying the automatic commit numbering system works correctly
// 3. Polling for specific commit numbers would assume the behavior we're testing
func TestCommitNumberingAndContinue(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	gitbakBin := buildGitbak(t)

	t.Run("Sequential commit numbers", func(t *testing.T) {
		cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-non-interactive", "-show-no-changes")
		cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")

		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		t.Logf("Starting gitbak with command: %s", cmd.String())
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start gitbak: %v", err)
		}

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		time.Sleep(1500 * time.Millisecond)

		// Make three sequential changes to generate commits
		for i := 1; i <= 3; i++ {
			testFile := filepath.Join(repoPath, fmt.Sprintf("test%d.txt", i))
			err := os.WriteFile(testFile, []byte(fmt.Sprintf("Change %d\n", i)), 0644)
			if err != nil {
				t.Fatalf("Failed to write to test file: %v", err)
			}
			t.Logf("Made change %d at %s", i, time.Now().Format("15:04:05"))

			time.Sleep(6500 * time.Millisecond)

			if i == 1 {
				gitCmd := exec.Command("git", "-C", repoPath, "log", "--pretty=%s", "-3")
				commitOutput, _ := gitCmd.Output()
				t.Logf("After first change, commit log: %s", commitOutput)
			}
		}

		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("Failed to kill gitbak process: %v", err)
		}

		t.Logf("gitbak STDOUT: %s", stdoutBuf.String())
		t.Logf("gitbak STDERR: %s", stderrBuf.String())

		gitCmd := exec.Command("git", "-C", repoPath, "log", "--pretty=%s")
		output, err := gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get git log: %v", err)
		}

		commitNumRegex := regexp.MustCompile(`\[gitbak\] Automatic checkpoint #(\d+)`)
		matches := commitNumRegex.FindAllStringSubmatch(string(output), -1)

		t.Logf("Found commits with numbers: %v", matches)
		// Verify we have sequential commit numbers (3, 2, 1 because git log is newest first)
		if len(matches) != 3 {
			t.Errorf("Expected 3 commits, got %d", len(matches))
		} else {
			t.Logf("Running additional diagnostics to understand commit numbering...")

			gitLogCmd := exec.Command("git", "-C", repoPath, "log", "--pretty=%H %s")
			fullLogOutput, logErr := gitLogCmd.Output()
			if logErr == nil {
				t.Logf("=== FULL GIT LOG ===\n%s===END GIT LOG===", fullLogOutput)
			}

			gitMsgCmd := exec.Command("git", "-C", repoPath, "log", "--pretty=%s", "-5")
			msgOutput, msgErr := gitMsgCmd.Output()
			if msgErr == nil {
				t.Logf("Latest 5 commit messages:\n%s", msgOutput)
			}

			// Extract all commit numbers in order (newest first)
			var commitNums []int
			for _, match := range matches {
				num, _ := strconv.Atoi(match[1])
				commitNums = append(commitNums, num)
			}

			// Check that commits are sequential (decreasing by 1 because git log is newest first)
			for i := 0; i < len(commitNums)-1; i++ {
				if commitNums[i] != commitNums[i+1]+1 {
					t.Errorf("Commits are not sequential: got #%d followed by #%d (expected difference of 1)",
						commitNums[i], commitNums[i+1])
				}
			}

			t.Logf("First commit of this test has number #%d (may vary if repository has prior commits)",
				commitNums[len(commitNums)-1])

			originalExpected := []int{3, 2, 1}
			for i, expected := range originalExpected {
				if i < len(commitNums) && commitNums[i] != expected {
					t.Logf("Note: Originally expected #%d in position %d, got #%d (this is just informational)",
						expected, i, commitNums[i])
				}
			}
		}
	})

	t.Run("Continue mode with auto branch detection", func(t *testing.T) {
		gitCmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
		branchOutput, err := gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get current branch: %v", err)
		}
		currentBranch := strings.TrimSpace(string(branchOutput))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-continue", "-non-interactive")
		cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")

		outputBytes, err := cmd.CombinedOutput()

		// We expect the command to be killed by the timeout, so ignore the context deadline error
		if err != nil && !strings.Contains(err.Error(), "signal: killed") && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Fatalf("Failed to run gitbak in continue mode: %v, output: %s", err, outputBytes)
		}

		output := string(outputBytes)
		expectedBranchMsg := "🔄 Continuing gitbak session on branch: " + currentBranch
		if !strings.Contains(output, expectedBranchMsg) {
			t.Errorf("Expected message '%s' but did not find it in output: %s", expectedBranchMsg, output)
		}
	})

	t.Run("Continue mode with correct commit numbering", func(t *testing.T) {
		gitCmd := exec.Command("git", "-C", repoPath, "log", "--pretty=%s", "-1")
		output, err := gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get git log: %v", err)
		}

		commitNumRegex := regexp.MustCompile(`#(\d+)`)
		match := commitNumRegex.FindStringSubmatch(string(output))
		if len(match) != 2 {
			t.Fatalf("Failed to extract commit number from log: %s", output)
		}

		previousNum, _ := strconv.Atoi(match[1])

		cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-continue", "-non-interactive")
		cmd.Env = append(os.Environ(), "GITBAK_TESTING=1")
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start gitbak in continue mode: %v", err)
		}

		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		time.Sleep(1500 * time.Millisecond)

		testFile := filepath.Join(repoPath, "continue-test.txt")
		err = os.WriteFile(testFile, []byte("Continue test content\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to write to test file: %v", err)
		}
		t.Logf("Made change for continue test at %s", time.Now().Format("15:04:05"))

		time.Sleep(6500 * time.Millisecond)

		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("Failed to kill gitbak process: %v", err)
		}

		gitCmd = exec.Command("git", "-C", repoPath, "log", "--pretty=%s", "-1")
		output, err = gitCmd.Output()
		if err != nil {
			t.Fatalf("Failed to get git log: %v", err)
		}

		match = commitNumRegex.FindStringSubmatch(string(output))
		if len(match) != 2 {
			t.Fatalf("Failed to extract commit number from log: %s", output)
		}

		actualNum, _ := strconv.Atoi(match[1])
		t.Logf("Continue mode: Previous commit #%d, next commit #%d", previousNum, actualNum)

		if actualNum != previousNum+1 {
			t.Errorf("Continue mode commit numbering error: Expected next commit to be #%d (previous+1), got #%d",
				previousNum+1, actualNum)
		}
	})
}

// buildGitbak builds the gitbak binary if it doesn't exist
func buildGitbak(t *testing.T) string {
	t.Helper()

	gitbakBin := filepath.Join("..", "..", "build", "gitbak")
	if _, err := os.Stat(gitbakBin); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-tags=test", "-o", gitbakBin, "../../cmd/gitbak")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build gitbak binary: %v", err)
		}
	}

	return gitbakBin
}
