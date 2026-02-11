#!/bin/bash
# Runtime scroll testing - streams container output and detects server ready state

set -euo pipefail

SCROLL_PATH="${1:?Usage: $0 <scroll-path>}"
TIMEOUT="${TIMEOUT:-600}"
SUCCESS_PATTERNS="Done \(|Server started|RCON running|Starting.*server on"

# Skip internal URLs
grep -q "192.168." "$SCROLL_PATH/scroll.yaml" 2>/dev/null && { echo "SKIP: Internal URL"; exit 0; }

# Determine image from release.yml
IMAGE="artifacts.druid.gg/druid-team/druid:stable-nix"
grep -q "druid registry push.*$SCROLL_PATH.*stable-nix-steamcmd" .github/workflows/release.yml 2>/dev/null && \
    IMAGE="artifacts.druid.gg/druid-team/druid:stable-nix-steamcmd"

echo "=== Testing: $SCROLL_PATH ==="
echo "Image: $IMAGE | Timeout: ${TIMEOUT}s"

docker pull -q "$IMAGE" || { echo "SKIP: Failed to pull image"; exit 0; }

# Setup temp directory with scroll
TEMP_DIR=$(mktemp -d)
mkdir -p "$TEMP_DIR/.scroll"
cp -r "$SCROLL_PATH/"* "$TEMP_DIR/.scroll/"
chmod -R 777 "$TEMP_DIR"

cleanup() { docker kill "$CONTAINER_ID" 2>/dev/null || true; rm -rf "$TEMP_DIR"; }
trap cleanup EXIT

# Run container in background
CONTAINER_ID=$(docker run --rm -d -v "$TEMP_DIR:/scroll" -w /scroll "$IMAGE" serve)
echo "Container: ${CONTAINER_ID:0:12}"
echo "--- Container Output ---"

# Stream logs and check for success
START=$(date +%s)
docker logs -f "$CONTAINER_ID" 2>&1 | while IFS= read -r line; do
    echo "$line"
    if echo "$line" | grep -qE "$SUCCESS_PATTERNS"; then
        echo "--- PASS: Server started ($(( $(date +%s) - START ))s) ---"
        exit 0
    fi
    if [ $(( $(date +%s) - START )) -ge "$TIMEOUT" ]; then
        echo "--- FAIL: Timeout ---"
        exit 1
    fi
done &
LOG_PID=$!

# Wait for log reader or container to exit
while kill -0 $LOG_PID 2>/dev/null && docker ps -q --filter "id=$CONTAINER_ID" | grep -q .; do
    sleep 1
done

# Check exit status
wait $LOG_PID 2>/dev/null && exit 0
docker ps -q --filter "id=$CONTAINER_ID" | grep -q . || { echo "--- FAIL: Container exited ---"; exit 1; }
exit 1
