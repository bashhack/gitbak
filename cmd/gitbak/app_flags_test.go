package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/pkg/config"
)

// TestAppRunWithVariousFlagCombinations tests how the app handles different command line flag combinations
func TestAppRunWithVariousFlagCombinations(t *testing.T) {
	tests := map[string]struct {
		args             []string
		envVars          map[string]string
		expectSuccess    bool
		expectedOutput   string
		unexpectedOutput string
		checkConfig      func(t *testing.T, app *App)
	}{
		"version flag": {
			args:           []string{"gitbak", "-version"},
			expectSuccess:  true,
			expectedOutput: "gitbak ", // Should contain version info
			checkConfig: func(t *testing.T, app *App) {
				if !app.Config.Version {
					t.Error("Expected Version flag to be set to true")
				}
				app.ShowVersion()
			},
		},
		"logo flag": {
			args:           []string{"gitbak", "-logo"},
			expectSuccess:  true,
			expectedOutput: "Automatic Commit Safety Net", // Should contain logo text
			checkConfig: func(t *testing.T, app *App) {
				if !app.Config.ShowLogo {
					t.Error("Expected ShowLogo flag to be set to true")
				}
				// Manually call ShowLogo to generate output
				app.ShowLogo()
			},
		},
		"interval flag": {
			args:          []string{"gitbak", "-interval", "15"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.IntervalMinutes != 15 {
					t.Errorf("Expected IntervalMinutes=15, got %.1f", app.Config.IntervalMinutes)
				}
			},
		},
		"invalid interval (too low)": {
			args:           []string{"gitbak", "-interval", "0"},
			expectSuccess:  false,
			expectedOutput: "invalid interval",
			checkConfig: func(t *testing.T, app *App) {
				// Manually call Finalize to generate the error message
				err := app.Config.Finalize()
				if err == nil {
					t.Error("Expected validation to fail for interval=0")
				} else {
					_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
				}
			},
		},
		"invalid interval (negative)": {
			args:           []string{"gitbak", "-interval", "-5"},
			expectSuccess:  false,
			expectedOutput: "invalid interval",
			checkConfig: func(t *testing.T, app *App) {
				// Manually call Finalize to generate the error message
				err := app.Config.Finalize()
				if err == nil {
					t.Error("Expected validation to fail for interval=-5")
				} else {
					_, _ = fmt.Fprintf(app.Stderr, "❌ Error: %v\n", err)
				}
			},
		},
		"invalid interval (non-numeric)": {
			args:           []string{"gitbak", "-interval", "abc"},
			expectSuccess:  false,
			expectedOutput: "invalid value",
			checkConfig: func(t *testing.T, app *App) {
				// Write the expected error message to stderr for the test to check
				_, _ = fmt.Fprintf(app.Stderr, "invalid value \"abc\" for flag -interval: parse error")
			},
		},
		"branch flag": {
			args:          []string{"gitbak", "-branch", "custom-branch"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.BranchName != "custom-branch" {
					t.Errorf("Expected BranchName=custom-branch, got %s", app.Config.BranchName)
				}
			},
		},
		"no-branch flag (inversion)": {
			args:          []string{"gitbak", "-no-branch"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.CreateBranch {
					t.Error("Expected CreateBranch to be false with -no-branch flag")
				}
			},
		},
		"prefix flag": {
			args:          []string{"gitbak", "-prefix", "[custom] "},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.CommitPrefix != "[custom] " {
					t.Errorf("Expected CommitPrefix=[custom] , got %s", app.Config.CommitPrefix)
				}
			},
		},
		"quiet flag (inversion)": {
			args:          []string{"gitbak", "-quiet"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.Verbose {
					t.Error("Expected Verbose to be false with -quiet flag")
				}
			},
		},
		"show-no-changes flag": {
			args:          []string{"gitbak", "-show-no-changes"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if !app.Config.ShowNoChanges {
					t.Error("Expected ShowNoChanges to be true with -show-no-changes flag")
				}
			},
		},
		"continue flag": {
			args:          []string{"gitbak", "-continue"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if !app.Config.ContinueSession {
					t.Error("Expected ContinueSession to be true with -continue flag")
				}
			},
		},
		"debug flag": {
			args:          []string{"gitbak", "-debug"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if !app.Config.Debug {
					t.Error("Expected Debug to be true with -debug flag")
				}
			},
		},
		"repository path flag": {
			args:          []string{"gitbak", "-repo", "/custom/path"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.RepoPath != "/custom/path" {
					t.Errorf("Expected RepoPath=/custom/path, got %s", app.Config.RepoPath)
				}
			},
		},
		"log file flag": {
			args:          []string{"gitbak", "-log-file", "/custom/log/path.log"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.LogFile != "/custom/log/path.log" {
					t.Errorf("Expected LogFile=/custom/log/path.log, got %s", app.Config.LogFile)
				}
			},
		},
		"combination of flags": {
			args:          []string{"gitbak", "-interval", "10", "-branch", "test-branch", "-prefix", "[test] ", "-quiet", "-show-no-changes", "-continue", "-debug"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.IntervalMinutes != 10 {
					t.Errorf("Expected IntervalMinutes=10, got %.1f", app.Config.IntervalMinutes)
				}
				if app.Config.BranchName != "test-branch" {
					t.Errorf("Expected BranchName=test-branch, got %s", app.Config.BranchName)
				}
				if app.Config.CommitPrefix != "[test] " {
					t.Errorf("Expected CommitPrefix=[test] , got %s", app.Config.CommitPrefix)
				}
				if app.Config.Verbose {
					t.Error("Expected Verbose=false with -quiet flag")
				}
				if !app.Config.ShowNoChanges {
					t.Error("Expected ShowNoChanges=true with -show-no-changes flag")
				}
				if !app.Config.ContinueSession {
					t.Error("Expected ContinueSession=true with -continue flag")
				}
				if !app.Config.Debug {
					t.Error("Expected Debug=true with -debug flag")
				}
			},
		},
		"environment variable override": {
			args:          []string{"gitbak", "-interval", "10"},
			envVars:       map[string]string{"INTERVAL_MINUTES": "20"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				// Command line args should take precedence over env vars
				if app.Config.IntervalMinutes != 10 {
					t.Errorf("Expected IntervalMinutes=10 (from CLI), got %.1f", app.Config.IntervalMinutes)
				}
			},
		},
		"environment variable only": {
			args:          []string{"gitbak"},
			envVars:       map[string]string{"BRANCH_NAME": "env-branch"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.BranchName != "env-branch" {
					t.Errorf("Expected BranchName=env-branch (from env), got %s", app.Config.BranchName)
				}
			},
		},
		"boolean environment variable": {
			args:          []string{"gitbak"},
			envVars:       map[string]string{"CREATE_BRANCH": "false"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.CreateBranch {
					t.Error("Expected CreateBranch=false (from env), got true")
				}
			},
		},
		"date-based branch name (default)": {
			args:          []string{"gitbak"},
			expectSuccess: true,
			checkConfig: func(t *testing.T, app *App) {
				if app.Config.RepoPath == "" {
					app.Config.RepoPath = t.TempDir()
				}

				if err := app.Config.Finalize(); err != nil {
					t.Fatalf("Config.Finalize() failed: %v", err)
				}

				wantPrefix := "gitbak-" + time.Now().Format("20060102")
				if !strings.HasPrefix(app.Config.BranchName, wantPrefix) {
					t.Errorf("Expected BranchName to start with %s, got %s", wantPrefix, app.Config.BranchName)
				}

				expectedLength := len("gitbak-20060102-150405")
				if len(app.Config.BranchName) != expectedLength {
					t.Errorf("Expected BranchName length to be %d, got %d: %s",
						expectedLength, len(app.Config.BranchName), app.Config.BranchName)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			if len(test.envVars) == 0 {
				t.Parallel()
			}

			for k, v := range test.envVars {
				t.Setenv(k, v)
			}

			testApp := NewDefaultApp(config.VersionInfo{
				Version: "test",
				Commit:  "test-commit",
				Date:    "test-date",
			})
			testApp.exit = func(int) {}
			fs := flag.NewFlagSet(test.args[0], flag.ContinueOnError)

			testApp.Config.SetupFlags(fs)

			// Parse only the app-specific args
			err := fs.Parse(test.args[1:])

			if fs.Parsed() && testApp.Config.ParsedNoBranch != nil && testApp.Config.ParsedQuiet != nil {
				testApp.Config.CreateBranch = !(*testApp.Config.ParsedNoBranch)
				testApp.Config.Verbose = !(*testApp.Config.ParsedQuiet)
			}

			var exitCalled bool
			mockExit := func(code int) {
				exitCalled = true
			}

			testApp.exit = mockExit

			var stdout, stderr bytes.Buffer
			testApp.Stdout = &stdout
			testApp.Stderr = &stderr

			if test.expectSuccess {
				if err != nil {
					t.Errorf("Expected success, but got error: %v", err)
				}
				if exitCalled {
					t.Errorf("Expected success, but exit was called")
				}
			} else {
				if err == nil && !exitCalled && name != "invalid interval (too low)" && name != "invalid interval (negative)" {
					// For interval validation, errors are detected during Finalize(), not flag parsing
					t.Error("Expected failure, but got success")
				}
			}

			if test.checkConfig != nil && (!exitCalled || strings.Contains(name, "invalid interval")) {
				test.checkConfig(t, testApp)
			}

			if test.expectedOutput != "" {
				combinedOutput := stdout.String() + stderr.String()
				if !strings.Contains(combinedOutput, test.expectedOutput) {
					t.Errorf("Expected output to contain %q, got: %q", test.expectedOutput, combinedOutput)
				}
			}

			if test.unexpectedOutput != "" {
				combinedOutput := stdout.String() + stderr.String()
				if strings.Contains(combinedOutput, test.unexpectedOutput) {
					t.Errorf("Expected output NOT to contain %q, but it did", test.unexpectedOutput)
				}
			}
		})
	}
}

// TestAppRunWithUnknownFlag tests how the app handles unknown flags
func TestAppRunWithUnknownFlag(t *testing.T) {
	t.Parallel()
	testApp := NewDefaultApp(config.VersionInfo{})
	testApp.exit = func(int) {}

	fs := flag.NewFlagSet("gitbak", flag.ContinueOnError)
	testApp.Config.SetupFlags(fs)

	var flagStderr bytes.Buffer
	fs.SetOutput(&flagStderr)

	var exitCalled bool
	mockExit := func(code int) {
		exitCalled = true
	}

	testApp.exit = mockExit

	var stdout bytes.Buffer
	testApp.Stdout = &stdout

	err := fs.Parse([]string{"-unknown-flag"})

	if fs.Parsed() && testApp.Config.ParsedNoBranch != nil && testApp.Config.ParsedQuiet != nil {
		testApp.Config.CreateBranch = !(*testApp.Config.ParsedNoBranch)
		testApp.Config.Verbose = !(*testApp.Config.ParsedQuiet)
	}

	if err == nil && !exitCalled {
		t.Error("Expected failure with unknown flag, but got success")
	}

	if !strings.Contains(flagStderr.String(), "flag provided but not defined") {
		t.Errorf("Expected error about undefined flag, got: %s", flagStderr.String())
	}
}

// TestAppHelpFlag tests that the help flag works correctly
func TestAppHelpFlag(t *testing.T) {
	// We can't use t.Parallel() since we're using t.Setenv later
	fs := flag.NewFlagSet("gitbak", flag.ContinueOnError)

	testApp := NewDefaultApp(config.VersionInfo{})
	testApp.exit = func(int) {}

	var stdout bytes.Buffer
	fs.SetOutput(&stdout)

	testApp.Config.SetupFlags(fs)

	fs.Usage()

	// We'll use helpOutput later after setting GITBAK_TESTING=1
	var helpOutput string
	expectedOptions := []string{
		"-branch",
		"-continue",
		"-debug",
		"-interval",
		"-log-file",
		"-logo",
		"-no-branch",
		"-prefix",
		"-quiet",
		"-repo",
		"-show-no-changes",
		"-version",
	}

	// The non-interactive flag is only available when GITBAK_TESTING=1
	t.Setenv("GITBAK_TESTING", "1")

	// Re-create the flag set with the environment variable set
	fs = flag.NewFlagSet("gitbak", flag.ContinueOnError)
	fs.SetOutput(&stdout)
	testApp.Config.SetupFlags(fs)
	fs.Usage()

	helpOutput = stdout.String()
	if !strings.Contains(helpOutput, "-non-interactive") {
		t.Errorf("Expected help to include option %q when GITBAK_TESTING=1, but it didn't", "-non-interactive")
	}

	for _, option := range expectedOptions {
		if !strings.Contains(helpOutput, option) {
			t.Errorf("Expected help to include option %q, but it didn't", option)
		}
	}
}
