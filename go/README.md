<p align="center">
  <img src="https://raw.githubusercontent.com/bashhack/gitbak/main/assets/gitbak_retro_logo.png" alt="gitbak logo" width="300">
</p>

# gitbak - Automatic Commit Safety Net

> Automated checkpoint commits during programming sessions.

## Purpose

When programming (with humans or AI assistants alike), the conversation and code changes can move quickly.

gitbak provides safety by:

- Allowing you to focus on coding without worrying about losing changes
- Creating automatic commits at regular intervals
- Making a clean history of your pairing session progress
- Providing recovery points if something goes wrong

This helps you avoid common pitfalls like the:

- _"I forgot to commit" panic_
- _"I thought that git command did something else" confusion_
- _"I lost my changes" frustration_
- _"I wish we could go back to that thread we pulled on thirty minutes ago" regret_

## Features

- **Automatic Commits** - Set and forget checkpoints at regular intervals
- **Branch Management** - Creates a dedicated branch or uses current one
- **Session Continuation** - Resume sessions with sequential commit numbering
- **Manual + Auto Workflow** - Combine automatic safety with manual milestone commits
- **Robust Error Handling** - Smart retry logic and signal handling
- **Platform Support** - Available for macOS and Linux systems

## 💡 Power Workflow: Manual + Automatic Commits

gitbak's most powerful capability is supporting a hybrid workflow - automatic safety with meaningful milestones:

```bash
# Start gitbak without creating a new branch
gitbak -no-branch

# While gitbak runs in the background creating safety checkpoints...
# When you reach a meaningful point in your work:
git add src/feature.go
git commit -m "Implement user authentication"

# Continue coding with automatic safety commits between your manual milestones
```

This gives you:
- Automatic safety net (even when you forget to commit)
- Clean, meaningful commit history for important milestones
- Intelligent commit numbering (gitbak preserves sequential numbering)
- The best of both worlds - safety and clarity

You can later use git tools to keep only your milestone commits and discard the automatic ones if desired.

> 💡 See [Comparison with Alternatives](docs/COMPARISON.md) for why this approach is superior to IDE auto-save features.

## Installation

```bash
# Option 1: Install with Homebrew (macOS and Linux)
brew install bashhack/gitbak/gitbak
# Note: Homebrew automatically adds gitbak to your PATH, so it's ready to use immediately

# Option 2: Install using Go (requires Go 1.24+)
go install github.com/bashhack/gitbak/go/cmd/gitbak@latest
# Note: Ensure your Go bin directory (typically $HOME/go/bin) is in your PATH
# You can add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):
# export PATH=$PATH:$HOME/go/bin

# Option 3: Build from source
git clone https://github.com/bashhack/gitbak.git
cd gitbak/go
make install
```

## Quick Start

```bash
# Navigate to your Git repository
cd /path/to/your/repo

# Start gitbak with default settings (5-minute commits)
gitbak

# Press Ctrl+C to stop when finished
```

## Configuration

```bash
# View all available options
gitbak -help

# Custom interval (2 minutes)
gitbak -interval 2

# Custom branch name
gitbak -branch "feature-work-backup"

# Continue a previous session
gitbak -continue

# Use the current branch
gitbak -no-branch
```

## After Your Session

```bash
# Squash all checkpoint commits into one
git checkout main
git merge --squash gitbak-TIMESTAMP 
git commit -m "Complete feature implementation"
```

## Documentation

For detailed documentation, see:

- [Usage and Configuration Guide](https://github.com/bashhack/gitbak/blob/main/go/docs/USAGE_AND_CONFIGURATION.md)
- [After Session Guide](https://github.com/bashhack/gitbak/blob/main/go/docs/AFTER_SESSION.md)
- [IDE Integration](https://github.com/bashhack/gitbak/blob/main/go/docs/IDE_INTEGRATION.md)
- [Comparison with Alternatives](https://github.com/bashhack/gitbak/blob/main/go/docs/COMPARISON.md)

## Development

```bash
# Clone the repository
git clone https://github.com/bashhack/gitbak.git
cd gitbak/go

# Run tests
make test

# Run tests in Ubuntu container (simulates GitHub Actions environment)
./scripts/test-all.sh

# Test specific packages in Ubuntu container
./scripts/ubuntu-test.sh ./internal/lock/...

# Build for development
make build

# Install locally
make install
```

See the [scripts README](scripts/README.md) for more information on testing in Ubuntu containers to catch platform-specific issues before they reach CI.

## License

MIT
