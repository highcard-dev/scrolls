#!/bin/bash
# Runtime scroll testing - streams druid output and detects server ready state

set -euo pipefail

SCROLL_PATH="${1:?Usage: $0 <scroll-path>}"
TIMEOUT="${TIMEOUT:-600}"
SUCCESS_PATTERNS="Done \(|Server started|RCON running|Starting.*server on"
DRUID_BIN="${DRUID_BIN:-/tmp/druid}"

# Skip internal URLs
grep -q "192.168." "$SCROLL_PATH/scroll.yaml" 2>/dev/null && { echo "SKIP: Internal URL"; exit 0; }

echo "=== Testing: $SCROLL_PATH ==="
echo "Druid: $DRUID_BIN | Timeout: ${TIMEOUT}s"

# Check if LGSM scroll and running as root
if [[ "$SCROLL_PATH" == *"/lgsm/"* ]] && [ "$(id -u)" -eq 0 ]; then
    echo "SKIP: LGSM scrolls cannot run as root"
    exit 0
fi

# Setup temp directory with scroll
TEMP_DIR=$(mktemp -d)
mkdir -p "$TEMP_DIR/.scroll"
cp -r "$SCROLL_PATH/"* "$TEMP_DIR/.scroll/"
# Copy plugins if they exist
[ -f /tmp/druid_rcon ] && cp /tmp/druid_rcon "$TEMP_DIR/"
[ -f /tmp/druid_rcon_web_rust ] && cp /tmp/druid_rcon_web_rust "$TEMP_DIR/"
cd "$TEMP_DIR"

cleanup() {
    echo "--- Cleanup: Stopping druid and all child processes ---"
    if [ -n "${DRUID_PID:-}" ]; then
        # Kill entire process group (druid + game server + any children)
        pkill -P "$DRUID_PID" 2>/dev/null || true
        kill "$DRUID_PID" 2>/dev/null || true
        # Wait for processes to exit
        sleep 2
        # Force kill if still running
        kill -9 "$DRUID_PID" 2>/dev/null || true
    fi
    # Extra safety: kill any remaining druid/java/server processes
    pkill -f "druid serve" 2>/dev/null || true
    pkill -f "minecraft.*server.jar" 2>/dev/null || true
    sleep 1
    cd /
    rm -rf "$TEMP_DIR"
    echo "--- Cleanup complete ---"
}
trap cleanup EXIT

echo "Working directory: $TEMP_DIR"
echo "--- Druid Output ---"

# Run druid serve in background
$DRUID_BIN serve > druid.log 2>&1 &
DRUID_PID=$!

# Monitor output with timeout
START=$(date +%s)
LAST_SIZE=0

while true; do
    ELAPSED=$(( $(date +%s) - START ))
    
    # Check if druid is still running
    if ! kill -0 $DRUID_PID 2>/dev/null; then
        echo "--- FAIL: Druid exited after ${ELAPSED}s ---"
        tail -50 druid.log
        exit 1
    fi
    
    # Show new output
    CURRENT_SIZE=$(wc -l < druid.log 2>/dev/null || echo 0)
    if [ "$CURRENT_SIZE" -gt "$LAST_SIZE" ]; then
        tail -n +$((LAST_SIZE + 1)) druid.log 2>/dev/null
        LAST_SIZE=$CURRENT_SIZE
    fi
    
    # Check scroll-lock.json for errors or pending state
    if [ -f ".scroll/scroll-lock.json" ]; then
        # Check for error state or error messages in JSON
        if grep -qiE '"state":\s*"error"|"error":|"fail":|"fatal":|"exception"' ".scroll/scroll-lock.json" 2>/dev/null; then
            echo "--- FAIL: Error detected in scroll-lock.json after ${ELAPSED}s ---"
            echo "scroll-lock.json contents:"
            cat ".scroll/scroll-lock.json"
            echo ""
            echo "Druid log:"
            tail -50 druid.log
            exit 1
        fi
        # Check for pending state (not progressing)
        if grep -qE '"state":\s*"pending"' ".scroll/scroll-lock.json" 2>/dev/null; then
            echo "WARNING: Scroll still in pending state at ${ELAPSED}s"
        fi
    fi
    
    # Check for success
    if grep -qE "$SUCCESS_PATTERNS" druid.log 2>/dev/null; then
        echo "--- PASS: Server started after ${ELAPSED}s ---"
        exit 0
    fi
    
    # Check for timeout
    if [ $ELAPSED -ge "$TIMEOUT" ]; then
        echo "--- FAIL: Timeout after ${ELAPSED}s ---"
        if [ -f ".scroll/scroll-lock.json" ]; then
            echo "scroll-lock.json contents:"
            cat ".scroll/scroll-lock.json"
            echo ""
        fi
        echo "Druid log:"
        tail -50 druid.log
        exit 1
    fi
    
    sleep 2
done
