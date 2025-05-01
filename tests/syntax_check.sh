#!/bin/bash
# Simple syntax check for gitbak.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

echo "Checking syntax of gitbak.sh..."

# Check with bash
if bash -n "$GITBAK_SCRIPT" 2>/tmp/bash_syntax.log; then
    echo "✅ Bash syntax check passed"
else
    echo "❌ Bash syntax check failed:"
    cat /tmp/bash_syntax.log
fi

# Check with sh
if sh -n "$GITBAK_SCRIPT" 2>/tmp/sh_syntax.log; then
    echo "✅ Sh syntax check passed"
else
    echo "❌ Sh syntax check failed:"
    cat /tmp/sh_syntax.log
fi