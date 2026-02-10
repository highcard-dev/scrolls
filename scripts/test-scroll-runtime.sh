#!/bin/bash
# Runtime Scroll Testing Script
# Tests each scroll by actually starting it with druid serve and checking ports

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCROLLS_DIR="$REPO_ROOT/scrolls"
DRUID_CLI="${DRUID_CLI:-druid}"
RESULTS_FILE="/tmp/scroll-runtime-results.txt"
STARTUP_TIMEOUT=180  # 3 minutes max per scroll
PORT_CHECK_INTERVAL=5  # Check ports every 5 seconds

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Counters
total=0
passed=0
failed=0
skipped=0

# Initialize results
echo "Scroll Runtime Test Results - $(date)" > "$RESULTS_FILE"
echo "======================================" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Function to extract mandatory ports from scroll.yaml
get_mandatory_ports() {
    local scroll_yaml="$1"
    
    # Use yq if available, otherwise grep
    if command -v yq &> /dev/null; then
        yq eval '.ports[] | select(.mandatory == true) | .port' "$scroll_yaml" 2>/dev/null || echo ""
    else
        # Fallback to grep-based parsing
        awk '
            /^ports:/ { in_ports=1; next }
            in_ports && /^[^ ]/ { in_ports=0 }
            in_ports && /mandatory: true/ { 
                # Look backwards for port number
                for (i=NR-10; i<NR; i++) {
                    if (lines[i] ~ /port:/) {
                        match(lines[i], /port: *([0-9]+)/, arr)
                        print arr[1]
                        break
                    }
                }
            }
            { lines[NR] = $0 }
        ' "$scroll_yaml"
    fi
}

# Function to check if a port is listening
check_port() {
    local port="$1"
    ss -ltn 2>/dev/null | grep -q ":$port " || netstat -ltn 2>/dev/null | grep -q ":$port "
}

# Function to wait for all mandatory ports to be open
wait_for_ports() {
    local ports=("$@")
    local start_time=$(date +%s)
    
    while true; do
        local all_open=true
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        # Check timeout
        if [ $elapsed -ge $STARTUP_TIMEOUT ]; then
            echo "Timeout waiting for ports after ${STARTUP_TIMEOUT}s"
            return 1
        fi
        
        # Check all ports
        for port in "${ports[@]}"; do
            if ! check_port "$port"; then
                all_open=false
                echo "  Port $port not yet open (${elapsed}s elapsed)"
                break
            fi
        done
        
        if [ "$all_open" = true ]; then
            echo "  All ports open after ${elapsed}s"
            return 0
        fi
        
        sleep $PORT_CHECK_INTERVAL
    done
}

# Function to test a single scroll
test_scroll() {
    local scroll_dir="$1"
    local scroll_yaml="$scroll_dir/scroll.yaml"
    local rel_path=${scroll_dir#$SCROLLS_DIR/}
    
    echo -e "${YELLOW}Testing: $rel_path${NC}"
    
    # Check if scroll.yaml exists
    if [ ! -f "$scroll_yaml" ]; then
        echo -e "${YELLOW}  ⊘ No scroll.yaml, skipping${NC}"
        echo "SKIP: $rel_path - No scroll.yaml" >> "$RESULTS_FILE"
        return 2
    fi
    
    # Get mandatory ports
    local mandatory_ports=($(get_mandatory_ports "$scroll_yaml"))
    
    if [ ${#mandatory_ports[@]} -eq 0 ]; then
        echo -e "${YELLOW}  ⊘ No mandatory ports, skipping${NC}"
        echo "SKIP: $rel_path - No mandatory ports" >> "$RESULTS_FILE"
        return 2
    fi
    
    echo "  Mandatory ports: ${mandatory_ports[*]}"
    
    # Create temporary working directory
    local test_dir="/tmp/scroll-test-$(echo $rel_path | tr '/' '-')"
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    
    # Copy scroll files to test directory
    cp -r "$scroll_dir"/* "$test_dir/" 2>/dev/null || true
    
    # Start druid serve in background
    echo "  Starting druid serve..."
    cd "$test_dir"
    
    # Start druid serve (no coldstarter, port 8081 for API)
    local log_file="/tmp/druid-serve-$(echo $rel_path | tr '/' '-').log"
    $DRUID_CLI serve --port 8081 > "$log_file" 2>&1 &
    local druid_pid=$!
    
    echo "  Druid PID: $druid_pid"
    echo "  Log: $log_file"
    
    # Wait for ports to open
    echo "  Waiting for mandatory ports..."
    if wait_for_ports "${mandatory_ports[@]}"; then
        # Success!
        echo -e "${GREEN}  ✓ PASS - All mandatory ports listening${NC}"
        echo "PASS: $rel_path - Ports: ${mandatory_ports[*]}" >> "$RESULTS_FILE"
        
        # Kill druid
        kill $druid_pid 2>/dev/null || true
        wait $druid_pid 2>/dev/null || true
        
        # Cleanup
        cd "$REPO_ROOT"
        rm -rf "$test_dir"
        return 0
    else
        # Failed
        echo -e "${RED}  ✗ FAIL - Ports did not open within timeout${NC}"
        echo "FAIL: $rel_path - Timeout waiting for ports: ${mandatory_ports[*]}" >> "$RESULTS_FILE"
        echo "  Check log: $log_file"
        
        # Kill druid
        kill $druid_pid 2>/dev/null || true
        wait $druid_pid 2>/dev/null || true
        
        cd "$REPO_ROOT"
        return 1
    fi
}

# Main execution
echo "Scroll Runtime Testing"
echo "Repository: $REPO_ROOT"
echo "Druid CLI: $DRUID_CLI"
echo "======================================"
echo

# Check if druid CLI is available
if ! command -v $DRUID_CLI &> /dev/null; then
    echo -e "${RED}Error: druid CLI not found: $DRUID_CLI${NC}"
    echo "Set DRUID_CLI environment variable to the druid binary path"
    exit 1
fi

# Find all scroll directories
while IFS= read -r scroll_dir; do
    total=$((total + 1))
    
    if test_scroll "$scroll_dir"; then
        passed=$((passed + 1))
    elif [ $? -eq 2 ]; then
        skipped=$((skipped + 1))
    else
        failed=$((failed + 1))
    fi
    
    echo
done < <(find "$SCROLLS_DIR" -name "scroll.yaml" -type f -exec dirname {} \; | sort)

# Summary
echo "======================================" >> "$RESULTS_FILE"
echo "Total: $total | Passed: $passed | Failed: $failed | Skipped: $skipped" >> "$RESULTS_FILE"

echo -e "${YELLOW}======================================${NC}"
echo -e "${GREEN}✓ Passed: $passed${NC}"
echo -e "${RED}✗ Failed: $failed${NC}"
echo -e "${YELLOW}⊘ Skipped: $skipped${NC}"
echo -e "${YELLOW}━ Total: $total${NC}"
echo -e "${YELLOW}======================================${NC}"
echo
echo "Full results: $RESULTS_FILE"

# Exit with error if any failures
if [ $failed -gt 0 ]; then
    exit 1
else
    exit 0
fi
