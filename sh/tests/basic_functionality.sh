#!/bin/bash
echo "Test 1: Basic functionality"

ORIGINAL_DIR=$(cd "$(dirname "$0")/.." && pwd)

TEST_DIR=$(mktemp -d)
cd "$TEST_DIR" || exit 1

git init
echo "Initial content" >test.txt
git add test.txt
git commit -m "Initial commit"

cp "$ORIGINAL_DIR/gitbak.sh" .
chmod +x gitbak.sh

INTERVAL_MINUTES=1 DEBUG=true ./gitbak.sh &
GITBAK_PID=$!

sleep 10
echo "Change 1" >>test.txt
sleep 30
echo "Change 2" >>test.txt
sleep 30
echo "Change 3" >>test.txt
sleep 10

kill -TERM $GITBAK_PID
wait $GITBAK_PID

COMMIT_COUNT=$(git log --grep "\[gitbak\]" | grep -c "commit")
if [ "$COMMIT_COUNT" -ge 2 ]; then
    echo "✅ Basic functionality test passed: $COMMIT_COUNT commits created"
else
    echo "❌ Basic functionality test failed: Only $COMMIT_COUNT commits created"
fi

cd - >/dev/null || exit 1
rm -rf "$TEST_DIR"
