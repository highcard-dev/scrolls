#!/bin/bash
# Runtime scroll testing

set -euo pipefail

SCROLL_PATH="${1}"
TIMEOUT="${TIMEOUT:-600}"  # 10 minutes for downloads

# Skip scrolls with internal URLs (not accessible in CI)
if [ -f "$SCROLL_PATH/scroll.yaml" ] && grep -q "192.168." "$SCROLL_PATH/scroll.yaml"; then
    echo "SKIP: Internal URL (not accessible in CI)"
    exit 0
fi

# Determine Docker image from release.yml
get_image() {
    local line=$(grep "druid registry push.*$SCROLL_PATH" .github/workflows/release.yml | head -1)
    if echo "$line" | grep -q "stable-nix-steamcmd"; then
        echo "artifacts.druid.gg/druid-team/druid:stable-nix-steamcmd"
    else
        echo "artifacts.druid.gg/druid-team/druid:stable-nix"
    fi
}

# Get ports from scroll.yaml AND release.yml
get_ports() {
    local ports=()
    
    local yaml_ports=$(grep "^\s*port:" "$SCROLL_PATH/scroll.yaml" 2>/dev/null | grep -oE "[0-9]+" || true)
    if [ -n "$yaml_ports" ]; then
        ports+=($yaml_ports)
    fi
    
    local release_line=$(grep "druid registry push.*$SCROLL_PATH" .github/workflows/release.yml 2>/dev/null | head -1 || true)
    if [ -n "$release_line" ]; then
        local release_ports=$(echo "$release_line" | grep -oE '\-p [a-z]+=([0-9]+)' | grep -oE '[0-9]+' || true)
        if [ -n "$release_ports" ]; then
            ports+=($release_ports)
        fi
    fi
    
    printf '%s\n' "${ports[@]}" | sort -u | head -10
}

# Check if server started in logs (more reliable than port check)
check_server_started() {
    local container_id=$1
    local logs=$(docker logs "$container_id" 2>&1 | tail -200)
    
    # Minecraft servers log "Done" when ready
    echo "$logs" | grep -qE "(Done \(|Server started|RCON running|Starting.*server on)" && return 0
    
    # Check if still running
    docker ps -q --filter "id=$container_id" | grep -q . && return 1
    
    # Container exited
    return 2
}

# Main
if [ ! -f "$SCROLL_PATH/scroll.yaml" ]; then
    echo "SKIP: No scroll.yaml"
    exit 0
fi

PORTS=($(get_ports))
if [ ${#PORTS[@]} -eq 0 ]; then
    echo "SKIP: No ports defined"
    exit 0
fi

IMAGE=$(get_image)
if [ -z "$IMAGE" ]; then
    echo "SKIP: No image"
    exit 0
fi

echo "Scroll: $SCROLL_PATH"
echo "Image: $IMAGE"  
echo "Ports: ${PORTS[*]}"
echo "Timeout: ${TIMEOUT}s"

docker pull "$IMAGE" || {
    echo "SKIP: Failed to pull image"
    exit 0
}

# Create temp directory with correct structure:
# /tmp/test-XXXXX/
#   .scroll/        <- mounted to /scroll/.scroll
#     scroll.yaml
#     packet_handler/
#     init-files/
#     etc.
TEMP_DIR=$(mktemp -d)
mkdir -p "$TEMP_DIR/.scroll"
cp -r "$SCROLL_PATH/"* "$TEMP_DIR/.scroll/"
chmod -R 777 "$TEMP_DIR"

echo "Starting container with druid serve..."

# Mount the temp dir to /scroll
# druid will look for /scroll/.scroll/scroll.yaml
CONTAINER_ID=$(docker run --rm -d \
    -v "$TEMP_DIR:/scroll" \
    -w /scroll \
    "$IMAGE" \
    serve)

echo "Container: $CONTAINER_ID"
echo "---"

# Follow logs in background
docker logs -f "$CONTAINER_ID" > /tmp/container-$$.log 2>&1 &
LOGS_PID=$!

START=$(date +%s)

while true; do
    ELAPSED=$(($(date +%s) - START))
    
    # Check server status
    check_server_started "$CONTAINER_ID"
    STATUS=$?
    
    if [ $STATUS -eq 0 ]; then
        echo "---"
        echo "PASS: Server started after ${ELAPSED}s"
        docker logs "$CONTAINER_ID" 2>&1 | tail -30
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_DIR" /tmp/container-$$.log
        exit 0
    elif [ $STATUS -eq 2 ]; then
        echo "---"
        echo "FAIL: Container exited after ${ELAPSED}s"
        echo "Last 40 lines of logs:"
        tail -40 /tmp/container-$$.log
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_DIR" /tmp/container-$$.log
        exit 1
    fi
    
    # Timeout check
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "---"
        echo "FAIL: Timeout ${TIMEOUT}s reached"
        echo "Last 40 lines of logs:"
        tail -40 /tmp/container-$$.log
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_DIR" /tmp/container-$$.log
        exit 1
    fi
    
    sleep 2
done
