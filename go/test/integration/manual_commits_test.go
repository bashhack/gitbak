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

// TestManualCommitsMixing tests using gitbak with manual commits interspersed
// This simulates a real-world workflow where a developer uses gitbak for safety
// while still making meaningful milestone commits manually
func TestManualCommitsMixing(t *testing.T) {
	if os.Getenv("GITBAK_INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set GITBAK_INTEGRATION_TESTS=1 to run")
	}

	repoPath := setupTestRepo(t)
	gitbakBin := buildGitbak(t)

	cmd := exec.Command(gitbakBin, "-interval", "0.1", "-debug", "-repo", repoPath, "-non-interactive", "-no-branch")
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

	autoFile1 := filepath.Join(repoPath, "auto1.txt")
	err := os.WriteFile(autoFile1, []byte("Automatic commit content 1\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change for auto commit #1 at %s", time.Now().Format("15:04:05"))

	time.Sleep(6500 * time.Millisecond)

	manualFile1 := filepath.Join(repoPath, "manual1.txt")
	err = os.WriteFile(manualFile1, []byte("Manual commit content 1\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write manual file: %v", err)
	}

	gitCmd := exec.Command("git", "-C", repoPath, "add", "manual1.txt")
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to stage manual changes: %v", err)
	}

	gitCmd = exec.Command("git", "-C", repoPath, "commit", "-m", "Manual milestone: Feature A implemented")
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to create manual commit: %v", err)
	}
	t.Logf("Created manual commit #1 at %s", time.Now().Format("15:04:05"))

	autoFile2 := filepath.Join(repoPath, "auto2.txt")
	err = os.WriteFile(autoFile2, []byte("Automatic commit content 2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change for auto commit #2 at %s", time.Now().Format("15:04:05"))

	time.Sleep(6500 * time.Millisecond)

	manualFile2 := filepath.Join(repoPath, "manual2.txt")
	err = os.WriteFile(manualFile2, []byte("Manual commit content 2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write manual file: %v", err)
	}

	gitCmd = exec.Command("git", "-C", repoPath, "add", "manual2.txt")
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to stage manual changes: %v", err)
	}

	gitCmd = exec.Command("git", "-C", repoPath, "commit", "-m", "Manual milestone: Feature B implemented")
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to create manual commit: %v", err)
	}
	t.Logf("Created manual commit #2 at %s", time.Now().Format("15:04:05"))

	autoFile3 := filepath.Join(repoPath, "auto3.txt")
	err = os.WriteFile(autoFile3, []byte("Automatic commit content 3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	t.Logf("Made change for auto commit #3 at %s", time.Now().Format("15:04:05"))

	time.Sleep(6500 * time.Millisecond)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill gitbak process: %v", err)
	}

	t.Logf("gitbak STDOUT: %s", stdoutBuf.String())
	t.Logf("gitbak STDERR: %s", stderrBuf.String())

	gitCmd = exec.Command("git", "-C", repoPath, "log", "--pretty=%H %s")
	logOutput, err := gitCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}
	t.Logf("=== FULL GIT LOG ===\n%s===END GIT LOG===", logOutput)

	gitbakRegex := regexp.MustCompile(`\[gitbak\] Automatic checkpoint #(\d+)`)
	gitbakMatches := gitbakRegex.FindAllStringSubmatch(string(logOutput), -1)

	if len(gitbakMatches) != 3 {
		t.Errorf("Expected 3 gitbak commits, got %d", len(gitbakMatches))
	}

	manualRegex := regexp.MustCompile(`Manual milestone: Feature [A-Z] implemented`)
	manualMatches := manualRegex.FindAllStringSubmatch(string(logOutput), -1)

	if len(manualMatches) != 2 {
		t.Errorf("Expected 2 manual milestone commits, got %d", len(manualMatches))
	}

	var commitNums []int
	for _, match := range gitbakMatches {
		num, _ := strconv.Atoi(match[1])
		commitNums = append(commitNums, num)
	}

	t.Logf("Found gitbak commits with numbers (newest first): %v", commitNums)

	for i := 0; i < len(commitNums)-1; i++ {
		if commitNums[i] != commitNums[i+1]+1 {
			t.Errorf("Gitbak commits are not sequential: got #%d followed by #%d (expected difference of 1)",
				commitNums[i], commitNums[i+1])
		}
	}

	commitLines := strings.Split(strings.TrimSpace(string(logOutput)), "\n")

	t.Log("Commit sequence (newest first):")
	for i, line := range commitLines {
		if i < 7 { // Show only the first 7 commits (initial and our 6 test commits)
			t.Logf("  %s", line)
		}
	}

	// Note: The exact sequence might depend on timing, but we expect auto #3, manual B, auto #2, manual A, auto #1
	expectedPatterns := []string{
		`\[gitbak\] Automatic checkpoint #3`,
		`Manual milestone: Feature B implemented`,
		`\[gitbak\] Automatic checkpoint #2`,
		`Manual milestone: Feature A implemented`,
		`\[gitbak\] Automatic checkpoint #1`,
	}

	seenCount := 0
	for i, pattern := range expectedPatterns {
		found := false
		for j, line := range commitLines {
			if j < len(commitLines)-1 && regexp.MustCompile(pattern).MatchString(line) {
				found = true
				seenCount++
				break
			}
		}
		if !found {
			t.Logf("Warning: Expected pattern '%s' at position %d not found in commit history", pattern, i)
		}
	}

	t.Logf("Found %d of %d expected commit patterns", seenCount, len(expectedPatterns))
}
