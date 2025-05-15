# After Your gitbak Session

This document describes what to do with your gitbak commits after you've finished your development session.

## Session Summary

When you end your gitbak session (by pressing Ctrl+C), a summary is displayed showing:

- Total number of commits made
- Session duration
- Working branch name
- Suggested next steps

## Integrating Your Changes

When your pairing session is complete, you have several options for what to do with the gitbak branch (with the following using `main` by way of example):

> **Note:** If you've been using the "Manual + Auto" workflow (with `-no-branch` flag and manual milestone commits interspersed with gitbak's automatic commits), you may want to keep only your manual commits and discard the automatic ones. This can be done using interactive rebase: `git rebase -i main` and keeping only the commits with meaningful messages.

### 1. Squash All Commits into One

This is the most common approach for integrating gitbak changes. It combines all the automatic checkpoint commits into a single, meaningful commit:

```bash
# Switch back to your main branch
git checkout main

# Combine all gitbak commits into a single change set
git merge --squash gitbak-TIMESTAMP 

# Create a single, meaningful commit with all changes
git commit -m "Add feature X from pair programming session"
```

### 2. Cherry-pick Specific Changes

If you only want to keep some of the changes from your gitbak branch:

```bash
# Switch to your main branch
git checkout main

# Find the commit(s) you want
git log gitbak-TIMESTAMP

# Cherry-pick the specific commit(s) you want
git cherry-pick COMMIT_HASH
```

### 3. Merge the Branch As-Is

If you want to preserve the entire commit history:

```bash
# Switch to your main branch
git checkout main

# Merge the gitbak branch with all its history
git merge gitbak-TIMESTAMP
```

### 4. Discard the Branch

If you don't need the automatic commits anymore:

```bash
# Delete the branch locally
git branch -D gitbak-TIMESTAMP
```

## Reviewing Checkpoint History

Before deciding what to do with your gitbak branch, you might want to review the changes made at each checkpoint:

```bash
# View a graph of all commits
git log --graph --oneline --decorate gitbak-TIMESTAMP

# See what changed in each commit
git log --patch gitbak-TIMESTAMP

# Compare the branch to main
git diff main...gitbak-TIMESTAMP
```

## Continuing a Session Later

If you need to continue working on the same feature in another session:

```bash
# Later, when you want to resume
gitbak -continue
```

This will:
1. Find the last commit number used
2. Continue numbering from where you left off
3. Use the same branch you were on before