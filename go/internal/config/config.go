package config

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bashhack/gitbak/internal/errors"
)

const (
	// DefaultIntervalMinutes between commits in minutes
	DefaultIntervalMinutes = 5.0

	// DefaultCommitPrefix for commit messages
	DefaultCommitPrefix = "[gitbak] Automatic checkpoint"
)

// Config holds all gitbak application settings
type Config struct {
	// Repository configuration
	RepoPath        string
	IntervalMinutes float64 // Using float64 to support fractional minutes
	BranchName      string
	CommitPrefix    string
	CreateBranch    bool
	ContinueSession bool

	// User experience
	Verbose        bool
	ShowNoChanges  bool
	NonInteractive bool // Skips interactive prompts

	// Error handling
	MaxRetries int // Maximum number of consecutive identical errors before quitting (0 = unlimited)

	// Debugging
	Debug   bool
	LogFile string

	// Special flags
	Version  bool
	ShowLogo bool // Shows ASCII logo and exits

	// Build metadata
	VersionInfo VersionInfo

	// Flag parsing state (exported for testing)
	ParsedNoBranch *bool
	ParsedQuiet    *bool
}

// VersionInfo contains build-time version metadata
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
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
		MaxRetries:      3, // Default to 3 consecutive identical errors before quitting

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
	fs.IntVar(&c.MaxRetries, "max-retries", c.MaxRetries, "Maximum consecutive identical errors before quitting (0 = unlimited)")

	// Add test-specific flags if we're in a test build
	// This calls the appropriate function based on build tags
	// or the GITBAK_TESTING environment variable (for compatibility)
	c.SetupTestFlags(fs)

	// Store the temporary values for later use after successful parsing
	c.ParsedNoBranch = &noBranch
	c.ParsedQuiet = &quiet
}

// ParseFlags parses the command-line arguments and updates the config
func (c *Config) ParseFlags() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	c.SetupFlags(fs)

	// For simplicity, parse whatever arguments were passed to the program.
	// If running under 'go test', the test binary will handle test flags separately.
	var appArgs []string
	// Skip the program name (os.Args[0])
	if len(os.Args) > 1 {
		appArgs = os.Args[1:]
	}

	// Parse only the application arguments
	if err := fs.Parse(appArgs); err != nil {
		return errors.NewConfigError("flags", nil, errors.Wrap(err, "failed to parse command-line arguments"))
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
		return errors.NewConfigError("interval", c.IntervalMinutes, errors.Wrap(err, "invalid interval"))
	}

	if c.RepoPath == "" {
		var err error
		c.RepoPath, err = os.Getwd()
		if err != nil {
			return errors.NewConfigError("repoPath", "", errors.Wrap(err, "failed to get current directory"))
		}
	}

	absRepoPath, err := filepath.Abs(c.RepoPath)
	if err != nil {
		return errors.NewConfigError("repoPath", c.RepoPath, errors.Wrap(err, "failed to resolve absolute path"))
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
				// Fallback to the temp directory if home dir can't be determined
				logDir = os.TempDir()
			}
		}

		// Create a unique identifier for the repository
		repoHash := fmt.Sprintf("%x", sha256OfString(c.RepoPath)[:8])

		// Final log directory and file
		gitbakLogDir := filepath.Join(logDir, "gitbak", "logs")
		c.LogFile = filepath.Join(gitbakLogDir, fmt.Sprintf("gitbak-%s.log", repoHash))

		if err := os.MkdirAll(filepath.Dir(c.LogFile), 0o700); err != nil {
			return errors.NewConfigError("logFile", c.LogFile, errors.Wrap(err, "cannot create log directory"))
		}
	}

	if c.BranchName == "" {
		if c.ContinueSession {
			currentBranch, err := getCurrentBranchName(c.RepoPath)
			if err != nil {
				return errors.NewConfigError("branchName", "",
					errors.Wrap(err, "failed to get current branch name in continue mode"))
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
