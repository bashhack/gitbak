#!/bin/bash
# Test on different shells
echo "Test 5: Shell portability test"

# Get the path to the gitbak script (one directory up from tests)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

for SHELL_CMD in bash dash sh zsh; do
    if command -v $SHELL_CMD >/dev/null 2>&1; then
        echo "Testing on $SHELL_CMD..."
        if $SHELL_CMD -n "$GITBAK_SCRIPT"; then
            echo "✅ Syntax valid in $SHELL_CMD"
        else
            echo "❌ Syntax invalid in $SHELL_CMD"
        fi
    else
        echo "⚠️ $SHELL_CMD not available for testing"
    fi
done