#!/bin/bash
# Runtime scroll testing using Docker

set -euo pipefail

SCROLL_PATH="${1}"
TIMEOUT="${TIMEOUT:-180}"
CHECK_INTERVAL=5

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

# Check if any port is open inside container
check_ports_in_container() {
    local container_id=$1
    shift
    local ports=("$@")
    
    for port in "${ports[@]}"; do
        if docker exec "$container_id" sh -c "ss -ltn 2>/dev/null | grep -q ':$port ' || netstat -ltn 2>/dev/null | grep -q ':$port '" 2>/dev/null; then
            echo "$port"
            return 0
        fi
    done
    return 1
}

# Main
if [ ! -f "$SCROLL_PATH/scroll.yaml" ]; then
    echo "SKIP: No scroll.yaml found at $SCROLL_PATH"
    exit 0
fi

PORTS=($(get_ports))

if [ ${#PORTS[@]} -eq 0 ]; then
    echo "SKIP: No ports defined in scroll.yaml"
    exit 0
fi

IMAGE=$(get_image)

if [ -z "$IMAGE" ]; then
    echo "SKIP: Could not determine image from release.yml"
    exit 0
fi

echo "Scroll: $SCROLL_PATH"
echo "Image: $IMAGE"
echo "Ports: ${PORTS[*]}"

# Pull image
echo "Pulling image: $IMAGE"
if ! docker pull "$IMAGE"; then
    echo "SKIP: Cannot pull image $IMAGE"
    exit 0
fi

# Start container - mount scroll to /home/druid/.scroll
echo "Starting container..."
echo "Mounting: $(pwd)/$SCROLL_PATH -> /home/druid/.scroll"

CONTAINER_ID=$(docker run --rm -d \
    -v "$(pwd)/$SCROLL_PATH:/home/druid/.scroll" \
    "$IMAGE")

echo "Container: $CONTAINER_ID"

START_TIME=$(date +%s)

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    # Check if container is still running
    if ! docker ps -q --filter "id=$CONTAINER_ID" | grep -q .; then
        echo "FAIL: Container exited after ${ELAPSED}s"
        echo "=== Container logs ==="
        docker logs "$CONTAINER_ID" 2>&1 | tail -100
        exit 1
    fi
    
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "FAIL: Timeout after ${TIMEOUT}s"
        echo "=== Container logs ==="
        docker logs "$CONTAINER_ID" 2>&1 | tail -100
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        exit 1
    fi
    
    if OPEN_PORT=$(check_ports_in_container "$CONTAINER_ID" "${PORTS[@]}"); then
        echo "PASS: Port $OPEN_PORT open after ${ELAPSED}s"
        docker kill "$CONTAINER_ID" 2>/dev/null || true
        exit 0
    fi
    
    echo "  Waiting... ${ELAPSED}s elapsed"
    sleep $CHECK_INTERVAL
done
