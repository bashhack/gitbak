#!/bin/bash
echo "Test 3: Session continuation"

ORIGINAL_DIR=$(cd "$(dirname "$0")/.." && pwd)

TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
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
sleep 20
kill -TERM $GITBAK_PID
wait $GITBAK_PID

CONTINUE_SESSION=true INTERVAL_MINUTES=1 DEBUG=true ./gitbak.sh &
GITBAK_PID=$!
sleep 10
echo "Change 2" >>test.txt
sleep 20
kill -TERM $GITBAK_PID
wait $GITBAK_PID

COMMIT_MSGS=$(git log --pretty=format:"%s" | grep -E "\[gitbak\] Automatic checkpoint #[0-9]+" | sort -r)
FIRST_NUM=$(echo "$COMMIT_MSGS" | head -1 | grep -o "#[0-9]\+" | grep -o "[0-9]\+")
SECOND_NUM=$(echo "$COMMIT_MSGS" | head -2 | tail -1 | grep -o "#[0-9]\+" | grep -o "[0-9]\+")

if [ "$FIRST_NUM" -gt "$SECOND_NUM" ]; then
    echo "✅ Continuation test passed: Commit numbering continues ($SECOND_NUM -> $FIRST_NUM)"
else
    echo "❌ Continuation test failed: Commit numbering doesn't continue ($SECOND_NUM -> $FIRST_NUM)"
    echo "Commit messages:"
    echo "$COMMIT_MSGS"
fi

cd - >/dev/null
rm -rf "$TEST_DIR"