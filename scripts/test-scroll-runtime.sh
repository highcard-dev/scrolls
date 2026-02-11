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

# Setup temp directory with scroll
TEMP_DIR=$(mktemp -d)
mkdir -p "$TEMP_DIR/.scroll"
cp -r "$SCROLL_PATH/"* "$TEMP_DIR/.scroll/"
# Copy plugins if they exist
[ -f /tmp/druid_rcon ] && cp /tmp/druid_rcon "$TEMP_DIR/"
[ -f /tmp/druid_rcon_web_rust ] && cp /tmp/druid_rcon_web_rust "$TEMP_DIR/"
cd "$TEMP_DIR"

cleanup() { kill "$DRUID_PID" 2>/dev/null || true; cd /; rm -rf "$TEMP_DIR"; }
trap cleanup EXIT

echo "Working directory: $TEMP_DIR"
echo "--- Druid Output ---"

# Run druid serve and stream output
$DRUID_BIN serve 2>&1 | while IFS= read -r line; do
    echo "$line"
    if echo "$line" | grep -qE "$SUCCESS_PATTERNS"; then
        echo "--- PASS: Server started ---"
        exit 0
    fi
done &
DRUID_PID=$!

# Wait with timeout
START=$(date +%s)
while kill -0 $DRUID_PID 2>/dev/null; do
    if [ $(( $(date +%s) - START )) -ge "$TIMEOUT" ]; then
        echo "--- FAIL: Timeout ---"
        exit 1
    fi
    sleep 1
done

# Check exit status
wait $DRUID_PID && exit 0 || { echo "--- FAIL: Druid exited ---"; exit 1; }
