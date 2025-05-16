// Package main implements gitbak, an automatic commit safety net
//
// gitbak automatically commits changes to git at regular intervals,
// providing a safety net for programming sessions. It's especially valuable for
// pair programming, long coding sessions, or exploratory coding where you want
// to focus on the work rather than remembering to commit. It runs as a background
// process that monitors your repository and creates sequential, numbered commits.
//
// While gitbak creates automatic checkpoint commits, you can still make your own
// meaningful manual commits at important milestones. This powerful combination gives
// you both a detailed safety net AND a clean, meaningful commit history - the best
// of both worlds.
//
// # Command-Line Documentation
//
// This package provides the command-line interface for the gitbak tool. For additional
// documentation on the underlying functionality, see: https://pkg.go.dev/github.com/bashhack/gitbak
//
// # Features
//
//   - Automatically commits changes at specified intervals (default: 5 minutes)
//   - Creates a dedicated branch for backup commits (configurable)
//   - Handles concurrent executions safely with file locking
//   - Continuous tracking with sequential commit numbering
//   - Support for continuing sessions after breaks or interruptions
//   - Robust error handling with configurable retry limits
//   - Smart retry logic that resets on different errors or successful operations
//   - Terminal disconnect protection (SIGHUP handling)
//
// # Basic Usage
//
//	gitbak                     # Run with default settings (5-minute interval)
//	gitbak -interval 2         # Run with a custom interval (2 minutes)
//	gitbak -branch "my-branch" # Run with a custom branch name
//	gitbak -continue           # Continue from an existing gitbak session
//	gitbak -no-branch          # Use current branch instead of creating a new one
//
// # Configuration Options
//
// The tool can be configured via command-line flags or environment variables:
//
//	-interval        Minutes between commit checks (env: INTERVAL_MINUTES)
//	-branch          Branch name to use (env: BRANCH_NAME)
//	-prefix          Commit message prefix (env: COMMIT_PREFIX)
//	-no-branch       Stay on current branch (env: CREATE_BRANCH=false)
//	-continue        Continue existing session (env: CONTINUE_SESSION=true)
//	-show-no-changes Show messages when no changes detected (env: SHOW_NO_CHANGES=true)
//	-quiet           Hide informational messages (env: VERBOSE=false)
//	-max-retries     Max consecutive identical errors before exiting (env: MAX_RETRIES)
//	-debug           Enable detailed logging (env: DEBUG=true)
//	-version         Print version information and exit
//	-logo            Display ASCII logo and exit
//
// # Session Continuation vs Branch Creation
//
// Two important flags control how gitbak interacts with Git branches:
//
//  1. -no-branch: Tells gitbak to use the current branch instead of creating a new one.
//     When this flag is not specified, gitbak creates a new timestamped branch.
//
//  2. -continue: Used when resuming a previous gitbak session. This flag:
//
//     a. Automatically stays on the current branch (implicitly includes -no-branch behavior)
//
//     b. Identifies the last commit number used by gitbak and continues numbering sequentially
//
//     c. Preserves branch history and continues tracking from where you left off
//
// # After Your Session
//
// When your session is complete, you can:
//
// 1. Squash merge all gitbak commits into a single commit:
//
//	git checkout main
//	git merge --squash gitbak-<timestamp>
//	git commit -m "Add feature X from pair programming session"
//
// 2. Cherry-pick specific changes from the gitbak branch:
//
//	git checkout main
//	git cherry-pick <commit-hash>
//
// 3. Merge the branch as-is to keep all individual commits
//
// See https://github.com/bashhack/gitbak for additional documentation.
package main
