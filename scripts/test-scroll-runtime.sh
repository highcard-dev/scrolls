#!/bin/bash
# Runtime scroll testing

set -euo pipefail

SCROLL_PATH="${1}"
TIMEOUT="${TIMEOUT:-60}"  # Reduced to 60s for fail-fast

# Determine Docker image from release.yml
get_image() {
    local line=$(grep "druid registry push.*$SCROLL_PATH" .github/workflows/release.yml | head -1)
    if echo "$line" | grep -q "stable-nix-steamcmd"; then
        echo "artifacts.druid.gg/druid-team/druid:stable-nix-steamcmd"
    else
        echo "artifacts.druid.gg/druid-team/druid:stable-nix"
    fi
}

# Get ports from scroll.yaml AND release.yml (-p arguments)
get_ports() {
    local ports=()
    
    # From scroll.yaml
    local yaml_ports=$(grep "^\s*port:" "$SCROLL_PATH/scroll.yaml" 2>/dev/null | grep -oE "[0-9]+" || true)
    if [ -n "$yaml_ports" ]; then
        ports+=($yaml_ports)
    fi
    
    # From release.yml (-p arguments like "-p main=25565/tcp")
    local release_line=$(grep "druid registry push.*$SCROLL_PATH" .github/workflows/release.yml 2>/dev/null | head -1 || true)
    if [ -n "$release_line" ]; then
        local release_ports=$(echo "$release_line" | grep -oE '\-p [a-z]+=([0-9]+)' | grep -oE '[0-9]+' || true)
        if [ -n "$release_ports" ]; then
            ports+=($release_ports)
        fi
    fi
    
    # Return unique ports
    printf '%s\n' "${ports[@]}" | sort -u | head -10
}

# Check if any port is open (works for both TCP and UDP)
check_port() {
    local container_id=$1
    local port=$2
    # Try multiple methods (ss, netstat, lsof) - one should work
    docker exec "$container_id" sh -c "
        ss -tuln 2>/dev/null | grep -q ':$port ' || \
        netstat -tuln 2>/dev/null | grep -q ':$port ' || \
        (command -v lsof >/dev/null 2>&1 && lsof -i :$port 2>/dev/null | grep -q LISTEN)
    " 2>/dev/null
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
echo "---"

# Follow logs in background (suppress to reduce noise)
docker logs -f "$CONTAINER_ID" > /tmp/container-$$.log 2>&1 &
LOGS_PID=$!

# Check for ports
START=$(date +%s)
LAST_CHECK=0

while true; do
    ELAPSED=$(($(date +%s) - START))
    
    # Check if container is still running
    if ! docker ps -q --filter "id=$CONTAINER_ID" | grep -q .; then
        echo "FAIL: Container exited after ${ELAPSED}s"
        echo "Exit code: $(docker inspect $CONTAINER_ID --format='{{.State.ExitCode}}' 2>/dev/null || echo 'unknown')"
        echo "Last 30 lines of logs:"
        tail -30 /tmp/container-$$.log
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_SCROLL" /tmp/container-$$.log
        exit 1
    fi
    
    # Timeout check
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "FAIL: Timeout ${TIMEOUT}s reached"
        echo "Last 30 lines of logs:"
        tail -30 /tmp/container-$$.log
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        kill $LOGS_PID 2>/dev/null || true
        rm -rf "$TEMP_SCROLL" /tmp/container-$$.log
        exit 1
    fi
    
    # Check ports every 2 seconds (not every loop to reduce overhead)
    if [ $((ELAPSED - LAST_CHECK)) -ge 2 ]; then
        for port in "${PORTS[@]}"; do
            if check_port "$CONTAINER_ID" "$port"; then
                echo "PASS: Port $port open after ${ELAPSED}s"
                docker kill "$CONTAINER_ID" 2>/dev/null || true
                kill $LOGS_PID 2>/dev/null || true
                rm -rf "$TEMP_SCROLL" /tmp/container-$$.log
                exit 0
            fi
        done
        LAST_CHECK=$ELAPSED
    fi
    
    sleep 1
done
