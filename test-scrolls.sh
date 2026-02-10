#!/bin/bash
# Test published scrolls by running druid serve and checking if mandatory ports open

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_RESULTS_DIR="${SCRIPT_DIR}/test-results"
TIMEOUT_SECONDS=60
DRUID_BIN="${DRUID_BIN:-druid}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Statistics
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

mkdir -p "${TEST_RESULTS_DIR}"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Get published scrolls from release.yml
get_published_scrolls() {
    local release_yml="${SCRIPT_DIR}/.github/workflows/release.yml"
    
    if [ ! -f "$release_yml" ]; then
        log_error "release.yml not found at $release_yml"
        exit 1
    fi
    
    # Extract scroll paths from matrix
    grep "scroll:" "$release_yml" | sed 's/.*scroll:\s*//' | tr -d '"' | tr -d "'"
}

# Parse scroll.yaml to extract ports
parse_ports() {
    local scroll_file="$1"
    grep -A 100 "^ports:" "$scroll_file" | grep "^\s*port:" | sed 's/.*port:\s*//' || echo ""
}

# Check if port is open
check_port() {
    local port="$1"
    local timeout="$2"
    
    timeout "$timeout" bash -c "until nc -z localhost $port 2>/dev/null; do sleep 1; done" 2>/dev/null
    return $?
}

# Test a single scroll
test_scroll() {
    local scroll_path="$1"
    local scroll_dir="${SCRIPT_DIR}/${scroll_path}"
    local scroll_yaml="${scroll_dir}/scroll.yaml"
    
    if [ ! -f "$scroll_yaml" ]; then
        log_error "No scroll.yaml found at ${scroll_yaml}"
        return 1
    fi
    
    log_info "Testing: ${scroll_path}"
    
    # Parse ports from scroll.yaml
    local ports=$(parse_ports "$scroll_yaml")
    
    if [ -z "$ports" ]; then
        log_warn "No ports defined in ${scroll_path}, skipping"
        return 2
    fi
    
    log_info "  Ports to check: ${ports}"
    
    # Start druid serve in background
    cd "$scroll_dir"
    
    local log_file="${TEST_RESULTS_DIR}/${scroll_path//\//_}.log"
    mkdir -p "$(dirname "$log_file")"
    
    log_info "  Starting druid serve..."
    $DRUID_BIN serve > "$log_file" 2>&1 &
    local druid_pid=$!
    
    # Wait a bit for startup
    sleep 5
    
    # Check if process is still running
    if ! kill -0 $druid_pid 2>/dev/null; then
        log_error "  druid serve exited immediately"
        log_error "  Log: $log_file"
        return 1
    fi
    
    # Check each port
    local all_ports_ok=true
    for port in $ports; do
        log_info "  Checking port ${port}..."
        if check_port "$port" "$TIMEOUT_SECONDS"; then
            log_info "  ✓ Port ${port} is open"
        else
            log_error "  ✗ Port ${port} failed to open within ${TIMEOUT_SECONDS}s"
            all_ports_ok=false
        fi
    done
    
    # Cleanup: kill druid process
    log_info "  Stopping druid..."
    kill $druid_pid 2>/dev/null || true
    wait $druid_pid 2>/dev/null || true
    
    # Give it a moment to clean up
    sleep 2
    
    if [ "$all_ports_ok" = true ]; then
        log_info "  ✓ PASSED: ${scroll_path}"
        return 0
    else
        log_error "  ✗ FAILED: ${scroll_path}"
        return 1
    fi
}

# Main test function
main() {
    log_info "Starting scroll tests (PUBLISHED SCROLLS ONLY)..."
    log_info "Druid binary: $DRUID_BIN"
    log_info "Test results: $TEST_RESULTS_DIR"
    
    # Check if druid is available
    if ! command -v "$DRUID_BIN" &> /dev/null; then
        log_error "druid binary not found: $DRUID_BIN"
        log_error "Set DRUID_BIN environment variable or install druid"
        exit 1
    fi
    
    # Check if nc (netcat) is available for port checking
    if ! command -v nc &> /dev/null; then
        log_error "netcat (nc) not found, required for port checking"
        exit 1
    fi
    
    # Get published scrolls from release.yml
    local published_scrolls=$(get_published_scrolls)
    local scroll_count=$(echo "$published_scrolls" | wc -l)
    
    log_info "Found ${scroll_count} published scrolls to test (from release.yml)"
    
    # Test each scroll
    for scroll_path in $published_scrolls; do
        TOTAL=$((TOTAL + 1))
        
        if test_scroll "$scroll_path"; then
            PASSED=$((PASSED + 1))
        else
            local exit_code=$?
            if [ $exit_code -eq 2 ]; then
                SKIPPED=$((SKIPPED + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        fi
        
        # Return to script directory
        cd "$SCRIPT_DIR"
        
        echo ""
    done
    
    # Print summary
    echo "========================================"
    log_info "Test Summary:"
    log_info "  Total:   $TOTAL"
    log_info "  Passed:  $PASSED"
    log_info "  Failed:  $FAILED"
    log_info "  Skipped: $SKIPPED"
    echo "========================================"
    
    if [ $FAILED -gt 0 ]; then
        log_error "Some tests failed!"
        exit 1
    else
        log_info "All tests passed!"
        exit 0
    fi
}

# Run main function
main
