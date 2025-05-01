# GitBak - Automatic Commit Safety Net for Pair Programming

This lightweight utility script automates creating checkpoint commits during pair programming sessions, providing a safety net against accidental code loss.

## Purpose

When pair programming (with humans or AI assistants like Claude), the conversation and code changes can move quickly. GitBak provides safety by:

- Creating automatic commits at regular intervals
- Making a clean history of your pairing session progress
- Providing recovery points if something goes wrong

## Usage

### Basic Usage

1. Make the script executable:
   ```
   chmod +x gitbak.sh
   ```

2. Navigate to your project repository:
   ```
   cd /path/to/your/project
   ```

3. Start the auto-commit process:
   ```
   /path/to/gitbak.sh
   ```

4. Continue pair programming in other terminals while this runs in the background

5. Press `Ctrl+C` to stop when finished

### Configuration with Environment Variables

You can customize the script behavior without editing it by using environment variables:

```bash
# Change commit interval to 10 minutes
INTERVAL_MINUTES=10 ./gitbak.sh

# Use a custom branch name
BRANCH_NAME="feature-refactoring" ./gitbak.sh

# Use a specific commit message prefix
COMMIT_PREFIX="[WIP] Feature implementation" ./gitbak.sh

# Stay on current branch instead of creating a new one
CREATE_BRANCH=false ./gitbak.sh

# Combine multiple options
INTERVAL_MINUTES=2 COMMIT_PREFIX="[Checkpoint]" ./gitbak.sh

# Continue an existing GitBak session after a break
CONTINUE_SESSION=true ./gitbak.sh

# Enable debug logging
DEBUG=true ./gitbak.sh
```

### Available Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `INTERVAL_MINUTES` | `5` | Minutes between commit checks |
| `BRANCH_NAME` | `gitbak-TIMESTAMP` | Name for the new branch |
| `COMMIT_PREFIX` | `[GitBak] Automatic checkpoint` | Prefix for commit messages |
| `CREATE_BRANCH` | `true` | Whether to create a new branch |
| `VERBOSE` | `true` | Whether to show informational messages |
| `SHOW_NO_CHANGES` | `false` | Whether to show messages when no changes are detected |
| `REPO_PATH` | Current directory | Path to repository (if run from elsewhere) |
| `CONTINUE_SESSION` | `false` | Continue on current branch and resume commit numbering |
| `DEBUG` | `false` | Enable detailed logging for troubleshooting |
| `LOG_FILE` | `$(pwd)/.gitbak.log` | Path to debug log file (when DEBUG=true) |

## How It Works

1. The script creates a timestamped branch (unless `CREATE_BRANCH=false`)
2. It checks for changes every `INTERVAL_MINUTES` minutes
3. When changes are detected, they're committed with a numbered, timestamped message
4. The script handles errors gracefully and retries when git commands fail
5. It only allows one instance to run per repository using a lock file mechanism
6. It captures Ctrl+C, terminal disconnections, and other signals to ensure clean termination
7. When terminated, it displays a helpful session summary with statistics and next steps
8. The script continues until you stop it with `Ctrl+C`

## After Your Session

When your pairing session is complete, you can:

1. Keep all commits by merging the branch as-is
2. Squash commits into a single meaningful commit 
3. Cherry-pick specific changes
4. Discard the branch if you don't need the automatic commits

## Advanced Features

### Session Summary

When you stop GitBak using Ctrl+C, it provides a session summary with:

- Total number of commits made
- Session duration in hours, minutes, and seconds
- Branch information
- Helpful commands for merging or squashing commits
- Git branch visualization in ASCII format
- Timestamp of termination

The branch visualization shows a graphical representation of your commit history, making it easy to see:
- The relationship between your GitBak branch and the original branch
- All commits created during your session
- The overall structure of your repository's branches

### Continuing Sessions

GitBak supports taking breaks and continuing later with the `CONTINUE_SESSION` option:

```bash
# Step 1: Start a session
./gitbak.sh

# Later - Press Ctrl+C to pause for lunch or a break

# Step 2: After the break, continue on the same branch with sequential commit numbers
CONTINUE_SESSION=true ./gitbak.sh
```

When you set `CONTINUE_SESSION=true`:
- GitBak stays on the current branch (regardless of `CREATE_BRANCH` setting)
- It automatically detects the last commit number and continues sequentially
- It preserves the branch history and continues where you left off
- Perfect for lunch breaks, bio breaks, or interruptions during a pair programming session

### Lock File Protection

GitBak prevents multiple instances from running for the same repository by:

- Creating a lock file with the process ID
- Checking for existing lock files when starting
- Cleaning up stale lock files if a previous instance crashed
- Removing the lock file on clean termination

### Error Handling and Recovery

The script includes robust error handling:

- Captures and displays git command errors
- Retries failed git operations at the next interval check
- Gracefully handles common git issues
- Shows detailed error messages for troubleshooting

## IDE Integration

### Visual Studio Code

Add GitBak to VS Code Tasks:

1. Create or edit `.vscode/tasks.json`:
```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Start GitBak",
      "type": "shell",
      "command": "/path/to/gitbak.sh",
      "isBackground": true,
      "problemMatcher": []
    }
  ]
}
```

2. Run via `Terminal > Run Task... > Start GitBak`

For more details on VS Code tasks, see [VS Code Tasks Documentation](https://code.visualstudio.com/docs/editor/tasks).

### JetBrains IDEs (IntelliJ, WebStorm, etc.)

Add GitBak as an External Tool:

1. Go to `Preferences/Settings > Tools > External Tools`
2. Click `+` and configure:
   - Name: `GitBak`
   - Program: `/path/to/gitbak.sh`
   - Working directory: `$ProjectFileDir$`
   - Advanced Options: Check "Asynchronous execution"
3. Run from `Tools > External Tools > GitBak`

For more details, see [JetBrains External Tools Documentation](https://www.jetbrains.com/help/idea/configuring-third-party-tools.html#local-ext-tools).

### Emacs

Add to your `.emacs` or `init.el`:

```elisp
(defun start-gitbak ()
  "Start GitBak in the current project."
  (interactive)
  (let ((default-directory (or (projectile-project-root) default-directory)))
    (start-process "gitbak" "*GitBak*" "/path/to/gitbak.sh")))

(global-set-key (kbd "C-c g b") 'start-gitbak)
```

This assumes you have projectile installed. If not, you can simplify to just use the current directory.

## Frequently Asked Questions

### Resource Usage

GitBak is designed to be very lightweight:
- Memory usage: Approximately 2-3 MB when running
- CPU usage: Minimal (only active when checking for changes or committing)
- Disk usage: Only the space required for Git commits
- Network: None (operates entirely locally)

### Local vs Remote Repositories

GitBak operates entirely locally:
- It does not push changes to remote repositories
- All commits remain on your local machine until you explicitly push them
- You maintain complete control over when/if changes go to remote

### File Handling

GitBak respects your repository's existing `.gitignore` files:
- It does not implement a separate ignore system
- Any files ignored by Git will also be ignored by GitBak
- Standard Git practices for ignoring files apply

### Debugging and Troubleshooting

When issues occur:
1. Enable debug mode: `DEBUG=true ./gitbak.sh`
2. Check the log file (default: `.gitbak.log` in your repository)
3. Look for error messages in the console output
4. Check for stale lock files: `/tmp/gitbak-*.lock`

Common issues and solutions:

**"Another GitBak instance is running"**
Check for running GitBak processes:
```bash
# Find existing GitBak processes
ps aux | grep gitbak.sh

# Kill a specific GitBak process
kill <PID>

# Remove stale lock files (use with caution)
REPO_HASH=$(echo "$(pwd)" | shasum | cut -d' ' -f1)
rm -f /tmp/gitbak-$REPO_HASH.lock

# Find all GitBak lock files
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
BRANCH_NAME="gitbak-custom" ./gitbak.sh
```

## Git Integration Notes

GitBak is intentionally designed to work with standard Git operations and workflows. It:

- Creates branches using standard Git commands
- Makes commits using standard Git commands
- Follows your repository's .gitignore configuration

### Common Git Operations with GitBak

- **Merging GitBak changes**: Use standard Git merge operations (`git merge`, `git merge --squash`)
- **Cleaning up branches**: Use standard Git branch management (`git branch -D`)
- **Handling conflicts**: Resolve using Git's standard conflict resolution process
- **Team usage**: Each team member can run GitBak on their own branches

For more information on these Git operations, see [Git documentation](https://git-scm.com/doc).

## Tips

- Start the script BEFORE beginning your work with Claude
- Run it in a separate terminal window that you can minimize
- If you notice issues in your code, check the git history to find a working state
- Use the `CREATE_BRANCH=false` option if you want commits on your current branch
- For very active sessions, consider lowering the interval (`INTERVAL_MINUTES=2`)
- For longer sessions, consider increasing the interval (`INTERVAL_MINUTES=10`)
- Use `VERBOSE=false` to minimize output if you're using the script frequently
- Set `SHOW_NO_CHANGES=true` if you want to see when the script checks but finds no changes
- When taking a break, use `Ctrl+C` to stop GitBak and `CONTINUE_SESSION=true` to resume later
- For multi-day sessions, use `CONTINUE_SESSION=true` each morning to pick up where you left off