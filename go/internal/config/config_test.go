package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to check for environment variable errors
func checkEnvErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	c := New()

	// Verify default values
	if c.IntervalMinutes != DefaultIntervalMinutes {
		t.Errorf("Expected IntervalMinutes=%d, got %d", DefaultIntervalMinutes, c.IntervalMinutes)
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

	// Verify values from environment
	if c.IntervalMinutes != 10 {
		t.Errorf("Expected IntervalMinutes=10, got %d", c.IntervalMinutes)
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
		t.Errorf("Expected IntervalMinutes=15, got %d", c.IntervalMinutes)
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
			t.Errorf("Expected IntervalMinutes=30, got %d", c.IntervalMinutes)
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

	// Test error case separately for an invalid interval
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
}

func TestFinalize(t *testing.T) {
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
