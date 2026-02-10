#!/bin/bash
# Scroll Validation Script - CI Version
# Validates all scroll.yaml files in the repository

set -euo pipefail

# Get repository root directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCROLLS_DIR="$REPO_ROOT/scrolls"
RESULTS_FILE="/tmp/scroll-validation-results.txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Initialize counters
total=0
passed=0
failed=0

# Initialize results file
echo "Scroll Validation Results - $(date)" > "$RESULTS_FILE"
echo "======================================" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Function to validate a single scroll.yaml
validate_scroll() {
    local scroll_file="$1"
    local scroll_dir=$(dirname "$scroll_file")
    local rel_path=${scroll_file#$SCROLLS_DIR/}
    local display_name=$(dirname "$rel_path")
    
    local errors=()
    local warnings=()
    
    # Check if file is readable
    if [ ! -r "$scroll_file" ]; then
        errors+=("File not readable")
    fi
    
    # Check required fields
    if ! grep -q "^name:" "$scroll_file"; then
        errors+=("Missing required field: name")
    fi
    
    if ! grep -q "^desc:" "$scroll_file"; then
        errors+=("Missing required field: desc")
    fi
    
    if ! grep -q "^app_version:" "$scroll_file"; then
        errors+=("Missing required field: app_version")
    fi
    
    # Check ports section
    if ! grep -q "^ports:" "$scroll_file"; then
        warnings+=("No ports defined")
    else
        # Check for mandatory ports
        if ! grep -q "mandatory: true" "$scroll_file"; then
            warnings+=("No mandatory ports defined")
        fi
        
        # Check port definitions (look for port: followed by a number)
        while IFS= read -r line; do
            if echo "$line" | grep -q "^    port: "; then
                port_num=$(echo "$line" | sed 's/.*port: //' | tr -d ' ')
                if ! [[ "$port_num" =~ ^[0-9]+$ ]]; then
                    errors+=("Invalid port number: $port_num")
                elif [ "$port_num" -lt 1 ] || [ "$port_num" -gt 65535 ]; then
                    errors+=("Port number out of range: $port_num")
                fi
            fi
        done < "$scroll_file"
        
        # Check for sleep_handler files
        while IFS= read -r line; do
            if echo "$line" | grep -q "sleep_handler:"; then
                handler=$(echo "$line" | sed 's/.*sleep_handler: //' | tr -d ' ')
                handler_path="$scroll_dir/$handler"
                if [ ! -f "$handler_path" ]; then
                    errors+=("Sleep handler not found: $handler")
                fi
            fi
        done < "$scroll_file"
    fi
    
    # Check init field (should reference a command)
    if ! grep -q "^init:" "$scroll_file"; then
        warnings+=("No init command defined")
    fi
    
    # Check commands section
    if grep -q "^commands:" "$scroll_file"; then
        # Verify at least some commands exist
        if ! grep -A 5 "^commands:" "$scroll_file" | grep -q "  [a-z]"; then
            warnings+=("Commands section empty")
        fi
        
        # Check for procedures in commands (basic check)
        if ! grep -q "procedures:" "$scroll_file"; then
            warnings+=("No procedures defined in commands")
        fi
    else
        warnings+=("No commands section defined")
    fi
    
    # Check dependencies
    if grep -q "^dependencies:" "$scroll_file"; then
        # Just check if it's present, detailed validation would require YAML parsing
        :
    fi
    
    # Report results
    if [ ${#errors[@]} -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $display_name"
        echo "PASS: $display_name" >> "$RESULTS_FILE"
        
        # Show warnings
        for warning in "${warnings[@]}"; do
            echo -e "    ${YELLOW}⚠${NC} $warning"
            echo "  WARNING: $warning" >> "$RESULTS_FILE"
        done
        
        return 0
    else
        echo -e "${RED}✗${NC} $display_name"
        echo "FAIL: $display_name" >> "$RESULTS_FILE"
        
        for error in "${errors[@]}"; do
            echo -e "    ${RED}→${NC} $error"
            echo "  ERROR: $error" >> "$RESULTS_FILE"
        done
        
        return 1
    fi
}

# Main execution
echo "Scroll Validation Script"
echo "Repository: $REPO_ROOT"
echo "Scrolls directory: $SCROLLS_DIR"
echo "======================================"
echo

if [ ! -d "$SCROLLS_DIR" ]; then
    echo -e "${RED}Error: Scrolls directory not found: $SCROLLS_DIR${NC}"
    exit 1
fi

# Find and validate all scroll.yaml files
while IFS= read -r scroll_file; do
    total=$((total + 1))
    
    if validate_scroll "$scroll_file"; then
        passed=$((passed + 1))
    else
        failed=$((failed + 1))
    fi
    
done < <(find "$SCROLLS_DIR" -name "scroll.yaml" -type f | sort)

# Summary
echo
echo "======================================"
echo >> "$RESULTS_FILE"
echo "======================================" >> "$RESULTS_FILE"
echo "SUMMARY" >> "$RESULTS_FILE"
echo "======================================" >> "$RESULTS_FILE"
echo "Total: $total | Passed: $passed | Failed: $failed" >> "$RESULTS_FILE"

echo -e "${GREEN}✓ Passed:${NC} $passed"
echo -e "${RED}✗ Failed:${NC} $failed"
echo -e "Total: $total"
echo "======================================"
echo

if [ $failed -gt 0 ]; then
    echo -e "${RED}Validation failed with $failed errors${NC}"
    echo "See results file: $RESULTS_FILE"
    exit 1
else
    echo -e "${GREEN}All scrolls validated successfully!${NC}"
    exit 0
fi
