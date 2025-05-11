<p align="center">
  <img src="assets/gitbak_retro_logo.png" alt="gitbak logo" width="300">
</p>

<div align="center">

[![Tests](https://github.com/bashhack/gitbak/actions/workflows/ci.yml/badge.svg)](https://github.com/bashhack/gitbak/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/bashhack/gitbak/graph/badge.svg?token=Y3K7R3MHXH)](https://codecov.io/gh/bashhack/gitbak)
[![Go Reference](https://pkg.go.dev/badge/github.com/bashhack/gitbak)](https://pkg.go.dev/github.com/bashhack/gitbak)
![CodeRabbit Reviews](https://img.shields.io/coderabbit/prs/github/bashhack/gitbak?utm_source=oss&utm_medium=github&utm_campaign=bashhack%2Fgitbak&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

</div>

# gitbak - Automatic Commit Safety Net

> Automated checkpoint commits during programming sessions.

## 🎯 Purpose

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

## 💡 Best Practice: Manual + Automatic Workflow

A powerful workflow pattern with gitbak is combining automatic safety checkpoints with manual milestone commits:

```bash
# Start gitbak on your current branch
gitbak -no-branch

# While gitbak creates automatic commits, you can still:
git add <files>
git commit -m "Implement login feature"

# gitbak continues creating safety checkpoints while you create
# meaningful commits for important milestones
```

This gives you both a detailed safety net AND a clean, meaningful commit history - the best of both worlds!

## 🌟 Features

- **Automatic Commits** - Set and forget checkpoints at regular intervals
- **Branch Management** - Creates a dedicated branch or uses current one
- **Session Continuation** - Resume sessions with sequential commit numbering
- **Robust Error Handling** - Smart retry logic and signal handling
- **Platform Support** - Available for macOS and Linux systems

## 📦 Installation

### Go Version (Recommended)

```bash
# Option 1: Install with Homebrew (macOS and Linux)
brew install bashhack/gitbak/gitbak
# Note: Homebrew automatically adds gitbak to your PATH, so it's ready to use immediately

# Option 2: Install using Go (requires Go 1.24+)
go install github.com/bashhack/gitbak/go/cmd/gitbak@latest
# Note: Ensure your Go bin directory (typically $HOME/go/bin) is in your PATH
# You can add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):
# export PATH=$PATH:$HOME/go/bin

# Option 3: Download pre-built binary
# Visit: https://github.com/bashhack/gitbak/releases
```

### Shell Script Version

```bash
curl -sL https://github.com/bashhack/gitbak/releases/latest/download/gitbak-shell.tar.gz -o gitbak-shell.tar.gz
tar -xzf gitbak-shell.tar.gz
./gitbak-shell/install.sh
```

The install script will check if the installation directory (default: `~/.local/bin`) is in your PATH and provide instructions if it's not. For more details, see the [Shell Version Documentation](/sh/README.md).

## 🚀 Quick Start

```bash
# Navigate to your Git repository
cd /path/to/your/repo

# Start gitbak with default settings (5-minute commits)
gitbak

# Press Ctrl+C to stop when finished
```

## ⚙️ Configuration

```bash
# Custom interval (2 minutes)
gitbak -interval 2

# Custom branch name
gitbak -branch "feature-work-backup"

# Continue a previous session
gitbak -continue

# Use the current branch
gitbak -no-branch

# Full options list
gitbak -help
```

## 🔄 After Your Session

```bash
# Squash all checkpoint commits into one
git checkout main
git merge --squash gitbak-TIMESTAMP 
git commit -m "Complete feature implementation"
```

## 📚 Documentation

- [Go Version Documentation](/go/README.md) - Complete instructions for the Go implementation
- [Shell Script Version](/sh/README.md) - Documentation for the shell script implementation
- [Usage & Configuration](go/docs/USAGE_AND_CONFIGURATION.md) - Detailed usage instructions with workflow diagrams
- [After Session Guide](go/docs/AFTER_SESSION.md) - What to do when your session ends
- [IDE Integration](go/docs/IDE_INTEGRATION.md) - How to integrate with popular editors
- [Comparison with Alternatives](go/docs/COMPARISON.md) - Why gitbak outshines IDE auto-save features

## 📋 Implementation Comparison

| Feature         | Go Version (Recommended)                     | Shell Version                      |
|-----------------|----------------------------------------------|------------------------------------|
| Dependencies    | Git only                                     | Git, common Unix tools             |
| Platform        | macOS and Linux                              | Unix-like systems with POSIX shell |
| Configuration   | Command-line flags and environment variables | Environment variables              |
| Resource usage  | ~5-6 MB                                      | ~2-3 MB                            |

## 📄 License

MIT
