#!/bin/bash
# Stress test with rapid commits
echo "Test 4: Stress test with rapid commits"
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
git init
echo "Initial content" > test.txt
git add test.txt
git commit -m "Initial commit"

# Get the path to the gitbak script (one directory up from tests)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

# Run GitBak with very short intervals
INTERVAL_MINUTES=1 SHOW_NO_CHANGES=true DEBUG=true "$GITBAK_SCRIPT" > output.log 2>&1 &
GITBAK_PID=$!

# Make many rapid changes
for i in $(seq 1 5); do
    echo "Change $i" >> test.txt
    sleep 3  # Shorter than the commit interval
done

# Let GitBak catch up
sleep 30
kill -TERM $GITBAK_PID
wait $GITBAK_PID

# Verify rapid changes were committed properly
COMMIT_COUNT=$(git log --grep "\[GitBak\]" | grep -c "commit")
if [ "$COMMIT_COUNT" -ge 2 ]; then
    echo "✅ Stress test passed: $COMMIT_COUNT commits created"
else
    echo "❌ Stress test failed: Only $COMMIT_COUNT commits created"
    echo "Log:"
    cat output.log
fi

cd - > /dev/null
rm -rf "$TEST_DIR"