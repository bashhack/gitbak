<p align="center">
  <img src="/assets/gitbak_retro_logo.png" alt="gitbak logo" width="300">
</p>

# gitbak (Shell Script Version)

> ⚠️ **IMPORTANT**: This shell script implementation is **UNSUPPORTED** and maintained only for historical purposes. For production use, please use the [Go implementation](/go/README.md) which provides better reliability, performance, and ongoing support.

A lightweight shell script that automatically commits changes at regular intervals, providing a safety net during programming sessions.

## Overview

The shell script version of gitbak is designed to be highly portable, working across a variety of Unix-like systems and shell environments. It provides automatic checkpoint commits during programming sessions without requiring any compiled components.

## Features

- Works with bash, zsh, dash, and standard sh shells
- Automatically commits changes at regular intervals (default: 5 minutes)
- Creates a dedicated branch for backup commits (configurable)
- Continuous tracking with sequential commit numbering
- Support for continuing sessions after breaks or interruptions
- Clean termination when signals are received (including terminal close)
- Session summary with statistics when you finish

## Requirements

- A POSIX-compliant shell (bash, zsh, dash, sh, etc.)
- Git
- Standard Unix utilities:
    - date
    - grep
    - sleep
    - ps
    - wc
    - find
    - cat

## Installation

### Option 1: Download from GitHub Releases

1. Download and install using the tar.gz package:
   ```bash
   curl -sL https://github.com/bashhack/gitbak/releases/latest/download/gitbak-shell.tar.gz -o gitbak-shell.tar.gz
   tar -xzf gitbak-shell.tar.gz
   ./gitbak-shell/install.sh
   ```

   The install script will:
   - Install gitbak to `~/.local/bin` by default (or a custom directory specified via `INSTALL_DIR`)
   - Make the script executable
   - Check if the installation directory is in your PATH
   - Provide instructions for adding it to your PATH if needed

2. Or manually:
   - Visit the [GitHub Releases page](https://github.com/bashhack/gitbak/releases)
   - Download the `gitbak.sh` file from the latest release
   - Make it executable: `chmod +x gitbak.sh`
   - Move it to a directory in your PATH: `mv gitbak.sh ~/.local/bin/gitbak`

### Option 2: Manual Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/bashhack/gitbak.git
   ```

2. Install using the Makefile:
   ```bash
   cd gitbak/sh
   make install
   ```

3. Or manually copy the script:
   ```bash
   cp gitbak/sh/gitbak.sh ~/.local/bin/gitbak
   chmod +x ~/.local/bin/gitbak
   ```

## Usage

### Basic Usage

1. Navigate to your project repository:
   ```bash
   cd /path/to/your/project
   ```

2. Run gitbak:
   ```bash
   gitbak
   ```

3. Press `Ctrl+C` to stop when finished

### Configuration Options

The script can be configured using environment variables:

```bash
# Change commit interval to 10 minutes
INTERVAL_MINUTES=10 gitbak

# Use a custom branch name
BRANCH_NAME="feature-branch-backup" gitbak

# Use a custom commit message prefix
COMMIT_PREFIX="[WIP] Checkpoint" gitbak

# Stay on current branch instead of creating a new one
CREATE_BRANCH=false gitbak

# Continue an existing gitbak session
CONTINUE_SESSION=true gitbak

# Show more verbose output
VERBOSE=true gitbak

# Show messages even when no changes are detected
SHOW_NO_CHANGES=true gitbak
```

### Environment Variables

| Variable           | Default                      | Description                                           |
|--------------------|------------------------------|-------------------------------------------------------|
| `INTERVAL_MINUTES` | `5`                          | Minutes between commit checks                         |
| `BRANCH_NAME`      | `gitbak-TIMESTAMP`          | Name for the new branch                               |
| `COMMIT_PREFIX`    | `[gitbak]`                   | Prefix for commit messages                            |
| `CREATE_BRANCH`    | `true`                       | Whether to create a new branch                        |
| `VERBOSE`          | `true`                       | Whether to show informational messages                |
| `SHOW_NO_CHANGES`  | `false`                      | Whether to show messages when no changes are detected |
| `CONTINUE_SESSION` | `false`                      | Continue on current branch and resume commit numbering|

## Shell Compatibility Notes

The script is designed to work with a variety of shell environments:

- **bash**: Full compatibility with all features
- **zsh**: Full compatibility with all features
- **dash**: Compatible with core functionality
- **sh (POSIX)**: Compatible with core functionality

To verify compatibility with your specific shell, run the shell compatibility test:

```bash
cd /path/to/gitbak/sh/tests
./shell_compatibility.sh
```

## How It Works

1. The script creates a timestamped branch (unless `CREATE_BRANCH=false`)
2. It checks for changes every `INTERVAL_MINUTES` minutes
3. When changes are detected, they're committed with a numbered, timestamped message
4. It handles errors gracefully and retries when git commands fail
5. The script continues until you stop it with `Ctrl+C`

## 💡 Power User Workflow: Mixing Manual and Automatic Commits

One of gitbak's most powerful capabilities is supporting manual commits alongside automatic ones:

```bash
# Start gitbak with no-branch option
CREATE_BRANCH=false gitbak

# While gitbak runs in the background:
# 1. Make changes to your code
# 2. When you reach a meaningful milestone, make a manual commit:
git add <specific-files>
git commit -m "Implement login feature"
# 3. Continue working as normal - gitbak keeps making checkpoints
```

Benefits of this approach:
- Automatic safety checkpoints happen even if you forget to commit
- You maintain control over your repository's important milestones
- gitbak's automatic numbering remains sequential despite manual commits
- When you're done, you can keep your meaningful commits and discard the automatic ones

This gives you the best of both worlds: meaningful commit history AND comprehensive safety.

> 💡 See [Comparison with Alternatives](/go/docs/COMPARISON.md) for why this approach is superior to IDE auto-save features.

## After Your Session

When you stop gitbak using Ctrl+C, it provides a session summary with:

- Total number of commits made
- Session duration
- Branch information
- Helpful commands for integrating your changes

For integrating your changes, see the common approaches below:

### Squash Merge (Most Common)

```bash
# Switch back to your main branch
git checkout main

# Combine all gitbak commits into a single change set
git merge --squash gitbak-TIMESTAMP 

# Create a single, meaningful commit with all changes
git commit -m "Add feature X from pair programming session"
```

## Debugging and Troubleshooting

If you encounter issues:

1. Run the script with `set -x` to see debug output:
   ```bash
   bash -x /path/to/gitbak/sh/gitbak.sh
   ```

2. Check for common issues:
    - Make sure all required utilities are available in your PATH
    - Verify you're in a git repository
    - Check if another instance is already running
    - Ensure git is properly configured in your repository

## Limitations

Compared to the Go version, the shell script has a few limitations:

- Requires more external dependencies (standard Unix utilities)
- Limited cross-platform compatibility (primarily for Unix-like systems)
- Less robust error handling and recovery
- Simpler locking mechanism

For a more robust, cross-platform implementation with additional features, consider using the [Go version](/go/README.md).

## Testing

The shell script version includes a comprehensive test suite:

```bash
# Run all tests
cd /path/to/gitbak/sh/tests
./run_tests.sh

# Run specific tests
./run_tests.sh basic_functionality.sh
```

Available tests include:
- `basic_functionality.sh`: Core functionality
- `continuation.sh`: Session continuation features
- `lock_file.sh`: Lock file behavior
- `shell_compatibility.sh`: Shell environment compatibility
- `stress_test.sh`: Performance under load

## License

MIT
