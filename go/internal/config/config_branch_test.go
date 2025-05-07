package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBranchHandling tests branch name handling in different modes
func TestBranchHandling(t *testing.T) {
	tempDir := t.TempDir()
	setupTestRepo(tempDir, t)

	tests := map[string]struct {
		continueSession bool
		initialBranch   string
		expectBranch    string
		expectError     bool
	}{
		"continue mode with empty branch": {
			continueSession: true,
			initialBranch:   "",
			expectBranch:    "",
			expectError:     false,
		},
		"continue mode with specified branch": {
			continueSession: true,
			initialBranch:   "custom-branch",
			expectBranch:    "custom-branch", // Should keep specified branch
			expectError:     false,
		},
		"normal mode with empty branch": {
			continueSession: false,
			initialBranch:   "",
			// expectBranch not specified as it will use a timestamp-based name
			expectError: false,
		},
		"normal mode with specified branch": {
			continueSession: false,
			initialBranch:   "custom-branch",
			expectBranch:    "custom-branch", // Should keep specified branch
			expectError:     false,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			cfg := New()
			cfg.ContinueSession = test.continueSession
			cfg.RepoPath = tempDir
			cfg.BranchName = test.initialBranch

			err := cfg.Finalize()

			if test.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			} else if !test.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if name == "continue mode with empty branch" {
				// Get the actual branch name from the repo
				cmd := exec.Command("git", "-C", tempDir, "branch", "--show-current")
				output, err := cmd.Output()
				if err != nil {
					t.Fatalf("Failed to get current branch: %v", err)
				}
				expectedBranch := strings.TrimSpace(string(output))
				if cfg.BranchName != expectedBranch {
					t.Errorf("Expected BranchName to match current branch %s, got %s",
						expectedBranch, cfg.BranchName)
				}
			} else if test.expectBranch != "" {
				if cfg.BranchName != test.expectBranch {
					t.Errorf("Expected BranchName=%s, got %s",
						test.expectBranch, cfg.BranchName)
				}
			}

			// For normal mode with empty branch, just check that a branch name was generated
			if !test.continueSession && test.initialBranch == "" && cfg.BranchName == "" {
				t.Errorf("Expected a generated branch name, got empty string")
			}
		})
	}
}

// setupTestRepo initializes a git repository in the given directory for testing
func setupTestRepo(dir string, t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{"git", []string{"init", "--initial-branch=main", dir}},
		{"git", []string{"-C", dir, "config", "user.email", "test@example.com"}},
		{"git", []string{"-C", dir, "config", "user.name", "Test User"}},
	}

	for _, cmd := range commands {
		c := exec.Command(cmd.name, cmd.args...)
		err := c.Run()
		if err != nil {
			// If initializing with --initial-branch fails, try the standard way
			if cmd.args[0] == "init" && cmd.args[1] == "--initial-branch=main" {
				// Fall back to standard init without specifying branch
				fallbackCmd := exec.Command("git", "init", dir)
				if fallbackErr := fallbackCmd.Run(); fallbackErr != nil {
					t.Fatalf("Failed to initialize git repo: %v", fallbackErr)
				}
			} else {
				t.Fatalf("Failed to run %s %v: %v", cmd.name, cmd.args, err)
			}
		}
	}

	// Verify the branch was created correctly, or fall back if --initial-branch isn't supported
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "main" {
		// Older Git version or branch creation failed - create and checkout main explicitly
		mainBranchCmds := []struct {
			name string
			args []string
		}{
			{"git", []string{"-C", dir, "checkout", "-b", "main"}},
		}

		for _, cmd := range mainBranchCmds {
			c := exec.Command(cmd.name, cmd.args...)
			if err := c.Run(); err != nil {
				t.Fatalf("Failed to create main branch: %v", err)
			}
		}
	}

	initialFile := filepath.Join(dir, "initial.txt")
	writeErr := os.WriteFile(initialFile, []byte("Initial content"), 0644)
	if writeErr != nil {
		t.Fatalf("Failed to create initial file: %v", writeErr)
	}

	commitCommands := []struct {
		name string
		args []string
	}{
		{"git", []string{"-C", dir, "add", "initial.txt"}},
		{"git", []string{"-C", dir, "commit", "-m", "Initial commit"}},
	}

	for _, cmd := range commitCommands {
		c := exec.Command(cmd.name, cmd.args...)
		err := c.Run()
		if err != nil {
			t.Fatalf("Failed to run %s %v: %v", cmd.name, cmd.args, err)
		}
	}
}
