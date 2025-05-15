// Package gitbak is an automatic commit safety net
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
// # Quick Start
//
//	# Navigate to your Git repository
//	cd /path/to/your/repo
//
//	# Start gitbak with default settings (5-minute commits)
//	gitbak
//
//	# Press Ctrl+C to stop when finished
//
// # Key Features
//
//   - Automatic Commits: Creates commits at regular intervals (default: 5 minutes)
//   - Branch Management: Creates a dedicated branch or uses the current one
//   - Session Continuation: Resume sessions with sequential numbering
//   - Robust Error Handling: Smart retry logic and graceful signal handling
//
// # Documentation Structure
//
// The gitbak GitHub repository README can be found at:
//   - Github Readme: https://github.com/bashhack/gitbak/blob/main/README.md
//
// Additional documentation is organized into several sections:
//
//   - Usage & Configuration: https://github.com/bashhack/gitbak/blob/main/docs/USAGE_AND_CONFIGURATION.md
//   - After Session Guide: https://github.com/bashhack/gitbak/blob/main/docs/AFTER_SESSION.md
//   - IDE Integration: https://github.com/bashhack/gitbak/blob/main/docs/IDE_INTEGRATION.md
//   - Comparison with Alternatives: https://github.com/bashhack/gitbak/blob/main/docs/COMPARISON.md
//
// # Module Structure
//
// The module is organized into these packages:
//
//   - cmd/gitbak: Command-line interface
//   - pkg/git: Git operations and commit logic
//   - pkg/config: Configuration and flag parsing
//   - pkg/lock: File-based locking mechanism
//   - pkg/logger: Logging facilities
//   - pkg/errors: Error handling utilities
//   - pkg/constants: ASCII art and fixed values
//
// # Common Configuration Options
//
//	# Run with 2-minute intervals
//	gitbak -interval 2
//
//	# Use a custom branch name
//	gitbak -branch "my-backup-branch"
//
//	# Continue a previous session
//	gitbak -continue
//
//	# Use the current branch instead of creating a new one
//	gitbak -no-branch
//
// # After Your Session
//
// When your session is complete, you can squash all checkpoint commits into one:
//
//	# Switch back to your main branch
//	git checkout main
//
//	# Combine all gitbak commits into a single change set
//	git merge --squash gitbak-TIMESTAMP
//
//	# Create a single, meaningful commit with all changes
//	git commit -m "Complete feature implementation"
//
// # Design Philosophy
//
// gitbak is designed with the following principles in mind:
//
//   - Simple Usage: Provide a straightforward interface for common use cases
//   - Robustness: Handle errors gracefully and recover when possible
//   - Non-Intrusion: Stay out of the way of the developer's primary workflow
//   - Safety: Avoid data loss and never push to remote repositories
//   - Transparency: Make it clear what operations are being performed
//
// # Platform Support
//
// gitbak is available for:
//
//   - macOS (Intel and Apple Silicon)
//   - Linux (x86_64, ARM64)
//   - Unix-like systems with a POSIX-compliant shell
//
// # Implementation Notes
//
// gitbak uses the command-line Git executable rather than a Go Git library to ensure
// compatibility with all Git features and repository configurations. Commands are
// executed through an abstracted interface that can be replaced for testing.
//
// The application handles signals (such as SIGINT, SIGTERM, and SIGHUP) to ensure
// proper cleanup and summary display even when terminated unexpectedly.
package gitbak
