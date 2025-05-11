#!/bin/sh

# gitbak - Automatic Commit Safety Net
#
# This script automatically commits changes to git at regular intervals,
# providing a safety net for programming sessions, especially useful for
# pair programming, long coding sessions or exploratory coding.
#
# Usage:
#   - Run from inside a Git repository: ./gitbak.sh
#   - Press Ctrl+C to stop and view the session summary
#
# Configuration (via environment variables):
#   - INTERVAL_MINUTES: Minutes between commits (default: 5)
#   - BRANCH_NAME: Custom branch name (default: gitbak-{timestamp})
#   - COMMIT_PREFIX: Custom commit message prefix (default: "[gitbak] Automatic checkpoint")
#   - CREATE_BRANCH: Whether to create a new branch (default: "true")
#   - VERBOSE: Show/hide informational messages (default: "true")
#   - SHOW_NO_CHANGES: Show messages when no changes detected (default: "false")
#   - REPO_PATH: Path to repository (default: current directory)
#   - CONTINUE_SESSION: Continue from existing branch (default: "false")
#   - DEBUG: Enable debug logging (default: "false")
#   - LOG_FILE: Path to log file (default: ~/.local/share/gitbak/logs/gitbak-{repo-hash}.log)

DEBUG=${DEBUG:-"false"}

# Set up repository path and log file path following XDG Base Directory Specification
REPO_PATH=${REPO_PATH:-$(pwd)}
if [ -z "$LOG_FILE" ]; then
    if [ -n "$XDG_DATA_HOME" ]; then
        LOG_BASE_DIR="$XDG_DATA_HOME"
    else
        LOG_BASE_DIR="$HOME/.local/share"
    fi

    # Create a unique hash for the repository path
    if command -v shasum >/dev/null 2>&1; then
        REPO_HASH=$(echo "$REPO_PATH" | shasum | cut -d' ' -f1 | head -c8)
    elif command -v md5sum >/dev/null 2>&1; then
        REPO_HASH=$(echo "$REPO_PATH" | md5sum | cut -d' ' -f1 | head -c8)
    else
        # Simple fallback if no hash commands are available
        REPO_HASH=$(echo "$REPO_PATH" | tr -cd '[:alnum:]' | head -c8)
    fi

    LOG_DIR="$LOG_BASE_DIR/gitbak/logs"
    LOG_FILE="$LOG_DIR/gitbak-$REPO_HASH.log"
fi

if [ "$DEBUG" = "true" ]; then
    echo "🔍 Debug logging enabled. Logs will be written to: $LOG_FILE"
    LOG_DIR=$(dirname "$LOG_FILE")
    if [ "$LOG_DIR" != "." ]; then
        if ! mkdir -p "$LOG_DIR" 2>/dev/null; then
            echo "⚠️ Failed to create log directory: $LOG_DIR"
            # Try using temp directory as fallback
            LOG_FILE="/tmp/gitbak-$REPO_HASH.log"
            echo "🔄 Using fallback log location: $LOG_FILE"
        fi
    fi
    echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] gitbak debug logging started" >"$LOG_FILE"
fi

log() {
    [ "$DEBUG" != "true" ] && return 0

    # Using POSIX-compatible variable scoping instead of 'local'
    _log_level="$1"
    shift
    # Separate declaration and assignment to avoid masking return values
    _log_msg=""
    _log_msg="$(date '+%Y-%m-%d %H:%M:%S') [$_log_level] $*"

    # Always echo errors to console, warnings if verbose
    if [ "$_log_level" = "ERROR" ] || { [ "$VERBOSE" = "true" ] && [ "$_log_level" = "WARNING" ]; }; then
        echo "$_log_msg"
    fi

    echo "$_log_msg" >>"$LOG_FILE"
}

check_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "❌ Error: Required command '$1' not found. Please install it and try again."
        exit 1
    fi
}

# Verify essential dependencies
check_command git
check_command grep
check_command sed
check_command shasum || check_command md5sum || {
    echo "❌ Error: Neither shasum nor md5sum found"
    exit 1
}

INTERVAL_MINUTES=${INTERVAL_MINUTES:-5}
if ! echo "$INTERVAL_MINUTES" | grep -q '^[0-9][0-9]*$' || [ "$INTERVAL_MINUTES" -lt 1 ]; then
    echo "⚠️  Warning: Invalid INTERVAL_MINUTES '$INTERVAL_MINUTES'. Using default of 5 minutes."
    log "WARNING" "Invalid INTERVAL_MINUTES '$INTERVAL_MINUTES', using default of 5 minutes"
    INTERVAL_MINUTES=5
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "❌ Error: Not a git repository. Please run this script from a git repository."
    log "ERROR" "Not in a git repository"
    exit 1
fi
log "INFO" "Git repository verified"

BRANCH_NAME=${BRANCH_NAME:-"gitbak-$(date +%Y%m%d-%H%M%S)"}
COMMIT_PREFIX=${COMMIT_PREFIX:-"[gitbak] Automatic checkpoint"}
CREATE_BRANCH=${CREATE_BRANCH:-"true"}
VERBOSE=${VERBOSE:-"true"}                    # Show/hide informational messages
SHOW_NO_CHANGES=${SHOW_NO_CHANGES:-"false"}   # Show messages when no changes detected
CONTINUE_SESSION=${CONTINUE_SESSION:-"false"} # Continue an existing gitbak session

EXIT_CODE=0

COMMITS_MADE=0
COUNTER=1 # Default starting counter
START_TIME=$(date +%s)
ORIGINAL_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
log "INFO" "Starting gitbak on branch: $ORIGINAL_BRANCH"

cleanup() {
    log "INFO" "Cleanup function called"

    # Clean up termination flag
    rm -f "$TERM_FLAG"

    # Clean up lock file
    if [ -f "$LOCK_FILE" ] && [ "$(cat "$LOCK_FILE" 2>/dev/null)" = "$$" ]; then
        log "INFO" "Cleaning up lock file for PID $$"
        rm -f "$LOCK_FILE"
    fi

    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))

    HOURS=$((DURATION / 3600))
    MINUTES=$(((DURATION % 3600) / 60))
    # Rename SECONDS to SECS to avoid conflict with special variable in some shells
    SECS=$((DURATION % 60))

    echo ""
    echo "---------------------------------------------"
    echo "📊 gitbak Session Summary"
    echo "---------------------------------------------"
    echo "✅ Total commits made: $COMMITS_MADE"
    echo "⏱️  Session duration: ${HOURS}h ${MINUTES}m ${SECS}s"
    if [ "$CREATE_BRANCH" = "true" ]; then
        echo "🌿 Working branch: $BRANCH_NAME"
        echo ""
        echo "To merge these changes to your original branch:"
        echo "  git checkout $ORIGINAL_BRANCH"
        echo "  git merge $BRANCH_NAME"
        echo ""
        echo "To squash all commits into one:"
        echo "  git checkout $ORIGINAL_BRANCH"
        echo "  git merge --squash $BRANCH_NAME"
        echo "  git commit -m \"Merged gitbak session\""
    else
        echo "🌿 Working branch: $ORIGINAL_BRANCH (unchanged)"
    fi

    if command -v git log >/dev/null; then
        echo ""
        echo "🔍 Branch visualization (last 10 commits):"
        echo "---------------------------------------------"
        # Try to use a more colorful format if possible
        if git log --graph --oneline --decorate --all --color=always -n 10 >/dev/null 2>&1; then
            git log --graph --oneline --decorate --all --color=always -n 10
        else
            # Fallback to simpler format
            git log --graph --oneline --decorate -n 10
        fi
    fi

    echo "---------------------------------------------"
    echo "🛑 gitbak terminated at $(date)"
    # Exit with the last error code or 0 if successful
    exit ${EXIT_CODE:-0}
}

# Create a termination flag file that we can check in the main loop
TERM_FLAG="/tmp/gitbak-term-$$.flag"
rm -f "$TERM_FLAG"

# Set up trap for Ctrl+C and other signals (including HUP for terminal disconnects)
# This ensures the cleanup function runs when the script exits for any reason,
# including unexpected termination like terminal disconnection
trap 'touch "$TERM_FLAG"; trap - EXIT; cleanup' INT TERM HUP
trap cleanup EXIT

# Create a unique lock file based on repository path to prevent multiple instances
# Trying to prevent race conditions and data corruption from concurrent gitbak processes
REPO_HASH=$(echo "$REPO_PATH" | shasum | cut -d' ' -f1)
LOCK_FILE="/tmp/gitbak-$REPO_HASH.lock"

# Create a temporary file with PID for atomic lock acquisition
TEMP_LOCK_FILE="${LOCK_FILE}.$$"

# Write current process ID to a temporary file first
echo $$ >"$TEMP_LOCK_FILE"

# Attempt to create a hard link to the real lock file, which is atomic
# Just employing common techniques for cross-process locking in shell scripts
if ln "$TEMP_LOCK_FILE" "$LOCK_FILE" 2>/dev/null; then
    # We got the lock
    rm -f "$TEMP_LOCK_FILE"
else
    rm -f "$TEMP_LOCK_FILE"
    PID=$(cat "$LOCK_FILE" 2>/dev/null || echo "unknown")

    if ps -p "$PID" >/dev/null 2>&1; then
        echo "❌ Error: Another gitbak instance is already running for this repository (PID: $PID)"
        log "ERROR" "Another gitbak instance is already running (PID: $PID)"
        exit 1
    else
        echo "⚠️  Warning: Found stale lock file. Removing it."
        log "WARNING" "Found stale lock file for PID $PID. Removing it."
        rm -f "$LOCK_FILE"

        TEMP_LOCK_FILE="${LOCK_FILE}.$$"
        echo $$ >"$TEMP_LOCK_FILE"

        if ln "$TEMP_LOCK_FILE" "$LOCK_FILE" 2>/dev/null; then
            rm -f "$TEMP_LOCK_FILE"
            log "INFO" "Successfully acquired lock after removing stale lock"
        else
            echo "❌ Error: Failed to acquire lock even after removing stale lock"
            log "ERROR" "Failed to acquire lock after removing stale lock"
            rm -f "$TEMP_LOCK_FILE"
            exit 1
        fi
    fi
fi

CURRENT_BRANCH=$(git branch --show-current)

if [ "$CONTINUE_SESSION" = "true" ]; then
    CREATE_BRANCH="false"
    echo "🔄 Continuing gitbak session on branch: $CURRENT_BRANCH"

    ESCAPED_PREFIX=$(echo "$COMMIT_PREFIX" | sed 's/[][\\/.*^$]/\\&/g')

    # Set pipefail to catch errors in multi-command pipelines so that the script
    # fails if any command in a pipeline is borked, not just the last one
    # shellcheck disable=SC3040
    set -o pipefail 2>/dev/null || log "WARNING" "pipefail not supported in this shell"

    # Get the highest commit number by examining commit messages
    # (sequential numbering) in order to continue from the last commit
    git log --pretty=format:"%s" >/tmp/gitbak-log.$$ 2>/dev/null
    GIT_LOG_EXIT_CODE=$?
    if [ $GIT_LOG_EXIT_CODE -ne 0 ]; then
        log "WARNING" "Failed to get git log history: exit code $GIT_LOG_EXIT_CODE"
        HIGHEST_NUM=0
    else
        # Extract commit numbers from previous gitbak commits
        # Steps: 1) Find lines with our prefix followed by a number
        #        2) Extract just the "#N" portion
        #        3) Extract just the numeric part
        HIGHEST_NUM=$(grep -E "$ESCAPED_PREFIX #[0-9]+" /tmp/gitbak-log.$$ | head -1 | grep -o "#[0-9]\+" | grep -o "[0-9]\+" || echo "0")
        log "INFO" "Detected highest commit number: $HIGHEST_NUM"
        rm -f /tmp/gitbak-log.$$
    fi

    # Disable pipefail to avoid affecting the rest of the script...
    # shellcheck disable=SC3040
    set +o pipefail 2>/dev/null

    # Set the counter to continue from the highest number found
    if [ -n "$HIGHEST_NUM" ] && [ "$HIGHEST_NUM" != "0" ]; then
        COUNTER=$((HIGHEST_NUM + 1))
        echo "ℹ️  Found previous commits - starting from commit #$COUNTER"
    else
        echo "ℹ️  No previous commits found with prefix '$COMMIT_PREFIX' - starting from commit #1"
    fi
elif [ "$CREATE_BRANCH" = "true" ]; then
    if [ -n "$(git status --porcelain)" ]; then
        echo "⚠️  Warning: You have uncommitted changes."
        echo "Would you like to commit them before creating the gitbak branch? (y/n)"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            git add .
            if ! git commit -m "Manual commit before starting gitbak session"; then
                echo "❌ Error: Failed to create initial commit. Please fix any git issues and try again."
                rm -f "$LOCK_FILE"
                exit 1
            fi
            echo "✅ Created initial commit"
        fi
    fi

    if git show-ref --verify --quiet refs/heads/"$BRANCH_NAME"; then
        echo "⚠️  Warning: Branch '$BRANCH_NAME' already exists."
        echo "Would you like to use a different branch name? (y/n)"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            # Oops! ...append seconds to make branch name unique
            BRANCH_NAME="$BRANCH_NAME-$(date +%H%M%S)"
            echo "🌿 Using new branch name: $BRANCH_NAME"
        fi
    fi

    if ! git checkout -b "$BRANCH_NAME"; then
        echo "❌ Error: Failed to create new branch. Please check your git configuration."
        rm -f "$LOCK_FILE"
        exit 1
    fi
    echo "🌿 Created and switched to new branch: $BRANCH_NAME"
else
    echo "🌿 Using current branch: $CURRENT_BRANCH"
fi

echo "🔄 gitbak started at $(date)"
echo "📂 Repository: $REPO_PATH"
echo "⏱️ Interval: $INTERVAL_MINUTES minutes"
echo "📝 Commit prefix: $COMMIT_PREFIX"
echo "🔊 Verbose mode: $VERBOSE"
echo "🔔 Show no-changes messages: $SHOW_NO_CHANGES"
echo "❓ Press Ctrl+C to stop and view session summary"

COUNTER=${COUNTER:-1}

# Main monitoring loop - runs continuously at specified intervals to:
# 1. Check for changes using "git status"
# 2. If changes exist, commit them with sequential numbering
# 3. Handle any errors during git operations with retry logic
# 4. Sleep for the configured interval before repeating
while true; do
    # Check if termination was requested
    if [ -f "$TERM_FLAG" ]; then
        log "INFO" "Termination flag detected, exiting loop"
        break
    fi
    # Check if there are changes (capturing the exit code)
    GIT_STATUS_OUTPUT=$(git status --porcelain 2>&1)
    GIT_STATUS_EXIT_CODE=$?
    if [ $GIT_STATUS_EXIT_CODE -ne 0 ]; then
        # Git status can fail if the repository is in a bad state or locked
        echo "❌ Error: Failed to check git status: $GIT_STATUS_OUTPUT"
        log "ERROR" "git status failed with exit code $GIT_STATUS_EXIT_CODE: $GIT_STATUS_OUTPUT"
        echo "Will retry in $INTERVAL_MINUTES minutes."
        sleep $((INTERVAL_MINUTES * 60))
        continue
    fi

    if [ -n "$GIT_STATUS_OUTPUT" ]; then
        # Non-empty output means there are uncommitted changes
        TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

        # Try to add and commit changes
        # Capture errors in case of gitignore issues or other problems ¯\_(ツ)_/¯
        GIT_ADD_OUTPUT=$(git add . 2>&1)
        GIT_ADD_EXIT_CODE=$?
        if [ $GIT_ADD_EXIT_CODE -ne 0 ]; then
            echo "⚠️  Warning: Failed to stage changes: $GIT_ADD_OUTPUT"
            log "WARNING" "git add failed with exit code $GIT_ADD_EXIT_CODE: $GIT_ADD_OUTPUT"
            echo "Will retry next interval."
            sleep $((INTERVAL_MINUTES * 60))
            continue
        fi

        # Create a secure temporary file with unique process ID
        COMMIT_OUTPUT="/tmp/gitbak-commit-output.$$"

        # Try to commit the changes while capturing output for error reporting
        git commit -m "$COMMIT_PREFIX #$COUNTER - $TIMESTAMP" 2>&1 | tee "$COMMIT_OUTPUT"
        GIT_COMMIT_EXIT_CODE=$?
        if [ $GIT_COMMIT_EXIT_CODE -eq 0 ]; then
            # Commit succeeded - update the counters
            echo "✅ Commit #$COUNTER created at $TIMESTAMP"
            log "INFO" "Successfully created commit #$COUNTER"
            COUNTER=$((COUNTER + 1))
            COMMITS_MADE=$((COMMITS_MADE + 1))
        else
            # Commit failed ...could happen with hooks, permissions, or some other sort of config issues...
            echo "⚠️  Warning: Failed to create commit:"
            cat "$COMMIT_OUTPUT"
            log "WARNING" "git commit failed with exit code $GIT_COMMIT_EXIT_CODE: $(cat "$COMMIT_OUTPUT" 2>/dev/null || echo "unknown error")"
            echo "Will retry next interval."
        fi
        rm -f "$COMMIT_OUTPUT"
    elif [ "$SHOW_NO_CHANGES" = "true" ] && [ "$VERBOSE" = "true" ]; then
        echo "ℹ️  No changes to commit at $(date +"%H:%M:%S")"
        log "INFO" "No changes to commit detected"
    fi

    # Wait for the configured interval before checking again, but check for termination every second
    # This ensures we respond to termination signals promptly
    # shellcheck disable=SC2034
    for i in $(seq 1 $((INTERVAL_MINUTES * 60))); do
        if [ -f "$TERM_FLAG" ]; then
            log "INFO" "Termination flag detected during sleep, exiting loop"
            break 2  # Break out of both the for loop and the while loop
        fi
        sleep 1
    done
done
