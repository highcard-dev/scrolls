#!/bin/bash
# Runtime scroll testing

set -euo pipefail

SCROLL_PATH="${1}"
TIMEOUT="${TIMEOUT:-180}"

# Determine Docker image from release.yml
get_image() {
    local line=$(grep "druid registry push.*$SCROLL_PATH" .github/workflows/release.yml | head -1)
    if echo "$line" | grep -q "stable-nix-steamcmd"; then
        echo "artifacts.druid.gg/druid-team/druid:stable-nix-steamcmd"
    else
        echo "artifacts.druid.gg/druid-team/druid:stable-nix"
    fi
}

# Get ports from scroll.yaml
get_ports() {
    grep "port:" "$SCROLL_PATH/scroll.yaml" | grep -oE "[0-9]+" | head -10
}

# Check if any port is open
check_port() {
    local container_id=$1
    local port=$2
    docker exec "$container_id" sh -c "ss -ltn 2>/dev/null | grep -q ':$port ' || netstat -ltn 2>/dev/null | grep -q ':$port '" 2>/dev/null
}

# Main
if [ ! -f "$SCROLL_PATH/scroll.yaml" ]; then
    echo "SKIP: No scroll.yaml"
    exit 0
fi

# Skip scrolls with internal URLs (not accessible in CI)
if grep -q "192.168." "$SCROLL_PATH/scroll.yaml"; then
    echo "SKIP: Internal URL (not accessible in CI)"
    exit 0
fi

PORTS=($(get_ports))
if [ ${#PORTS[@]} -eq 0 ]; then
    echo "SKIP: No ports"
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

docker pull "$IMAGE" || exit 0

# Copy scroll to temp location
TEMP_SCROLL="/tmp/test-scroll-$$"
mkdir -p "$TEMP_SCROLL"
cp -r "$SCROLL_PATH/"* "$TEMP_SCROLL/"
chmod -R 777 "$TEMP_SCROLL"

echo "Starting container..."

# Start container with temp scroll
CONTAINER_ID=$(docker run --rm -d \
    -v "$TEMP_SCROLL:/home/druid/.scroll" \
    "$IMAGE")

echo "Container: $CONTAINER_ID"

# Follow logs
docker logs -f "$CONTAINER_ID" 2>&1 &
LOGS_PID=$!

# Check for ports
START=$(date +%s)
while true; do
    ELAPSED=$(($(date +%s) - START))
    
    if ! docker ps -q --filter "id=$CONTAINER_ID" | grep -q .; then
        echo "FAIL: Container exited after ${ELAPSED}s"
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_SCROLL"
        exit 1
    fi
    
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "FAIL: Timeout ${TIMEOUT}s"
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_SCROLL"
        exit 1
    fi
    
    # Check ports every 5 seconds
    if [ $((ELAPSED % 5)) -eq 0 ]; then
        for port in "${PORTS[@]}"; do
            if check_port "$CONTAINER_ID" "$port"; then
                echo "PASS: Port $port open after ${ELAPSED}s"
                docker kill "$CONTAINER_ID" 2>/dev/null || true
                kill $LOGS_PID 2>/dev/null || true
                rm -rf "$TEMP_SCROLL"
                exit 0
            fi
        done
    fi
    
    sleep 1
done
