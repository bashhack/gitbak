// Package config provides configuration handling for the gitbak application.
//
// This package manages all configuration parameters for gitbak, including
// parsing command-line flags, loading environment variables, and providing
// default values. It ensures configuration values are consistent and valid
// before they are used by the application.
//
// # Core Components
//
// - Config: Main configuration type that holds all gitbak settings
// - VersionInfo: Type for version, commit, and build date information
//
// # Configuration Sources
//
// Configuration values are loaded with the following precedence:
//
// 1. Command-line flags (highest priority)
// 2. Environment variables
// 3. Default values (lowest priority)
//
// # Environment Variables
//
// The following environment variables are supported:
//
//	INTERVAL_MINUTES   Minutes between commit checks (default: 5)
//	BRANCH_NAME        Branch name to use (default: gitbak-<timestamp>)
//	COMMIT_PREFIX      Commit message prefix (default: "[gitbak]")
//	CREATE_BRANCH      Whether to create a new branch (default: true)
//	CONTINUE_SESSION   Continue an existing gitbak session (default: false)
//	VERBOSE            Whether to show informational messages (default: true)
//	SHOW_NO_CHANGES    Show messages when no changes detected (default: false)
//	DEBUG              Enable debug logging (default: false)
//	REPO_PATH          Path to repository (default: current directory)
//	MAX_RETRIES        Max consecutive identical errors before exiting (default: 3)
//	LOG_FILE           Path to log file (default: ~/.local/share/gitbak/logs/gitbak-<hash>.log)
//
// # Command-line Flags
//
// The following command-line flags are supported:
//
//	-interval        Minutes between commit checks
//	-branch          Branch name to use
//	-prefix          Commit message prefix
//	-no-branch       Stay on current branch instead of creating a new one
//	-continue        Continue existing session
//	-show-no-changes Show messages when no changes detected
//	-quiet           Hide informational messages
//	-repo            Path to repository
//	-max-retries     Max consecutive identical errors before exiting
//	-debug           Enable debug logging
//	-log-file        Path to log file
//	-version         Print version information and exit
//	-logo            Display ASCII logo and exit
//
// # Usage
//
// Basic usage pattern:
//
//	cfg := config.New()
//	cfg.LoadFromEnvironment()
//
//	if err := cfg.ParseFlags(); err != nil {
//	    // Handle error
//	}
//
//	// Configuration is now ready to use
//	fmt.Printf("Interval: %.2f minutes\n", cfg.IntervalMinutes)
//	fmt.Printf("Branch: %s\n", cfg.BranchName)
//
// # Thread Safety
//
// The Config type is not designed to be thread-safe. Configuration is typically
// loaded at startup and then used in a read-only fashion by the application.
// Concurrent modifications to a Config instance are not supported.
package config
