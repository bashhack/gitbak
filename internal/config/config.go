package config

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
)

const (
	// DefaultIntervalMinutes is the default time (in minutes) between checking for changes.
	// This interval determines how frequently gitbak looks for changes to commit.
	// Lower values provide more frequent checkpoints but can increase resource usage.
	// The value can be a fractional number (e.g., 0.5 for 30 seconds).
	DefaultIntervalMinutes = 5.0

	// DefaultCommitPrefix is the default prefix added to all commit messages.
	// This prefix helps identify commits made by gitbak and is used to extract
	// commit numbers when continuing a session. The commit message format is:
	// "[gitbak] Automatic checkpoint #N (YYYY-MM-DD HH:MM:SS)"
	DefaultCommitPrefix = "[gitbak] Automatic checkpoint"

	// DefaultMaxRetries is the default number of consecutive identical errors
	// allowed before gitbak exits. A value of 0 means retry indefinitely.
	// The error counter resets when errors change or successful operations occur.
	DefaultMaxRetries = 3
)

// Config holds all gitbak application settings.
// This struct combines settings from command-line flags, environment variables,
// and default values to control all aspects of gitbak behavior.
type Config struct {
	// Repository configuration

	// RepoPath is the path to the Git repository to monitor.
	// If empty, the current working directory is used.
	RepoPath string

	// IntervalMinutes is how often (in minutes) to check for changes.
	// Uses float64 to support fractional minutes (e.g., 0.5 for 30 seconds).
	IntervalMinutes float64

	// BranchName is the Git branch to use for checkpoint commits.
	// If empty and CreateBranch is true, a timestamp-based name is generated.
	BranchName string

	// CommitPrefix is prepended to all commit messages.
	// Used to identify gitbak commits and extract commit numbers.
	CommitPrefix string

	// CreateBranch determines whether to create a new branch or use existing one.
	// If true, a new branch named BranchName will be created.
	CreateBranch bool

	// ContinueSession enables continuation mode for resuming a previous session.
	// When true, gitbak finds the last commit number and continues numbering from there.
	ContinueSession bool

	// User experience options

	// Verbose controls the amount of informational output.
	// When true, gitbak provides detailed status updates.
	Verbose bool

	// ShowNoChanges determines whether to report when no changes are detected.
	// When true, gitbak logs a message at each interval even if nothing changed.
	ShowNoChanges bool

	// NonInteractive disables any prompts and uses default responses.
	// Useful for running gitbak in automated environments.
	NonInteractive bool

	// Error handling options

	// MaxRetries defines how many consecutive identical errors are allowed before exiting.
	// A value of 0 means retry indefinitely.
	// Errors of different types or successful operations reset this counter.
	MaxRetries int

	// Debugging options

	// Debug enables detailed logging.
	// When true, gitbak logs additional information helpful for troubleshooting.
	Debug bool

	// LogFile specifies where to write debug logs.
	// If empty, logs are written to a default location based on repository path.
	LogFile string

	// Special flags

	// Version indicates whether to show version information and exit.
	// When true, gitbak prints version details and exits without running.
	Version bool

	// ShowLogo indicates whether to display the ASCII logo and exit.
	// When true, gitbak prints the logo and exits without running.
	ShowLogo bool

	// ShowHelp indicates whether to display the help message and exit.
	// When true, gitbak prints help information and exits without running.
	ShowHelp bool

	// Build metadata

	// VersionInfo contains version, commit, and build date information.
	// This is typically injected at build time.
	VersionInfo VersionInfo

	// Flag parsing state (exported for testing)

	// ParsedNoBranch tracks the state of the -no-branch flag.
	// Used during flag parsing to handle flag inversion.
	ParsedNoBranch *bool

	// ParsedQuiet tracks the state of the -quiet flag.
	// Used during flag parsing to handle flag inversion.
	ParsedQuiet *bool
}

// VersionInfo contains build-time version metadata.
// This information is typically injected during the build process
// and is used to display version information to users.
type VersionInfo struct {
	// Version is the semantic version number (e.g., "v1.2.3").
	Version string

	// Commit is the Git commit hash from which the binary was built.
	Commit string

	// Date is the build timestamp in human-readable format.
	Date string
}

// New creates a new Config with default values
func New() *Config {
	return &Config{
		IntervalMinutes: DefaultIntervalMinutes,
		CommitPrefix:    DefaultCommitPrefix,
		CreateBranch:    true,
		Verbose:         true,
		ShowNoChanges:   false,
		RepoPath:        "",
		ContinueSession: false,
		Debug:           false,
		LogFile:         "",
		Version:         false,
		ShowLogo:        false,
		ShowHelp:        false,
		MaxRetries:      DefaultMaxRetries,

		// Default version info, will be overridden if provided
		VersionInfo: VersionInfo{
			Version: "dev",
			Commit:  "unknown",
			Date:    "unknown",
		},
	}
}

// LoadFromEnvironment updates config from environment variables
func (c *Config) LoadFromEnvironment() {
	c.IntervalMinutes = getEnvFloat("INTERVAL_MINUTES", c.IntervalMinutes)
	c.BranchName = getEnvString("BRANCH_NAME", c.BranchName)
	c.CommitPrefix = getEnvString("COMMIT_PREFIX", c.CommitPrefix)
	c.CreateBranch = getEnvBool("CREATE_BRANCH", c.CreateBranch)
	c.Verbose = getEnvBool("VERBOSE", c.Verbose)
	c.NonInteractive = getEnvBool("NON_INTERACTIVE", c.NonInteractive)
	c.ShowNoChanges = getEnvBool("SHOW_NO_CHANGES", c.ShowNoChanges)
	c.RepoPath = getEnvString("REPO_PATH", c.RepoPath)
	c.ContinueSession = getEnvBool("CONTINUE_SESSION", c.ContinueSession)
	c.Debug = getEnvBool("DEBUG", c.Debug)
	c.LogFile = getEnvString("LOG_FILE", c.LogFile)
	c.MaxRetries = getEnvInt("MAX_RETRIES", c.MaxRetries)
}

// SetupFlags sets up command-line flags to override config values
func (c *Config) SetupFlags(fs *flag.FlagSet) {
	// Separate variables for inverted flags
	var noBranch bool
	var quiet bool

	// Define command-line flags
	fs.Float64Var(&c.IntervalMinutes, "interval", c.IntervalMinutes, "Minutes between commits (supports decimal values like 0.1 for 6 seconds)")
	fs.StringVar(&c.BranchName, "branch", c.BranchName, "Custom branch name (default: gitbak-{timestamp})")
	fs.StringVar(&c.CommitPrefix, "prefix", c.CommitPrefix, "Custom commit message prefix")
	fs.BoolVar(&noBranch, "no-branch", !c.CreateBranch, "Use current branch instead of creating a new one")
	fs.BoolVar(&quiet, "quiet", !c.Verbose, "Hide informational messages")
	fs.BoolVar(&c.ShowNoChanges, "show-no-changes", c.ShowNoChanges, "Show messages when no changes detected")
	fs.StringVar(&c.RepoPath, "repo", c.RepoPath, "Path to repository (default: current directory)")
	fs.BoolVar(&c.ContinueSession, "continue", c.ContinueSession, "Continue from existing branch")
	fs.BoolVar(&c.Debug, "debug", c.Debug, "Enable debug logging")
	fs.StringVar(&c.LogFile, "log-file", c.LogFile, "Path to log file (default: ~/.local/share/gitbak/logs/gitbak-{repo-hash}.log)")
	fs.BoolVar(&c.Version, "version", c.Version, "Print version information and exit")
	fs.BoolVar(&c.ShowLogo, "logo", c.ShowLogo, "Display ASCII logo and exit")
	fs.BoolVar(&c.ShowHelp, "help", c.ShowHelp, "Display help message and exit")
	fs.IntVar(&c.MaxRetries, "max-retries", c.MaxRetries, "Maximum consecutive identical errors before quitting (0 = unlimited)")

	// Add test-specific flags if we're in a test build
	// This calls the appropriate function based on build tags
	// or the GITBAK_TESTING environment variable (for compatibility)
	c.SetupTestFlags(fs)

	// Store the temporary values for later use after successful parsing
	c.ParsedNoBranch = &noBranch
	c.ParsedQuiet = &quiet
}

// PrintUsage prints a formatted help message with command descriptions, examples, and grouped flags
func (c *Config) PrintUsage(fs *flag.FlagSet, w io.Writer) {
	programName := filepath.Base(os.Args[0])

	_, _ = fmt.Fprintf(w, "gitbak: An automatic commit safety net\n\n")
	_, _ = fmt.Fprintf(w, "Usage: %s [options]\n\n", programName)
	_, _ = fmt.Fprintf(w, "gitbak automatically creates checkpoint commits at regular intervals,\n")
	_, _ = fmt.Fprintf(w, "providing protection against accidental code loss during programming sessions.\n\n")

	_, _ = fmt.Fprintf(w, "Examples:\n")
	_, _ = fmt.Fprintf(w, "  %s                                    # Run with defaults (5-minute interval)\n", programName)
	_, _ = fmt.Fprintf(w, "  %s -interval 1                        # Commit every minute\n", programName)
	_, _ = fmt.Fprintf(w, "  %s -interval 0.1 -prefix \"[pair]\"   # Commit every 6 seconds with custom prefix\n", programName)
	_, _ = fmt.Fprintf(w, "  %s -branch feature-backup -no-branch  # Use existing branch instead of creating\n", programName)
	_, _ = fmt.Fprintf(w, "  %s -continue                          # Continue numbering from previous session\n\n", programName)

	// Group flags by category
	_, _ = fmt.Fprintf(w, "Core Options:\n")
	printFlagIfExists(w, fs, "interval")
	printFlagIfExists(w, fs, "branch")
	printFlagIfExists(w, fs, "prefix")
	printFlagIfExists(w, fs, "no-branch")
	printFlagIfExists(w, fs, "repo")
	printFlagIfExists(w, fs, "continue")
	_, _ = fmt.Fprintf(w, "\n")

	_, _ = fmt.Fprintf(w, "Output Options:\n")
	printFlagIfExists(w, fs, "quiet")
	printFlagIfExists(w, fs, "show-no-changes")
	printFlagIfExists(w, fs, "debug")
	printFlagIfExists(w, fs, "log-file")
	_, _ = fmt.Fprintf(w, "\n")

	_, _ = fmt.Fprintf(w, "Error Handling:\n")
	printFlagIfExists(w, fs, "max-retries")
	_, _ = fmt.Fprintf(w, "\n")

	_, _ = fmt.Fprintf(w, "Information:\n")
	printFlagIfExists(w, fs, "version")
	printFlagIfExists(w, fs, "logo")
	printFlagIfExists(w, fs, "help")
	_, _ = fmt.Fprintf(w, "\n")

	_, _ = fmt.Fprintf(w, "Environment variables:\n")
	_, _ = fmt.Fprintf(w, "  INTERVAL_MINUTES          Minutes between commits (supports decimal values)\n")
	_, _ = fmt.Fprintf(w, "  BRANCH_NAME               Branch name to use\n")
	_, _ = fmt.Fprintf(w, "  COMMIT_PREFIX             Custom prefix for commit messages\n")
	_, _ = fmt.Fprintf(w, "  CREATE_BRANCH             Whether to create a new branch (true/false)\n")
	_, _ = fmt.Fprintf(w, "  VERBOSE                   Whether to show informational messages (true/false)\n")
	_, _ = fmt.Fprintf(w, "  SHOW_NO_CHANGES           Whether to show 'no changes' messages (true/false)\n")
	_, _ = fmt.Fprintf(w, "  REPO_PATH                 Path to repository\n")
	_, _ = fmt.Fprintf(w, "  CONTINUE_SESSION          Whether to continue from existing branch (true/false)\n")
	_, _ = fmt.Fprintf(w, "  DEBUG                     Enable debug logging (true/false)\n")
	_, _ = fmt.Fprintf(w, "  LOG_FILE                  Path to log file\n")
	_, _ = fmt.Fprintf(w, "  MAX_RETRIES               Maximum consecutive identical errors before quitting\n")
}

// printFlagIfExists prints a flag's usage if it exists in the FlagSet
func printFlagIfExists(w io.Writer, fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		return
	}

	// Format: -flag [type] (default: value): description
	defaultValue := f.DefValue
	if defaultValue != "" {
		defaultValue = fmt.Sprintf(" (default: %s)", defaultValue)
	}

	_, _ = fmt.Fprintf(w, "  -%s%s: %s\n", f.Name, defaultValue, f.Usage)
}

// ParseFlags parses the command-line arguments and updates the config
func (c *Config) ParseFlags() error {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-help" || arg == "-h" || arg == "--h" {
			// Create a fake FlagSet to set up flags for help display
			fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			c.SetupFlags(fs)

			c.PrintUsage(fs, os.Stdout)
			os.Exit(0)
		}
	}

	// Create a flag set with custom error handling to suppress
	// the initial error message from flag package
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Redirect stderr temporarily to capture and discard the flag parse error message
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		if err := w.Close(); err != nil {
			_, _ = fmt.Fprintf(oldStderr, "Error closing pipe: %v\n", err)
		}
		os.Stderr = oldStderr
	}()

	// Prevent default flag usage output
	fs.Usage = func() {}

	c.SetupFlags(fs)

	// For simplicity, parse whatever arguments were passed to the program.
	// If running under 'go test', the test binary will handle test flags separately.
	var appArgs []string
	// Skip the program name (os.Args[0])
	if len(os.Args) > 1 {
		appArgs = os.Args[1:]
	}

	if err := fs.Parse(appArgs); err != nil {
		helpFS := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		c.SetupFlags(helpFS)

		fmt.Printf("Error: %s\n\n", err)

		c.PrintUsage(helpFS, os.Stdout)

		return gitbakErrors.NewConfigError("flags", nil, gitbakErrors.Wrap(gitbakErrors.ErrInvalidFlag, err.Error()))
	}

	// Apply inverted flags only after successful parsing
	if c.ParsedNoBranch != nil {
		c.CreateBranch = !(*c.ParsedNoBranch)
	}
	if c.ParsedQuiet != nil {
		c.Verbose = !(*c.ParsedQuiet)
	}

	return nil
}

// Finalize validates and finalizes the configuration
func (c *Config) Finalize() error {
	if c.IntervalMinutes <= 0 {
		err := fmt.Errorf("invalid interval: %.2f (must be greater than 0)", c.IntervalMinutes)
		return gitbakErrors.NewConfigError("interval", c.IntervalMinutes, gitbakErrors.Wrap(err, "invalid interval"))
	}

	if c.RepoPath == "" {
		var err error
		c.RepoPath, err = os.Getwd()
		if err != nil {
			return gitbakErrors.NewConfigError("repoPath", "", gitbakErrors.Wrap(err, "failed to get current directory"))
		}
	}

	absRepoPath, err := filepath.Abs(c.RepoPath)
	if err != nil {
		return gitbakErrors.NewConfigError("repoPath", c.RepoPath, gitbakErrors.Wrap(err, "failed to resolve absolute path"))
	}
	c.RepoPath = absRepoPath

	if c.LogFile == "" {
		// Follow XDG Base Directory Specification
		logDir := os.Getenv("XDG_DATA_HOME")
		if logDir == "" {
			// Default XDG data home if not set
			homeDir, err := os.UserHomeDir()
			if err == nil {
				logDir = filepath.Join(homeDir, ".local", "share")
			} else {
				logDir = os.TempDir()
			}
		}

		repoHash := fmt.Sprintf("%x", sha256OfString(c.RepoPath)[:8])

		gitbakLogDir := filepath.Join(logDir, "gitbak", "logs")
		c.LogFile = filepath.Join(gitbakLogDir, fmt.Sprintf("gitbak-%s.log", repoHash))

		if err := os.MkdirAll(filepath.Dir(c.LogFile), 0o700); err != nil {
			return gitbakErrors.NewConfigError("logFile", c.LogFile, gitbakErrors.Wrap(err, "cannot create log directory"))
		}
	}

	if c.BranchName == "" {
		if c.ContinueSession {
			currentBranch, err := getCurrentBranchName(c.RepoPath)
			if err != nil {
				return gitbakErrors.NewConfigError("branchName", "",
					gitbakErrors.Wrap(err, "failed to get current branch name in continue mode"))
			}
			c.BranchName = currentBranch
		} else {
			timestamp := time.Now().Format("20060102-150405")
			c.BranchName = fmt.Sprintf("gitbak-%s", timestamp)
		}
	}

	return nil
}

// getEnvString returns an environment variable string or a default value
func getEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt returns an environment variable as int or a default value
func getEnvInt(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvFloat returns an environment variable as float64 or a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvBool returns an environment variable as bool or a default value
func getEnvBool(key string, defaultValue bool) bool {
	if valueStr, exists := os.LookupEnv(key); exists {
		valueLower := strings.ToLower(valueStr)
		if valueLower == "true" || valueLower == "1" || valueLower == "yes" {
			return true
		}
		if valueLower == "false" || valueLower == "0" || valueLower == "no" {
			return false
		}
		// For any other value, fall back to default
	}
	return defaultValue
}

// sha256OfString returns the SHA256 hash of a string
func sha256OfString(input string) []byte {
	hash := sha256.Sum256([]byte(input))
	return hash[:]
}

// getCurrentBranchName gets the current git branch name for a repository
func getCurrentBranchName(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetupTestFlags conditionally adds test-specific flags to the flag set.
func (c *Config) SetupTestFlags(fs *flag.FlagSet) {
	if os.Getenv("GITBAK_TESTING") == "1" {
		fs.BoolVar(&c.NonInteractive, "non-interactive", c.NonInteractive,
			"Skip all interactive prompts (for testing automation)")
	}
}
