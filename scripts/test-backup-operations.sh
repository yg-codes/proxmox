#!/bin/bash
# test-backup-operations.sh - Comprehensive test script for Proxmox Snapshot Manager backup operations
# Tests: BKUP-001 to BKUP-016 and DRY-001 to DRY-016

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GO_BIN="${GO_BIN:-./proxmox-admin-cli/build/proxmox-admin-cli}"
TEST_VM="${TEST_VM:-7300}"
TEST_STORAGE="${TEST_STORAGE:-local-zfs}"
RESULTS_DIR="test-results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_FILE="${RESULTS_DIR}/backup-ops-test-${TIMESTAMP}.md"

# Test counters
TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

# Arrays to track results
declare -a FAILED_TESTS
declare -a SKIPPED_TESTS

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_section() {
    echo -e "\n${YELLOW}>>> $1${NC}\n"
}

run_test() {
    local test_id=$1
    local description=$2
    local command=$3
    local expected=$4
    local check_type=${5:-"output"}  # output, exit_code, or file_exists

    TOTAL=$((TOTAL + 1))
    echo -e "${BLUE}[TEST ${test_id}]${NC} ${description}"
    echo "  Command: ${command}"

    # Execute command and capture output and exit code
    set +e
    output=$(eval "$command" 2>&1)
    exit_code=$?
    set -e

    # Determine pass/fail based on check type
    local passed=false
    case $check_type in
        "output")
            if [[ "$output" == *"$expected"* ]]; then
                passed=true
            fi
            ;;
        "exit_code")
            if [[ $exit_code -eq $expected ]]; then
                passed=true
            fi
            ;;
        "no_error")
            if [[ $exit_code -eq 0 ]]; then
                passed=true
            fi
            ;;
        "error")
            if [[ $exit_code -ne 0 ]]; then
                passed=true
            fi
            ;;
    esac

    # Report result
    if $passed; then
        echo -e "  ${GREEN}✅ PASS${NC}"
        PASSED=$((PASSED + 1))
        echo "✅ **PASS** | $test_id | $description" >> "$RESULTS_FILE"
    else
        echo -e "  ${RED}❌ FAIL${NC}"
        FAILED=$((FAILED + 1))
        FAILED_TESTS+=("$test_id: $description")
        echo "❌ **FAIL** | $test_id | $description" >> "$RESULTS_FILE"
        echo "  Output: $output" >> "$RESULTS_FILE"
        echo "  Exit Code: $exit_code" >> "$RESULTS_FILE"
    fi
    echo ""
}

skip_test() {
    local test_id=$1
    local description=$2
    local reason=$3

    TOTAL=$((TOTAL + 1))
    SKIPPED=$((SKIPPED + 1))
    SKIPPED_TESTS+=("$test_id: $description ($reason)")

    echo -e "${BLUE}[TEST ${test_id}]${NC} ${description}"
    echo -e "  ${YELLOW}⏭️  SKIPPED${NC} - $reason"
    echo "⏭️  **SKIP** | $test_id | $description | Reason: $reason" >> "$RESULTS_FILE"
    echo ""
}

check_prerequisites() {
    print_header "Checking Prerequisites"

    # Check if Go binary exists
    if [[ ! -f "$GO_BIN" ]]; then
        echo -e "${RED}❌ ERROR: Go binary not found at $GO_BIN${NC}"
        echo "Please build the Go implementation first:"
        echo "  cd proxmox-admin-cli && make build"
        exit 1
    fi
    echo -e "${GREEN}✅ Go binary found: $GO_BIN${NC}"

    # Check environment variables
    if [[ -z "$PVE_HOST" ]]; then
        echo -e "${RED}❌ ERROR: PVE_HOST environment variable not set${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ PVE_HOST set: $PVE_HOST${NC}"

    if [[ -z "$PVE_USER" ]]; then
        echo -e "${RED}❌ ERROR: PVE_USER environment variable not set${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ PVE_USER set: $PVE_USER${NC}"

    if [[ -z "$PVE_TOKEN_NAME" ]] || [[ -z "$PVE_TOKEN_VALUE" ]]; then
        echo -e "${RED}❌ ERROR: PVE_TOKEN_NAME or PVE_TOKEN_VALUE not set${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ API tokens configured${NC}"

    # Test connection
    echo -e "\nTesting connection to Proxmox..."
    if $GO_BIN list --vmid "$TEST_VM" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Connection successful${NC}"
    else
        echo -e "${RED}❌ ERROR: Cannot connect to Proxmox${NC}"
        exit 1
    fi
}

# Initialize results file
initialize_results() {
    cat > "$RESULTS_FILE" <<EOF
# Backup Operations Test Results

**Date:** $(date)
**Go Binary:** $GO_BIN
**Test VM:** $TEST_VM
**Test Storage:** $TEST_STORAGE
**Proxmox Host:** $PVE_HOST

---

## Test Results

| Status | Test ID | Description |
|--------|---------|-------------|
EOF
}

# Test Phase 1: Dry-Run Safety Tests (DRY-008 to DRY-013)
test_dry_run_operations() {
    print_section "Phase 1: Dry-Run Safety Tests"

    run_test "DRY-008" \
        "Dry-run create backup" \
        "$GO_BIN backup --vmid $TEST_VM --storage $TEST_STORAGE --mode snapshot --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-009" \
        "Dry-run delete backup by pattern" \
        "$GO_BIN delete-backups --vmid $TEST_VM --pattern '*test*' --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-010" \
        "Dry-run retention cleanup" \
        "$GO_BIN delete-backups --vmid $TEST_VM --keep-count 5 --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-011" \
        "Dry-run quick start all" \
        "$GO_BIN quick-start-all --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-012" \
        "Dry-run quick stop all" \
        "$GO_BIN quick-stop-all --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-013" \
        "Dry-run quick backup all" \
        "$GO_BIN quick-backup-all --storage $TEST_STORAGE --dry-run" \
        "[DRY-RUN]" \
        "output"

    run_test "DRY-015" \
        "Verify dry-run output format" \
        "$GO_BIN backup --vmid $TEST_VM --storage $TEST_STORAGE --dry-run" \
        "DRY-RUN SUMMARY" \
        "output"
}

# Test Phase 2: Storage Discovery (BKUP-001 to BKUP-003)
test_storage_discovery() {
    print_section "Phase 2: Storage Discovery"

    run_test "BKUP-001" \
        "List VM storages" \
        "$GO_BIN list-backups --vmid $TEST_VM" \
        "" \
        "no_error"

    skip_test "BKUP-002" \
        "List backup storages" \
        "Requires internal pkg/storage testing"

    skip_test "BKUP-003" \
        "Storage space validation" \
        "Requires internal pkg/storage testing"
}

# Test Phase 3: Create Backup (BKUP-004 to BKUP-007)
test_create_backup() {
    print_section "Phase 3: Create Backup Operations"

    echo -e "${YELLOW}⚠️  WARNING: The following tests will create actual backups${NC}"
    echo -e "${YELLOW}   This may take several minutes and consume storage space${NC}"
    read -p "Do you want to proceed with backup creation tests? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        skip_test "BKUP-004" "Create backup (snapshot mode)" "User skipped"
        skip_test "BKUP-005" "Create backup (suspend mode)" "User skipped"
        skip_test "BKUP-006" "Create backup (stop mode)" "User skipped"
        skip_test "BKUP-007" "Bulk backup creation" "User skipped"
        return
    fi

    run_test "BKUP-004" \
        "Create backup (snapshot mode)" \
        "$GO_BIN backup --vmid $TEST_VM --storage $TEST_STORAGE --mode snapshot" \
        "" \
        "no_error"

    skip_test "BKUP-005" \
        "Create backup (suspend mode)" \
        "Requires VM state manipulation"

    skip_test "BKUP-006" \
        "Create backup (stop mode)" \
        "Requires VM state manipulation"

    skip_test "BKUP-007" \
        "Bulk backup creation" \
        "Requires multiple test VMs"
}

# Test Phase 4: List & Restore Backups (BKUP-008 to BKUP-011)
test_list_restore_backup() {
    print_section "Phase 4: List & Restore Backup Operations"

    run_test "BKUP-008" \
        "List backups for VM" \
        "$GO_BIN list-backups --vmid $TEST_VM" \
        "" \
        "no_error"

    run_test "BKUP-009" \
        "List all backups in storage" \
        "$GO_BIN list-backups --vmid $TEST_VM --storage $TEST_STORAGE" \
        "" \
        "no_error"

    skip_test "BKUP-010" \
        "Restore from backup" \
        "Requires existing backup and would overwrite VM"

    skip_test "BKUP-011" \
        "Restore with protection check" \
        "Requires protected VM"
}

# Test Phase 5: Delete Backups (BKUP-012 to BKUP-016)
test_delete_backup() {
    print_section "Phase 5: Delete Backup Operations"

    echo -e "${YELLOW}⚠️  WARNING: The following tests will delete actual backups${NC}"
    echo -e "${YELLOW}   Make sure you have backups you can safely delete${NC}"
    read -p "Do you want to proceed with backup deletion tests? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        skip_test "BKUP-012" "Delete single backup" "User skipped"
        skip_test "BKUP-013" "Delete by pattern" "User skipped"
        skip_test "BKUP-014" "Delete with retention" "User skipped"
        skip_test "BKUP-015" "Delete by age" "User skipped"
        skip_test "BKUP-016" "Delete all backups" "User skipped"
        return
    fi

    skip_test "BKUP-012" \
        "Delete single backup" \
        "Requires specific backup volid"

    skip_test "BKUP-013" \
        "Delete by pattern" \
        "Requires test backups matching pattern"

    skip_test "BKUP-014" \
        "Delete with retention (keep-count)" \
        "Would delete real backups"

    skip_test "BKUP-015" \
        "Delete by age (max-age-days)" \
        "Would delete real backups"

    skip_test "BKUP-016" \
        "Delete all backups" \
        "Would delete all backups - too dangerous"
}

# Generate final report
generate_report() {
    print_header "Test Summary"

    local pass_rate=0
    if [[ $TOTAL -gt 0 ]]; then
        pass_rate=$((PASSED * 100 / TOTAL))
    fi

    echo "Total Tests:   $TOTAL"
    echo "Passed:        $PASSED"
    echo "Failed:        $FAILED"
    echo "Skipped:       $SKIPPED"
    echo "Pass Rate:     ${pass_rate}%"

    # Append summary to results file
    cat >> "$RESULTS_FILE" <<EOF

---

## Summary

- **Total Tests:** $TOTAL
- **Passed:** $PASSED
- **Failed:** $FAILED
- **Skipped:** $SKIPPED
- **Pass Rate:** ${pass_rate}%

### Failed Tests
EOF

    if [[ ${#FAILED_TESTS[@]} -eq 0 ]]; then
        echo "None" >> "$RESULTS_FILE"
    else
        for test in "${FAILED_TESTS[@]}"; do
            echo "- $test" >> "$RESULTS_FILE"
        done
    fi

    cat >> "$RESULTS_FILE" <<EOF

### Skipped Tests
EOF

    if [[ ${#SKIPPED_TESTS[@]} -eq 0 ]]; then
        echo "None" >> "$RESULTS_FILE"
    else
        for test in "${SKIPPED_TESTS[@]}"; do
            echo "- $test" >> "$RESULTS_FILE"
        done
    fi

    echo ""
    echo -e "${BLUE}Results saved to: $RESULTS_FILE${NC}"

    # Exit with appropriate code
    if [[ $FAILED -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Main execution
main() {
    print_header "Proxmox Snapshot Manager - Backup Operations Test Suite"
    echo "Testing backup operations (BKUP-001 to BKUP-016)"
    echo "Plus dry-run safety tests (DRY-008 to DRY-013)"
    echo ""

    check_prerequisites
    initialize_results

    test_dry_run_operations
    test_storage_discovery
    test_create_backup
    test_list_restore_backup
    test_delete_backup

    generate_report
}

# Run main function
main "$@"
