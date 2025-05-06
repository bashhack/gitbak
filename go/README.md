<p style="text-align: center;">
  <img src="/assets/gitbak_retro_logo.png" alt="gitbak logo" width="300">
</p>

# gitbak

A Go implementation of the gitbak automatic commit safety net for pair programming.

## Purpose

When pair programming (with humans or AI assistants like Claude), the conversation and code changes can move quickly. gitbak provides safety by:

- Creating automatic commits at regular intervals
- Making a clean history of your pairing session progress
- Providing recovery points if something goes wrong

## Overview

gitbak is a daemon that automatically commits changes to git at regular intervals, providing a safety net for pair programming sessions or long coding sessions. 
It's a Go port of the original shell script version with improved architecture, error handling, and cross-platform support.

## Features

- Automatically commits changes at specified intervals
- Creates a dedicated branch for backup commits (configurable)
- Handles concurrent executions safely with file locking
- Continuous tracking with sequential commit numbering
- Support for continuing sessions
- Robust error handling and retry logic
- Terminal disconnect protection (SIGHUP handling)
- Cross-platform support with native Go implementation

## Usage

### Basic Usage

1. Install gitbak:
   ```
   make install
   ```

2. Navigate to your project repository:
   ```
   cd /path/to/your/project
   ```

3. Start the auto-commit process:
   ```
   gitbak
   ```

4. Continue pair programming in other terminals while this runs in the background

5. Press `Ctrl+C` to stop when finished

### Command Examples

```shell
# Run with default settings (5-minute interval)
gitbak

# Run with a custom interval (2 minutes)
gitbak -interval 2

# Run with a custom branch name
gitbak -branch "my-backup-branch"

# Continue from an existing gitbak session
gitbak -continue

# Use current branch instead of creating a new one
gitbak -no-branch

# Show messages when no changes are detected
gitbak -show-no-changes

# Enable debug logging
gitbak -debug

# Display ASCII logo and exit
gitbak -logo
```

### Configuration Using Environment Variables

All command-line options can also be set via environment variables:

```shell
# Change commit interval to 10 minutes
INTERVAL_MINUTES=10 gitbak

# Use a custom branch name
BRANCH_NAME="feature-refactoring" gitbak

# Use a specific commit message prefix
COMMIT_PREFIX="[WIP] Feature implementation" gitbak

# Stay on current branch instead of creating a new one
CREATE_BRANCH=false gitbak

# Combine multiple options
INTERVAL_MINUTES=2 COMMIT_PREFIX="[Checkpoint]" gitbak

# Continue an existing gitbak session after a break
CONTINUE_SESSION=true gitbak

# Enable debug logging
DEBUG=true gitbak
```

## Configuration Options

### Command Line Flags

The following options can be configured via command-line flags:

- `-interval`: Minutes between commits (default: 5)
- `-branch`: Custom branch name (default: gitbak-{timestamp})
- `-prefix`: Custom commit message prefix (default: "[gitbak] Automatic checkpoint")
- `-no-branch`: Use current branch instead of creating a new one
- `-quiet`: Hide informational messages
- `-show-no-changes`: Show messages when no changes detected
- `-repo`: Path to repository (default: current directory)
- `-continue`: Continue from existing branch
- `-debug`: Enable debug logging
- `-log-file`: Path to log file (default: ~/.local/share/gitbak/logs/gitbak-{repo-hash}.log)
- `-version`: Print version information and exit
- `-logo`: Display ASCII logo and exit

### Environment Variables

| Variable           | Default                                             | Description                                            |
|--------------------|-----------------------------------------------------|--------------------------------------------------------|
| `INTERVAL_MINUTES` | `5`                                                 | Minutes between commit checks                          |
| `BRANCH_NAME`      | `gitbak-TIMESTAMP`                                  | Name for the new branch                                |
| `COMMIT_PREFIX`    | `[gitbak] Automatic checkpoint`                     | Prefix for commit messages                             |
| `CREATE_BRANCH`    | `true`                                              | Whether to create a new branch                         |
| `VERBOSE`          | `true`                                              | Whether to show informational messages                 |
| `SHOW_NO_CHANGES`  | `false`                                             | Whether to show messages when no changes are detected  |
| `REPO_PATH`        | Current directory                                   | Path to repository (if run from elsewhere)             |
| `CONTINUE_SESSION` | `false`                                             | Continue on current branch and resume commit numbering |
| `DEBUG`            | `false`                                             | Enable detailed logging for troubleshooting            |
| `LOG_FILE`         | `~/.local/share/gitbak/logs/gitbak-{repo-hash}.log` | Path to debug log file                                 |

## How It Works

1. The program creates a timestamped branch (unless `-no-branch` or `CREATE_BRANCH=false`)
2. It checks for changes every `-interval` minutes
3. When changes are detected, they're committed with a numbered, timestamped message
4. It handles errors gracefully and retries when git commands fail
5. It only allows one instance to run per repository using a file locking mechanism
6. It captures Ctrl+C, terminal disconnections, and other signals to ensure clean termination
7. When terminated, it displays a helpful session summary with statistics and next steps
8. The program continues until you stop it with `Ctrl+C`

## After Your Session

When your pairing session is complete, you can:

1. Keep all commits by merging the branch as-is
2. Squash commits into a single meaningful commit 
3. Cherry-pick specific changes
4. Discard the branch if you don't need the automatic commits

### Integrating Your Changes

The most common approach for integrating gitbak changes back to your main branch is the squash merge:

```bash
# Switch back to your main branch
git checkout main

# Combine all gitbak commits into a single change set
git merge --squash gitbak-TIMESTAMP 

# Create a single, meaningful commit with all changes
git commit -m "Add feature X from pair programming session"
```

This approach:
- Creates a clean, single commit in your main history
- Preserves all your work in one atomic change
- Simplifies code reviews
- Keeps your commit history tidy

If you want to preserve the detailed history of your session, use a standard merge instead:

```bash
git checkout main
git merge gitbak-TIMESTAMP
```

For more selective integration, cherry-pick specific commits:

```bash
git checkout main
git cherry-pick <commit-hash>  # Repeat for each desired commit
```

## Advanced Features

### Session Summary

When you stop gitbak using Ctrl+C, it provides a session summary with:

- Total number of commits made
- Session duration in hours, minutes, and seconds
- Branch information
- Helpful commands for merging or squashing commits
- Git branch visualization in ASCII format
- Timestamp of termination

The branch visualization shows a graphical representation of your commit history, making it easy to see:
- The relationship between your gitbak branch and the original branch
- All commits that were created during your session
- The overall structure of your repository's branches

### Continuing Sessions

gitbak supports taking breaks and continuing later with the `-continue` flag:

```bash
# Step 1: Start a session
gitbak

# Later - Press Ctrl+C to pause for lunch or a break

# Step 2: After the break, continue on the same branch with sequential commit numbers
gitbak -continue
```

When you use `-continue`:
- gitbak stays on the current branch (regardless of `-no-branch` setting)
- It automatically detects the last commit number and continues sequentially
- It preserves the branch history and continues where you left off
- Perfect for lunch breaks, bio breaks, or interruptions during a pair programming session

### Lock File Protection

gitbak prevents multiple instances from running for the same repository by:

- Creating a lock file with the process ID
- Checking for existing lock files when starting
- Cleaning up stale lock files if a previous instance crashed
- Removing the lock file on clean termination

### Error Handling and Recovery

The program includes robust error handling:

- Captures and displays git command errors
- Retries failed git operations at the next interval check
- Gracefully handles common git issues
- Shows detailed error messages for troubleshooting

## IDE Integration

### Visual Studio Code

Add gitbak to VS Code Tasks:

1. Create or edit `.vscode/tasks.json`:
```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Start gitbak",
      "type": "shell",
      "command": "gitbak",
      "isBackground": true,
      "problemMatcher": []
    }
  ]
}
```

2. Run via `Terminal > Run Task... > Start gitbak`

For more details on VS Code tasks, see [VS Code Tasks Documentation](https://code.visualstudio.com/docs/editor/tasks).

### JetBrains IDEs (GoLand, IntelliJ, etc.)

Add gitbak as an External Tool:

1. Go to `Preferences/Settings > Tools > External Tools`
2. Click `+` and configure:
   - Name: `gitbak`
   - Program: `gitbak`
   - Working directory: `$ProjectFileDir$`
   - Advanced Options: Check "Asynchronous execution"
3. Run from `Tools > External Tools > gitbak`

For more details, see [JetBrains External Tools Documentation](https://www.jetbrains.com/help/idea/configuring-third-party-tools.html#local-ext-tools).

### Emacs

Add to your `.emacs` or `init.el`:

```elisp
(defun start-gitbak ()
  "Start gitbak in the current project."
  (interactive)
  (let ((default-directory (or (projectile-project-root) default-directory)))
    (start-process "gitbak" "*gitbak*" "gitbak")))

(global-set-key (kbd "C-c g b") 'start-gitbak)
```

This assumes you have projectile installed. If not, you can simplify to just use the current directory.

## Frequently Asked Questions

### Resource Usage

gitbak is designed to be very lightweight:
- Memory usage: Typically ~5 MB when running
- CPU usage: Minimal (only active when checking for changes or committing)
- Disk usage: Only the space required for Git commits
- Network: None (operates entirely locally)

### Local vs Remote Repositories

gitbak operates entirely locally:
- It does not push changes to remote repositories
- All commits remain on your local machine until you explicitly push them
- You maintain complete control over when/if changes go to remote

### File Handling

gitbak respects your repository's existing `.gitignore` files:
- It does not implement a separate ignore system
- Any files ignored by Git will also be ignored by gitbak
- Standard Git practices for ignoring files apply

### Debugging and Troubleshooting

When issues occur:
1. Enable debug mode: `gitbak -debug`
2. Check the log file (default: `~/.local/share/gitbak/logs/gitbak-{repo-hash}.log`)
3. Look for error messages in the console output

Common issues and solutions:

**"Another gitbak instance is running"**
Check for running gitbak processes:
```bash
# Find existing gitbak processes
ps aux | grep gitbak

# Kill a specific gitbak process
kill <PID>

# Find all gitbak lock files
find /tmp -name "gitbak-*.lock"
```

**"Not a git repository"**
```bash
# Verify you're in a git repository
git rev-parse --is-inside-work-tree
```

**"Failed to create branch"**
```bash
# Check existing branches
git branch

# Try a different branch name
gitbak -branch "gitbak-custom"
```

## Dependencies

The following command must be available in your system:
- `git`: For repository operations

The Go implementation has been optimized to eliminate other external dependencies.

## Installation

### Using Homebrew (macOS and Linux)

```shell
# Install from the Homebrew tap
brew install bashhack/gitbak/gitbak
```

### Using GitHub Releases (macOS, Linux, Windows)

Download the appropriate binary for your platform from the [Releases](https://github.com/bashhack/gitbak/releases) page.

```shell
# Example for macOS (arm64)
curl -L https://github.com/bashhack/gitbak/releases/latest/download/gitbak_Darwin_arm64.tar.gz | tar xz
chmod +x gitbak
sudo mv gitbak /usr/local/bin/
```

### Using Go Install (requires Go 1.24+)

```shell
# Install directly from go.pkg.dev
go install github.com/bashhack/gitbak/go/cmd/gitbak@latest

# Or install a specific version
go install github.com/bashhack/gitbak/go/cmd/gitbak@v1.2.3
```

### From Source

```shell
# Clone the repository
git clone https://github.com/bashhack/gitbak.git
cd gitbak/go

# Build and install
make install
```

This will install the binary to `~/.local/bin/gitbak`. Make sure this directory is in your PATH.

## Development

### Requirements

- Go 1.24 or later
- Git

### Setting Up the Development Environment

1. Clone the repository:
   ```shell
   git clone https://github.com/bashhack/gitbak.git
   cd gitbak/go
   ```

2. Install development tools:
   ```shell
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

3. Build the project:
   ```shell
   make build
   ```

4. Run the application in development mode:
   ```shell
   make run
   ```

### Running Tests

```shell
# Run all unit tests
make test

# Run tests with verbose output
make test/verbose

# Run only quick tests
make test/short

# Run tests with coverage report
make coverage

# Run function-level coverage report
make coverage/func

# Run integration tests (requires more time)
GITBAK_INTEGRATION_TESTS=1 make test/integration
```

### Signal Handling

gitbak handles the following signals:

- `SIGINT` (Ctrl+C): Gracefully stops the application and prints a summary
- `SIGTERM`: Gracefully stops the application and prints a summary
- `SIGHUP`: Handles terminal disconnects properly by cleaning up and exiting gracefully

### Code Quality

```shell
# Run linters (go vet, staticcheck)
make lint

# Run full audit (format, tidy, lint, test)
make audit
```

### Building for Different Platforms

```shell
# Build for current platform
make build

# Build optimized binary (smaller size)
make build/optimize

# Build for all supported platforms
make build/all
```

## Tips

- Start gitbak BEFORE beginning your work with your human pairing partner or your AI assistant
- Run it in a separate terminal window that you can minimize
- If you notice issues in your code, check the git history to find a working state
- Use the `-no-branch` option if you want commits on your current branch
- For very active sessions, consider lowering the interval (`-interval 2`)
- For longer sessions, consider increasing the interval (`-interval 10`)
- Use `-quiet` to minimize output if you're using the program frequently
- Set `-show-no-changes` if you want to see when the program checks but finds no changes
- When taking a break, use `Ctrl+C` to stop gitbak and `-continue` to resume later
- For multi-day sessions, use `-continue` each morning to pick up where you left off

## Git Integration Notes

gitbak is intentionally designed to work with standard Git operations and workflows. It:

- Creates branches using standard Git commands
- Makes commits using standard Git commands
- Follows your repository's .gitignore configuration

### Common Git Operations with gitbak

- **Merging gitbak changes**: Use standard Git merge operations (`git merge`, `git merge --squash`)
- **Cleaning up branches**: Use standard Git branch management (`git branch -D`)
- **Handling conflicts**: Resolve using Git's standard conflict resolution process
- **Team usage**: Each team member can run gitbak on their own branches

For more information on these Git operations, see [Git documentation](https://git-scm.com/doc).

## License

MIT
