#!/bin/bash
echo "Test 4: Stress test with rapid commits"

ORIGINAL_DIR=$(cd "$(dirname "$0")/.." && pwd)

TEST_DIR=$(mktemp -d)
cd "$TEST_DIR" || exit 1

git init
echo "Initial content" >test.txt
git add test.txt
git commit -m "Initial commit"

cp "$ORIGINAL_DIR/gitbak.sh" .
chmod +x gitbak.sh

INTERVAL_MINUTES=1 SHOW_NO_CHANGES=true DEBUG=true ./gitbak.sh >output.log 2>&1 &
GITBAK_PID=$!

# Make many rapid changes
for i in $(seq 1 5); do
    echo "Change $i" >>test.txt
    sleep 3 # Shorter than the commit interval
done

# Let gitbak catch up
# Wait longer to allow multiple commit cycles
sleep 90
kill -TERM $GITBAK_PID
wait $GITBAK_PID

COMMIT_COUNT=$(git log --grep "\[gitbak\]" | grep -c "commit")
if [ "$COMMIT_COUNT" -ge 2 ]; then
    echo "✅ Stress test passed: $COMMIT_COUNT commits created"
else
    echo "❌ Stress test failed: Only $COMMIT_COUNT commits created"
    echo "Log:"
    cat output.log
fi

cd - >/dev/null || exit 1
rm -rf "$TEST_DIR"
