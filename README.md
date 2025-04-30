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

## How It Works

1. The script creates a timestamped branch (unless `CREATE_BRANCH=false`)
2. It checks for changes every `INTERVAL_MINUTES` minutes
3. When changes are detected, they're committed with a numbered, timestamped message
4. The script handles errors gracefully and retries when git commands fail
5. It only allows one instance to run per repository using a lock file mechanism
6. It captures Ctrl+C and other signals to ensure clean termination
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
- Timestamp of termination

### Lock File Protection

GitBak prevents multiple instances from running for the same repository by:

- Creating a lock file with the process ID
- Checking for existing lock files when starting
- Cleaning up stale lock files if a previous instance crashed
- Removing the lock file on clean termination

### Error Handling and Recovery

The script includes robust error handling:

- Captures and displays git command errors
- Automatically retries after failed operations
- Gracefully handles common git issues
- Shows detailed error messages for troubleshooting

## Tips

- Start the script BEFORE beginning your work with Claude
- Run it in a separate terminal window that you can minimize
- If you notice issues in your code, check the git history to find a working state
- Use the `CREATE_BRANCH=false` option if you want commits on your current branch
- For very active sessions, consider lowering the interval (`INTERVAL_MINUTES=2`)
- For longer sessions, consider increasing the interval (`INTERVAL_MINUTES=10`)
- Use `VERBOSE=false` to minimize output if you're using the script frequently
- Set `SHOW_NO_CHANGES=true` if you want to see when the script checks but finds no changes