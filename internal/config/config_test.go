package config

import (
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
)

// Helper function to check for environment variable errors
func checkEnvErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	c := New()

	if c.IntervalMinutes != DefaultIntervalMinutes {
		t.Errorf("Expected IntervalMinutes=%.1f, got %.1f", DefaultIntervalMinutes, c.IntervalMinutes)
	}
	if c.CommitPrefix != DefaultCommitPrefix {
		t.Errorf("Expected CommitPrefix=%s, got %s", DefaultCommitPrefix, c.CommitPrefix)
	}
	if !c.CreateBranch {
		t.Errorf("Expected CreateBranch=true, got false")
	}
	if !c.Verbose {
		t.Errorf("Expected Verbose=true, got false")
	}
	if c.ShowNoChanges {
		t.Errorf("Expected ShowNoChanges=false, got true")
	}
	if c.ContinueSession {
		t.Errorf("Expected ContinueSession=false, got true")
	}
	if c.Debug {
		t.Errorf("Expected Debug=false, got true")
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	checkEnvErr(t, os.Setenv("INTERVAL_MINUTES", "10"))
	checkEnvErr(t, os.Setenv("BRANCH_NAME", "test-branch"))
	checkEnvErr(t, os.Setenv("COMMIT_PREFIX", "[test] Commit"))
	checkEnvErr(t, os.Setenv("CREATE_BRANCH", "false"))
	checkEnvErr(t, os.Setenv("VERBOSE", "false"))
	checkEnvErr(t, os.Setenv("SHOW_NO_CHANGES", "true"))
	checkEnvErr(t, os.Setenv("REPO_PATH", "/tmp/test-repo"))
	checkEnvErr(t, os.Setenv("CONTINUE_SESSION", "true"))
	checkEnvErr(t, os.Setenv("DEBUG", "true"))
	checkEnvErr(t, os.Setenv("LOG_FILE", "/tmp/test.log"))

	defer func() {
		checkEnvErr(t, os.Unsetenv("INTERVAL_MINUTES"))
		checkEnvErr(t, os.Unsetenv("BRANCH_NAME"))
		checkEnvErr(t, os.Unsetenv("COMMIT_PREFIX"))
		checkEnvErr(t, os.Unsetenv("CREATE_BRANCH"))
		checkEnvErr(t, os.Unsetenv("VERBOSE"))
		checkEnvErr(t, os.Unsetenv("SHOW_NO_CHANGES"))
		checkEnvErr(t, os.Unsetenv("REPO_PATH"))
		checkEnvErr(t, os.Unsetenv("CONTINUE_SESSION"))
		checkEnvErr(t, os.Unsetenv("DEBUG"))
		checkEnvErr(t, os.Unsetenv("LOG_FILE"))
	}()

	c := New()
	c.LoadFromEnvironment()

	if c.IntervalMinutes != 10 {
		t.Errorf("Expected IntervalMinutes=10, got %.1f", c.IntervalMinutes)
	}
	if c.BranchName != "test-branch" {
		t.Errorf("Expected BranchName=test-branch, got %s", c.BranchName)
	}
	if c.CommitPrefix != "[test] Commit" {
		t.Errorf("Expected CommitPrefix=[test] Commit, got %s", c.CommitPrefix)
	}
	if c.CreateBranch {
		t.Errorf("Expected CreateBranch=false, got true")
	}
	if c.Verbose {
		t.Errorf("Expected Verbose=false, got true")
	}
	if !c.ShowNoChanges {
		t.Errorf("Expected ShowNoChanges=true, got false")
	}
	if c.RepoPath != "/tmp/test-repo" {
		t.Errorf("Expected RepoPath=/tmp/test-repo, got %s", c.RepoPath)
	}
	if !c.ContinueSession {
		t.Errorf("Expected ContinueSession=true, got false")
	}
	if !c.Debug {
		t.Errorf("Expected Debug=true, got false")
	}
	if c.LogFile != "/tmp/test.log" {
		t.Errorf("Expected LogFile=/tmp/test.log, got %s", c.LogFile)
	}
}

func TestSetupFlags(t *testing.T) {
	t.Parallel()
	c := New()
	c.BranchName = "env-branch" // Set a value to check override

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	c.SetupFlags(fs)

	args := []string{
		"-interval", "15",
		"-branch", "flag-branch",
		"-debug",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if c.IntervalMinutes != 15 {
		t.Errorf("Expected IntervalMinutes=15, got %.1f", c.IntervalMinutes)
	}
	if c.BranchName != "flag-branch" {
		t.Errorf("Expected BranchName=flag-branch, got %s", c.BranchName)
	}
	if !c.Debug {
		t.Errorf("Expected Debug=true, got false")
	}
}

// TestParseFlags tests the ParseFlags method which parses CLI args
func TestParseFlags(t *testing.T) {
	t.Run("Basic flags", func(t *testing.T) {
		originalArgs := os.Args
		defer func() { os.Args = originalArgs }()

		os.Args = []string{"gitbak", "-interval", "30", "-branch", "test-branch", "-prefix", "[custom] "}

		c := New()

		err := c.ParseFlags()

		if err != nil {
			t.Errorf("ParseFlags() error = %v, expected no error", err)
			return
		}

		if c.IntervalMinutes != 30 {
			t.Errorf("Expected IntervalMinutes=30, got %.1f", c.IntervalMinutes)
		}
		if c.BranchName != "test-branch" {
			t.Errorf("Expected BranchName=test-branch, got %s", c.BranchName)
		}
		if c.CommitPrefix != "[custom] " {
			t.Errorf("Expected CommitPrefix=[custom] , got %s", c.CommitPrefix)
		}
	})

	t.Run("Boolean flags", func(t *testing.T) {
		c := New()

		fs := flag.NewFlagSet("test", flag.ContinueOnError)

		fs.BoolVar(&c.CreateBranch, "no-branch", !c.CreateBranch, "")
		fs.BoolVar(&c.Verbose, "quiet", !c.Verbose, "")
		fs.BoolVar(&c.ShowNoChanges, "show-no-changes", c.ShowNoChanges, "")

		err := fs.Parse([]string{"-no-branch", "-quiet", "-show-no-changes"})
		if err != nil {
			t.Errorf("Flag parsing failed: %v", err)
			return
		}

		c.CreateBranch = !c.CreateBranch
		c.Verbose = !c.Verbose

		if c.CreateBranch != false {
			t.Errorf("Expected CreateBranch=false, got %v", c.CreateBranch)
		}
		if c.Verbose != false {
			t.Errorf("Expected Verbose=false, got %v", c.Verbose)
		}
		if c.ShowNoChanges != true {
			t.Errorf("Expected ShowNoChanges=true, got %v", c.ShowNoChanges)
		}
	})

	t.Run("Version flag", func(t *testing.T) {
		originalArgs := os.Args
		defer func() { os.Args = originalArgs }()

		os.Args = []string{"gitbak", "-version"}

		c := New()

		err := c.ParseFlags()

		if err != nil {
			t.Errorf("ParseFlags() error = %v, expected no error", err)
			return
		}

		if c.Version != true {
			t.Errorf("Expected Version=true, got %v", c.Version)
		}
	})

	t.Run("Logo flag", func(t *testing.T) {
		originalArgs := os.Args
		defer func() { os.Args = originalArgs }()

		os.Args = []string{"gitbak", "-logo"}

		c := New()

		err := c.ParseFlags()

		if err != nil {
			t.Errorf("ParseFlags() error = %v, expected no error", err)
			return
		}

		if c.ShowLogo != true {
			t.Errorf("Expected ShowLogo=true, got %v", c.ShowLogo)
		}
	})

	t.Run("Invalid interval", func(t *testing.T) {
		invalidArgs := []string{"gitbak", "-interval", "invalid"}

		c := New()

		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		c.SetupFlags(fs)

		err := fs.Parse(invalidArgs[1:])
		if err == nil {
			t.Errorf("Expected error for invalid interval, got nil")
		}
	})

	t.Run("Invalid flag", func(t *testing.T) {
		originalArgs := os.Args
		originalStdout := os.Stdout

		defer func() {
			os.Args = originalArgs
			os.Stdout = originalStdout
		}()

		r, w, _ := os.Pipe()
		os.Stdout = w

		os.Args = []string{"gitbak", "-invalid-flag"}

		c := New()
		err := c.ParseFlags()

		if err := w.Close(); err != nil {
			t.Errorf("Error closing pipe: %v", err)
		}
		os.Stdout = originalStdout

		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if err == nil {
			t.Errorf("Expected error for invalid flag, got nil")
		}

		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected error output to contain custom output 'Error:', got: %s", output)
		}

		if !strings.Contains(output, "gitbak: An automatic commit safety net") {
			t.Errorf("Expected custom help format with 'gitbak: An automatic commit safety net', got: %s", output)
		}

		if !gitbakErrors.Is(err, gitbakErrors.ErrInvalidFlag) {
			t.Errorf("Expected error to be ErrInvalidFlag, got: %v", err)
		}
	})

	t.Run("Usage printing", func(t *testing.T) {
		c := New()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		c.SetupFlags(fs)

		var buf strings.Builder
		c.PrintUsage(fs, &buf)
		output := buf.String()

		if !strings.Contains(output, "gitbak: An automatic commit safety net") {
			t.Errorf("Expected help message to contain 'gitbak: An automatic commit safety net', got: %s", output)
		}

		if !strings.Contains(output, "Core Options:") {
			t.Errorf("Expected help message to contain 'Core Options:', got: %s", output)
		}

		if !strings.Contains(output, "Environment variables:") {
			t.Errorf("Expected help message to contain 'Environment variables:', got: %s", output)
		}
	})
}

func TestFinalize(t *testing.T) {
	t.Parallel()
	c := New()
	c.IntervalMinutes = 0 // Invalid value

	err := c.Finalize()
	if err == nil {
		t.Errorf("Expected error for invalid interval, got nil")
	}
	if !strings.Contains(err.Error(), "invalid interval") {
		t.Errorf("Expected 'invalid interval' error, got: %v", err)
	}

	// Set valid values
	c.IntervalMinutes = 5
	c.RepoPath = "" // Should use the current directory
	c.LogFile = ""  // Should use XDG base directory

	err = c.Finalize()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if c.RepoPath == "" {
		t.Errorf("Expected RepoPath to be set to current directory, got empty string")
	}

	if c.LogFile == "" {
		t.Errorf("Expected LogFile to be set, got empty string")
	}

	if strings.Contains(c.LogFile, c.RepoPath) {
		t.Errorf("Expected LogFile to be outside the repository, got %s", c.LogFile)
	}

	homeDir, _ := os.UserHomeDir()
	xdgPattern := filepath.Join(homeDir, ".local", "share", "gitbak", "logs")
	xdgEnv := os.Getenv("XDG_DATA_HOME")

	if !strings.Contains(c.LogFile, xdgPattern) &&
		(xdgEnv == "" || !strings.Contains(c.LogFile, filepath.Join(xdgEnv, "gitbak", "logs"))) {
		t.Errorf("Expected LogFile to follow XDG Base Directory Specification, got %s", c.LogFile)
	}
}

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

func TestGetEnvInt(t *testing.T) {
	if err := os.Setenv("TEST_INT", "42"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_INT"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()

	result := getEnvInt("TEST_INT", 0)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	if err := os.Setenv("TEST_INVALID_INT", "not-an-int"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_INVALID_INT"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}()

	result = getEnvInt("TEST_INVALID_INT", 100)
	if result != 100 {
		t.Errorf("Expected default value 100, got %d", result)
	}

	result = getEnvInt("TEST_MISSING_INT", 200)
	if result != 200 {
		t.Errorf("Expected default value 200, got %d", result)
	}
}

func TestSetupTestFlags(t *testing.T) {
	oldEnv := os.Getenv("GITBAK_TESTING")
	if err := os.Setenv("GITBAK_TESTING", "1"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	defer func() {
		if err := os.Setenv("GITBAK_TESTING", oldEnv); err != nil {
			t.Logf("Failed to restore environment variable: %v", err)
		}
	}()

	c := New()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	c.SetupTestFlags(fs)

	flagVal := fs.Lookup("non-interactive")
	if flagVal == nil {
		t.Error("Expected non-interactive flag to be set")
	}

	if err := os.Unsetenv("GITBAK_TESTING"); err != nil {
		t.Fatalf("Failed to unset environment variable: %v", err)
	}

	c = New()
	newFs := flag.NewFlagSet("test", flag.ContinueOnError)

	c.SetupTestFlags(newFs)

	flagVal = newFs.Lookup("non-interactive")
	if flagVal != nil {
		t.Error("Expected non-interactive flag not to be set")
	}
}
