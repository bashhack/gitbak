#!/bin/bash
echo "Test: Shell compatibility"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

# List of shells to test
SHELLS="bash dash sh zsh"

# Track results
all_passed=true

echo "Checking syntax compatibility across multiple shells..."

for SHELL_CMD in $SHELLS; do
    if command -v "$SHELL_CMD" >/dev/null 2>&1; then
        echo "Testing on $SHELL_CMD..."
        if $SHELL_CMD -n "$GITBAK_SCRIPT" 2>/tmp/shell_syntax.log; then
            echo "✅ Syntax valid in $SHELL_CMD"
        else
            echo "❌ Syntax invalid in $SHELL_CMD:"
            cat /tmp/shell_syntax.log
            all_passed=false
        fi
    else
        echo "⚠️ $SHELL_CMD not available for testing"
    fi
done

# Output a final result
if [ "$all_passed" = true ]; then
    echo "✅ Shell compatibility test passed"
    exit 0
else
    echo "❌ Shell compatibility test failed"
    exit 1
fi
