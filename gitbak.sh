#!/bin/bash

# GitBak - Automatic Commit Safety Net for Pair Programming
# 
# This script automatically commits changes to git at regular intervals,
# providing a safety net for pair programming sessions or long coding sessions.
#
# Usage:
#   - Run from inside a Git repository: ./gitbak.sh
#   - Press Ctrl+C to stop and view the session summary
#
# Configuration (via environment variables):
#   - INTERVAL_MINUTES: Minutes between commits (default: 5)
#   - BRANCH_NAME: Custom branch name (default: gitbak-{timestamp})
#   - COMMIT_PREFIX: Custom commit message prefix (default: "[GitBak] Automatic checkpoint")
#   - CREATE_BRANCH: Whether to create a new branch (default: "true")
#   - VERBOSE: Show/hide informational messages (default: "true")
#   - SHOW_NO_CHANGES: Show messages when no changes detected (default: "false")
#   - REPO_PATH: Path to repository (default: current directory)

set -e  # Exit immediately if a command exits with a non-zero status

# Use environment variables if set, otherwise use defaults
REPO_PATH=${REPO_PATH:-$(pwd)}
INTERVAL_MINUTES=${INTERVAL_MINUTES:-5}
BRANCH_NAME=${BRANCH_NAME:-"gitbak-$(date +%Y%m%d-%H%M%S)"}
COMMIT_PREFIX=${COMMIT_PREFIX:-"[GitBak] Automatic checkpoint"}
CREATE_BRANCH=${CREATE_BRANCH:-"true"}
VERBOSE=${VERBOSE:-"true"}  # Show/hide informational messages
SHOW_NO_CHANGES=${SHOW_NO_CHANGES:-"false"}  # Show messages when no changes detected

# Initialize counters and stats
COMMITS_MADE=0
START_TIME=$(date +%s)
ORIGINAL_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")

# Function to display summary and clean up on exit
cleanup() {
  # Remove the lock file
  if [ -f "$LOCK_FILE" ]; then
    rm -f "$LOCK_FILE"
  fi

  END_TIME=$(date +%s)
  DURATION=$((END_TIME - START_TIME))
  
  # Format duration as HH:MM:SS
  HOURS=$((DURATION / 3600))
  MINUTES=$(( (DURATION % 3600) / 60 ))
  SECONDS=$((DURATION % 60))
  
  echo ""
  echo "---------------------------------------------"
  echo "📊 GitBak Session Summary"
  echo "---------------------------------------------"
  echo "✅ Total commits made: $COMMITS_MADE"
  echo "⏱️  Session duration: ${HOURS}h ${MINUTES}m ${SECONDS}s"
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
    echo "  git commit -m \"Merged GitBak session\""
  else
    echo "🌿 Working branch: $ORIGINAL_BRANCH (unchanged)"
  fi
  echo "---------------------------------------------"
  echo "🛑 GitBak terminated at $(date)"
  exit 0
}

# Set up trap for Ctrl+C and other signals
trap cleanup INT TERM EXIT

# Check if we're in a git repository
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "❌ Error: Not a git repository. Please run this script from a git repository."
    exit 1
fi

# Check for existing lock file to prevent multiple instances
LOCK_FILE="/tmp/gitbak-$(pwd | sed 's/\//-/g').lock"
if [ -f "$LOCK_FILE" ]; then
    PID=$(cat "$LOCK_FILE")
    if ps -p "$PID" > /dev/null; then
        echo "❌ Error: Another GitBak instance is already running for this repository (PID: $PID)"
        exit 1
    else
        echo "⚠️  Warning: Found stale lock file. Removing it."
        rm -f "$LOCK_FILE"
    fi
fi
echo $$ > "$LOCK_FILE"

# Create a new branch if requested
if [ "$CREATE_BRANCH" = "true" ]; then
    # Check if there are uncommitted changes
    if [ -n "$(git status --porcelain)" ]; then
        echo "⚠️  Warning: You have uncommitted changes."
        echo "Would you like to commit them before creating the GitBak branch? (y/n)"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            git add .
            if ! git commit -m "Manual commit before starting GitBak session"; then
                echo "❌ Error: Failed to create initial commit. Please fix any git issues and try again."
                rm -f "$LOCK_FILE"
                exit 1
            fi
            echo "✅ Created initial commit"
        fi
    fi
    
    # Create and checkout new branch
    if ! git checkout -b "$BRANCH_NAME"; then
        echo "❌ Error: Failed to create new branch. Please check your git configuration."
        rm -f "$LOCK_FILE"
        exit 1
    fi
    echo "🌿 Created and switched to new branch: $BRANCH_NAME"
else
    echo "🌿 Using current branch: $(git branch --show-current)"
fi

echo "🔄 GitBak started at $(date)"
echo "📂 Repository: $REPO_PATH"
echo "⏱️  Interval: $INTERVAL_MINUTES minutes"
echo "📝 Commit prefix: $COMMIT_PREFIX"
echo "🔊 Verbose mode: $VERBOSE"
echo "🔔 Show no-changes messages: $SHOW_NO_CHANGES"
echo "❓ Press Ctrl+C to stop and view session summary"

# Counter for commit numbering
COUNTER=1

while true; do
    # Check if there are changes (capture the exit code)
    GIT_STATUS_OUTPUT=$(git status --porcelain 2>&1) || {
        echo "❌ Error: Failed to check git status: $GIT_STATUS_OUTPUT"
        echo "Will retry in $INTERVAL_MINUTES minutes."
        sleep "${INTERVAL_MINUTES}m"
        continue
    }
    
    if [[ -n "$GIT_STATUS_OUTPUT" ]]; then
        # There are changes to commit
        TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
        
        # Try to add and commit changes
        GIT_ADD_OUTPUT=$(git add . 2>&1) || {
            echo "⚠️  Warning: Failed to stage changes: $GIT_ADD_OUTPUT"
            echo "Will retry next interval."
            sleep "${INTERVAL_MINUTES}m"
            continue
        }
        
        # Try to commit the changes
        GIT_COMMIT_OUTPUT=$(git commit -m "$COMMIT_PREFIX #$COUNTER - $TIMESTAMP" 2>&1)
        if [ $? -eq 0 ]; then
            echo "✅ Commit #$COUNTER created at $TIMESTAMP"
            ((COUNTER++))
            ((COMMITS_MADE++))
        else
            echo "⚠️  Warning: Failed to create commit:"
            echo "$GIT_COMMIT_OUTPUT"
            echo "Will retry next interval."
        fi
    elif [ "$SHOW_NO_CHANGES" = "true" ] && [ "$VERBOSE" = "true" ]; then
        echo "ℹ️  No changes to commit at $(date +"%H:%M:%S")"
    fi
    
    # Wait for the next interval
    sleep "${INTERVAL_MINUTES}m"
done