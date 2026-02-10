#!/bin/bash
# Runtime scroll testing - starts server and checks if any port opens

set -euo pipefail

SCROLL_DIR="${1:-.}"
TIMEOUT="${TIMEOUT:-180}"
CHECK_INTERVAL=5

# Get all defined ports from scroll.yaml
get_ports() {
    if command -v yq &> /dev/null; then
        yq eval '.ports[].port' "$SCROLL_DIR/scroll.yaml" 2>/dev/null | grep -v '^$' || echo ""
    else
        grep -A 1 "port:" "$SCROLL_DIR/scroll.yaml" | grep -E "^\s+[0-9]+" | awk '{print $NF}' || echo ""
    fi
}

# Check if any port is listening
check_ports() {
    local ports=($1)
    for port in "${ports[@]}"; do
        if ss -ltn 2>/dev/null | grep -q ":$port " || netstat -ltn 2>/dev/null | grep -q ":$port "; then
            echo "$port"
            return 0
        fi
    done
    return 1
}

# Main
cd "$SCROLL_DIR"

PORTS=($(get_ports))

if [ ${#PORTS[@]} -eq 0 ]; then
    echo "SKIP: No ports defined"
    exit 0
fi

echo "Ports: ${PORTS[*]}"
echo "Starting druid serve..."

druid serve --port 8081 > /tmp/druid-serve.log 2>&1 &
DRUID_PID=$!

START_TIME=$(date +%s)

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "FAIL: Timeout after ${TIMEOUT}s"
        tail -50 /tmp/druid-serve.log
        kill $DRUID_PID 2>/dev/null || true
        exit 1
    fi
    
    if OPEN_PORT=$(check_ports "${PORTS[*]}"); then
        echo "PASS: Port $OPEN_PORT open after ${ELAPSED}s"
        kill $DRUID_PID 2>/dev/null || true
        exit 0
    fi
    
    sleep $CHECK_INTERVAL
done
