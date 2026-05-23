# Proxmox Admin CLI - Test Plan

**Version:** 1.2.0
**Last Updated:** 2026-02-16
**Target:** Complete functional testing of all CLI features

---

## Table of Contents

1. [Test Environment Setup](#1-test-environment-setup)
2. [Authentication Tests](#2-authentication-tests)
3. [VM Selection System Tests](#3-vm-selection-system-tests)
4. [Snapshot Operation Tests](#4-snapshot-operation-tests)
5. [Backup Operation Tests](#5-backup-operation-tests)
6. [VM Lifecycle Tests](#6-vm-lifecycle-tests)
7. [Cluster Operation Tests](#7-cluster-operation-tests)
8. [Node Operation Tests](#8-node-operation-tests)
9. [Container Operation Tests](#9-container-operation-tests)
10. [Bulk Operation Tests](#10-bulk-operation-tests)
11. [Error Handling Tests](#11-error-handling-tests)
12. [v1.2.0 Feature Tests](#12-v120-feature-tests)
13. [Test Execution Checklist](#13-test-execution-checklist)

---

## 1. Test Environment Setup

### Prerequisites

```bash
# 1. Build the binary
cd proxmox-admin-cli && make build

# 2. Set environment variables
export PVE_HOST=your-proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value

# 3. Verify API token permissions
# Run on Proxmox host:
pveum aclmod / -token 'username@pam!token-name' -role PVEVMAdmin

# 4. Create alias for testing
alias pve='./build/pve'
```

### Test VM Designation

| VM ID | Purpose | Notes |
|-------|---------|-------|
| 7303 | Primary test VM | Used for all destructive/modifying operations |
| 7201-7205 | Range test VMs | Used for range/wildcard selection tests (if available) |

### Pre-Test Checklist

- [ ] Binary compiled successfully
- [ ] Environment variables set correctly
- [ ] API token has PVEVMAdmin role
- [ ] Test VM 7303 exists and is accessible
- [ ] Storage with backup capability is available

---

## 2. Authentication Tests

### 2.1 Token Authentication (Recommended)

```bash
# Test 2.1.1: Valid token authentication
# Expected: Success - commands work normally
pve vm list

# Test 2.1.2: Missing PVE_HOST
# Expected: Error - host required
unset PVE_HOST && pve vm list
export PVE_HOST=your-proxmox-host.com

# Test 2.1.3: Missing PVE_USER
# Expected: Error - user required
unset PVE_USER && pve vm list
export PVE_USER=username@pam

# Test 2.1.4: Invalid token
# Expected: Error - authentication failed
export PVE_TOKEN_VALUE=invalid-token && pve vm list
export PVE_TOKEN_VALUE=your-valid-token
```

### 2.2 Password Authentication (Alternative)

```bash
# Test 2.2.1: Password authentication
# Expected: Success - commands work normally
unset PVE_TOKEN_NAME PVE_TOKEN_VALUE
export PVE_PASSWORD=your-password
pve vm list
```

---

## 3. VM Selection System Tests

### 3.1 Single VM Selection

```bash
# Test 3.1.1: Single VM by ID
# Expected: List details for VM 7303
pve vm details --vmid 7303

# Test 3.1.2: Non-existent VM ID
# Expected: Error - VM not found
pve vm details --vmid 99999
```

### 3.2 Multiple VM Selection

```bash
# Test 3.2.1: Comma-separated list (use available VMs)
# Expected: Operation on each specified VM
pve vm snapshot list --vmid 7303,7201,7202

# Test 3.2.2: Mixed valid and invalid VMs
# Expected: Error on invalid VM or partial success
pve vm snapshot list --vmid 7303,99999
```

### 3.3 Range Selection

```bash
# Test 3.3.1: VM ID range
# Expected: Operation on all VMs in range
pve vm snapshot list --vmid 7201-7205

# Test 3.3.2: Single VM range (edge case)
# Expected: Operation on single VM
pve vm snapshot list --vmid 7303-7303
```

### 3.4 Wildcard Selection

```bash
# Test 3.4.1: ID wildcard (prefix)
# Expected: Match all VMs starting with 72
pve vm list --vmid 72*

# Test 3.4.2: Name wildcard
# Expected: Match all VMs with names starting with 'web'
pve vm list --vmname web*

# Test 3.4.3: Wildcard with no matches
# Expected: Error - no VMs matched
pve vm list --vmid 99*
```

### 3.5 Keyword Selection

```bash
# Test 3.5.1: All VMs
# Expected: List all VMs
pve vm list --vmid all

# Test 3.5.2: Running VMs only
# Expected: List only running VMs
pve vm list --vmid running

# Test 3.5.3: Stopped VMs only
# Expected: List only stopped VMs
pve vm list --vmid stopped
```

### 3.6 Interactive Selection (v1.2.0+)

```bash
# Test 3.6.1: Interactive mode
# Expected: Checkbox-style VM selection UI
pve vm snapshot create --vmid i --prefix test

# Test 3.6.2: Interactive - Select all
# Input: all, then done
# Expected: All VMs selected
pve vm snapshot list --vmid i

# Test 3.6.3: Interactive - Clear selection
# Input: all, then none, then done
# Expected: No VMs selected (should error or prompt)
pve vm snapshot list --vmid i

# Test 3.6.4: Interactive - Toggle specific VMs
# Input: 1 3 5, then done
# Expected: Only VMs 1, 3, 5 selected
pve vm snapshot list --vmid i
```

---

## 4. Snapshot Operation Tests

> **IMPORTANT**: Use VMID 7303 for all snapshot tests unless otherwise noted.

### 4.1 Snapshot Create

```bash
# Test 4.1.1: Create snapshot with prefix (auto-timestamp)
# Expected: Snapshot created with name like "backup-20260216-143052"
pve vm snapshot create --vmid 7303 --prefix backup

# Test 4.1.2: Create snapshot with exact name
# Expected: Snapshot created with exact name "test-snapshot"
pve vm snapshot create --vmid 7303 --name test-snapshot

# Test 4.1.3: Create snapshot with VM state (RAM)
# Expected: Snapshot includes RAM, takes longer
pve vm snapshot create --vmid 7303 --prefix with-ram --vmstate

# Test 4.1.4: Create snapshot with prefix too long
# Expected: Error - prefix too long (max 25 chars)
pve vm snapshot create --vmid 7303 --prefix this-is-a-very-long-prefix-name

# Test 4.1.5: Create snapshot with invalid characters
# Expected: Invalid characters cleaned or error
pve vm snapshot create --vmid 7303 --name "test@invalid#chars"
```

### 4.2 Snapshot List

```bash
# Test 4.2.1: List all snapshots for VM
# Expected: Table of snapshots with names, dates, RAM status
pve vm snapshot list --vmid 7303

# Test 4.2.2: List snapshots for VM with no snapshots
# Expected: Empty list or "no snapshots found"
pve vm snapshot list --vmid <vm-with-no-snapshots>
```

### 4.3 Snapshot Rollback

```bash
# Prerequisite: Create a test snapshot first
pve vm snapshot create --vmid 7303 --name rollback-test -y

# Test 4.3.1: Rollback to snapshot
# Expected: VM rolled back to snapshot state
pve vm snapshot rollback --vmid 7303 --snapshot rollback-test -y

# Test 4.3.2: Rollback to non-existent snapshot
# Expected: Error - snapshot not found
pve vm snapshot rollback --vmid 7303 --snapshot nonexistent -y
```

### 4.4 Snapshot Delete

```bash
# Prerequisite: Create test snapshots
pve vm snapshot create --vmid 7303 --name delete-test-1 -y
pve vm snapshot create --vmid 7303 --name delete-test-2 -y
pve vm snapshot create --vmid 7303 --name delete-test-3 -y

# Test 4.4.1: Delete single snapshot
# Expected: Snapshot deleted successfully
pve vm snapshot delete --vmid 7303 --snapshot delete-test-1 -y

# Test 4.4.2: Delete multiple snapshots (v1.2.0+)
# Expected: Both snapshots deleted
pve vm snapshot delete --vmid 7303 --snapshot delete-test-2,delete-test-3 -y

# Test 4.4.3: Delete all snapshots with --all flag (v1.2.0+)
# Expected: All snapshots deleted
pve vm snapshot delete --vmid 7303 --all -y

# Test 4.4.4: Delete non-existent snapshot
# Expected: Error - snapshot not found
pve vm snapshot delete --vmid 7303 --snapshot nonexistent -y
```

---

## 5. Backup Operation Tests

### 5.1 Backup Create

```bash
# Test 5.1.1: Create backup with default mode (snapshot)
# Expected: Backup created in specified storage
pve vm backup create --vmid 7303 --storage local --mode snapshot -y

# Test 5.1.2: Create backup with suspend mode
# Expected: VM suspended, backup created, VM resumed
pve vm backup create --vmid 7303 --storage local --mode suspend -y

# Test 5.1.3: Create backup with stop mode
# Expected: VM stopped, backup created, VM stays stopped
pve vm backup create --vmid 7303 --storage local --mode stop -y

# Test 5.1.4: Create backup with compression
# Expected: Backup created with specified compression
pve vm backup create --vmid 7303 --storage local --compress zstd -y

# Test 5.1.5: Create backup to non-existent storage
# Expected: Error - storage not found
pve vm backup create --vmid 7303 --storage nonexistent -y
```

### 5.2 Backup List

```bash
# Test 5.2.1: List backups for specific VM
# Expected: List of backups for VM 7303
pve vm backup list --vmid 7303

# Test 5.2.2: List all backups in storage (v1.2.0+)
# Expected: All backups in storage, regardless of VM
pve vm backup list --all --storage local

# Test 5.2.3: List backups with --all but no --storage
# Expected: Error - storage required with --all
pve vm backup list --all
```

### 5.3 Backup Restore

```bash
# Prerequisite: Have a backup available
# Note: This will overwrite VM - use with caution!

# Test 5.3.1: Restore backup to same VM
# Expected: VM restored from backup
pve vm backup restore --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-*.vma.zst" --node pve1 -y

# Test 5.3.2: Restore with protected VM (v1.2.0+)
# Expected: Warning, option to disable protection
# First, enable protection on VM
# Then attempt restore
pve vm backup restore --vmid 7303 --backup-file "..." --node pve1
```

### 5.4 Backup Delete

```bash
# Prerequisite: Create test backups
pve vm backup create --vmid 7303 --storage local --prefix test-delete -y

# Test 5.4.1: Delete specific backup
# Expected: Backup deleted
pve vm backup delete --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-test-delete-*.vma.zst" -y

# Test 5.4.2: Delete by pattern
# Expected: All matching backups deleted
pve vm backup delete --vmid 7303 --pattern "*2024*" -y

# Test 5.4.3: Delete with retention (keep-count)
# Expected: Only newest N backups kept
pve vm backup delete --vmid 7303 --keep-count 3 -y

# Test 5.4.4: Delete by age
# Expected: Backups older than N days deleted
pve vm backup delete --vmid 7303 --max-age-days 30 -y
```

---

## 6. VM Lifecycle Tests

### 6.1 VM List

```bash
# Test 6.1.1: List all VMs
# Expected: Table of all VMs with ID, name, status
pve vm list

# Test 6.1.2: List VMs with verbose output
# Expected: Detailed VM information
pve vm list -v
```

### 6.2 VM Details

```bash
# Test 6.2.1: Show VM details
# Expected: Detailed VM configuration
pve vm details --vmid 7303
```

### 6.3 VM Start

```bash
# Prerequisite: VM must be stopped

# Test 6.3.1: Start single VM
# Expected: VM starts successfully
pve vm start --vmid 7303

# Test 6.3.2: Start already running VM
# Expected: Warning or error - VM already running
pve vm start --vmid 7303

# Test 6.3.3: Start multiple VMs
# Expected: All specified VMs start
pve vm start --vmid 7303,7201,7202
```

### 6.4 VM Stop

```bash
# Prerequisite: VM must be running

# Test 6.4.1: Force stop single VM
# Expected: VM stops immediately
pve vm stop --vmid 7303 -y

# Test 6.4.2: Stop already stopped VM
# Expected: Warning or error - VM already stopped
pve vm stop --vmid 7303 -y
```

### 6.5 VM Shutdown

```bash
# Prerequisite: VM must be running

# Test 6.5.1: Graceful shutdown
# Expected: VM shuts down gracefully
pve vm shutdown --vmid 7303 -y

# Test 6.5.2: Shutdown with timeout
# Expected: VM shuts down or force stopped after timeout
pve vm shutdown --vmid 7303 --timeout 60 -y
```

---

## 7. Cluster Operation Tests

### 7.1 Task List

```bash
# Test 7.1.1: List cluster tasks
# Expected: List of recent/running tasks
pve cluster task list

# Test 7.1.2: List tasks with verbose output
# Expected: Detailed task information
pve cluster task list -v
```

### 7.2 Storage List

```bash
# Test 7.2.1: List backup storages
# Expected: List of storages with backup capability
pve cluster storage list-backup
```

### 7.3 Network List

```bash
# Test 7.3.1: List network configuration
# Expected: Network configuration for specified node
pve cluster network list --node pve1

# Test 7.3.2: List network without --node
# Expected: Error - node required
pve cluster network list
```

---

## 8. Node Operation Tests

### 8.1 Node List

```bash
# Test 8.1.1: List all nodes
# Expected: Table of all cluster nodes
pve node list
```

### 8.2 Node Status

```bash
# Test 8.2.1: Show node status
# Expected: Node status information
pve node status --node pve1

# Test 8.2.2: Status of non-existent node
# Expected: Error - node not found
pve node status --node nonexistent
```

### 8.3 Node Resources

```bash
# Test 8.3.1: Show resource stats
# Expected: CPU, memory, disk statistics
pve node resource stats --node pve1
```

---

## 9. Container Operation Tests

### 9.1 Container List

```bash
# Test 9.1.1: List all containers
# Expected: Table of all LXC containers
pve container list

# Test 9.1.2: List with verbose output
# Expected: Detailed container information
pve container list -v
```

### 9.2 Container Start

```bash
# Prerequisite: Container must exist and be stopped

# Test 9.2.1: Start container
# Expected: Container starts
pve container start --vmid <container-id>
```

### 9.3 Container Stop

```bash
# Prerequisite: Container must be running

# Test 9.3.1: Stop container
# Expected: Container stops
pve container stop --vmid <container-id> -y
```

---

## 10. Bulk Operation Tests

### 10.1 Bulk Start

```bash
# Prerequisite: Some VMs must be stopped

# Test 10.1.1: Start all stopped VMs
# Expected: All stopped VMs start concurrently
pve vm bulk start

# Test 10.1.2: Bulk start with verbose output
# Expected: Detailed progress for each VM
pve vm bulk start -v
```

### 10.2 Bulk Stop

```bash
# Prerequisite: Some VMs must be running

# Test 10.2.1: Stop all running VMs
# Expected: All running VMs stop concurrently
pve vm bulk stop

# Test 10.2.2: Bulk stop with confirmation skip
# Expected: No confirmation prompt
pve vm bulk stop -y
```

### 10.3 Bulk Backup

```bash
# Test 10.3.1: Backup all VMs
# Expected: All VMs backed up concurrently
pve vm bulk backup --storage local

# Test 10.3.2: Bulk backup with mode
# Expected: All VMs backed up with specified mode
pve vm bulk backup --storage local --mode suspend
```

---

## 11. Error Handling Tests

### 11.1 API Errors

```bash
# Test 11.1.1: Invalid VMID
# Expected: Clear error message
pve vm details --vmid 999999

# Test 11.1.2: Invalid storage
# Expected: Clear error message
pve vm backup create --vmid 7303 --storage nonexistent

# Test 11.1.3: Insufficient permissions
# Expected: Permission error
# (Requires token with limited permissions)
```

### 11.2 Input Validation

```bash
# Test 11.2.1: Missing required flag
# Expected: Error - flag required
pve vm snapshot create --prefix backup

# Test 11.2.2: Invalid flag combination
# Expected: Error or warning
pve vm snapshot create --vmid 7303 --prefix backup --name exact-name

# Test 11.2.3: Invalid range format
# Expected: Error parsing range
pve vm list --vmid 7201-abc
```

### 11.3 Network Errors

```bash
# Test 11.3.1: Connection timeout
# Expected: Timeout error with retry suggestion
# (Can simulate with firewall or invalid host)
```

### 11.4 Confirmation Prompts

```bash
# Test 11.4.1: Destructive operation without -y
# Expected: Confirmation prompt
pve vm snapshot delete --vmid 7303 --snapshot test

# Test 11.4.2: Batch mode
# Expected: No prompts, continue on errors
pve vm stop --vmid 7303,7201,99999 --batch
```

---

## 12. v1.2.0 Feature Tests

### 12.1 Checkbox-Style Interactive Selection

```bash
# Test 12.1.1: Interactive selection with 'i'
# Expected: Checkbox UI appears
pve vm snapshot create --vmid i --prefix test

# Test 12.1.2: Interactive commands: all, none, done
# Input sequence: all -> none -> 1 3 5 -> done
# Expected: Only VMs 1, 3, 5 selected
pve vm snapshot list --vmid interactive

# Test 12.1.3: Interactive help display
# Expected: Help text shows available commands
pve vm snapshot list --vmid i
# (Help should be displayed automatically)
```

### 12.2 Storage-Wide Backup Listing

```bash
# Test 12.2.1: List all backups in storage
# Expected: All backups from all VMs in storage
pve vm backup list --all --storage local

# Test 12.2.2: Verify VMID extraction
# Expected: Each backup shows associated VMID
pve vm backup list --all --storage local -v

# Test 12.2.3: Error without storage
# Expected: Error - storage required
pve vm backup list --all
```

### 12.3 VM Protection Disable

```bash
# Prerequisite: Enable protection on test VM

# Test 12.3.1: Restore protected VM
# Expected: Warning displayed, option to disable
pve vm backup restore --vmid 7303 --backup-file "..." --node pve1

# Test 12.3.2: Cancel restore on protected VM
# Expected: Operation cancelled
# (Select 'Cancel' when prompted)

# Test 12.3.3: Proceed with protection disable
# Expected: Protection disabled, restore proceeds
# (Select 'Disable' when prompted)
```

### 12.4 Multiple Snapshot Delete

```bash
# Prerequisite: Create multiple test snapshots
pve vm snapshot create --vmid 7303 --name multi-test-1 -y
pve vm snapshot create --vmid 7303 --name multi-test-2 -y
pve vm snapshot create --vmid 7303 --name multi-test-3 -y

# Test 12.4.1: Delete multiple with comma-separated list
# Expected: All listed snapshots deleted
pve vm snapshot delete --vmid 7303 --snapshot multi-test-1,multi-test-2,multi-test-3 -y

# Test 12.4.2: Delete all with --all flag
# Expected: All snapshots deleted
pve vm snapshot delete --vmid 7303 --all -y

# Test 12.4.3: Mixed valid and invalid snapshot names
# Expected: Error on invalid or partial success
pve vm snapshot delete --vmid 7303 --snapshot existing,nonexistent -y
```

---

## 13. Test Execution Checklist

### Pre-Execution

- [ ] Binary compiled (`make build`)
- [ ] Environment variables configured
- [ ] Test VM 7303 exists and accessible
- [ ] Storage available for backups
- [ ] API token has correct permissions

### Test Categories

| Category | Tests | Status | Notes |
|----------|-------|--------|-------|
| Authentication | 5 | [ ] | |
| VM Selection | 16 | [ ] | |
| Snapshot Create | 5 | [ ] | |
| Snapshot List | 2 | [ ] | |
| Snapshot Rollback | 2 | [ ] | |
| Snapshot Delete | 4 | [ ] | |
| Backup Create | 5 | [ ] | |
| Backup List | 3 | [ ] | |
| Backup Restore | 2 | [ ] | |
| Backup Delete | 4 | [ ] | |
| VM Lifecycle | 11 | [ ] | |
| Cluster Ops | 5 | [ ] | |
| Node Ops | 5 | [ ] | |
| Container Ops | 4 | [ ] | |
| Bulk Ops | 5 | [ ] | |
| Error Handling | 9 | [ ] | |
| v1.2.0 Features | 11 | [ ] | |
| **Total** | **98** | [ ] | |

### Post-Execution Cleanup

```bash
# Remove test snapshots
pve vm snapshot delete --vmid 7303 --all -y

# Remove test backups (optional)
# pve vm backup delete --vmid 7303 --pattern "*test*" -y

# Verify VM state restored
pve vm status --vmid 7303
```

### Test Report Template

```markdown
## Test Report - [Date]

**Tester:** [Name]
**Version:** 1.2.0
**Environment:** [Proxmox Version, OS]

### Summary
- Total Tests: 98
- Passed: X
- Failed: X
- Skipped: X

### Failed Tests
| Test ID | Description | Error | Resolution |
|---------|-------------|-------|------------|
| 4.1.4 | Prefix too long | ... | ... |

### Notes
[Any additional observations or recommendations]
```

---

## Appendix A: Automated Test Script

For quick validation, use this automated test script:

```bash
#!/bin/bash
# quick-test.sh - Basic smoke test for pve CLI

set -e

echo "=== Proxmox Admin CLI Quick Test ==="

# Check binary
if [ ! -f "./build/pve" ]; then
    echo "ERROR: Binary not found. Run 'make build' first."
    exit 1
fi

alias pve='./build/pve'

# Authentication test
echo "1. Testing authentication..."
pve vm list > /dev/null && echo "   PASS: Authentication" || echo "   FAIL: Authentication"

# VM list test
echo "2. Testing VM list..."
pve vm list > /dev/null && echo "   PASS: VM list" || echo "   FAIL: VM list"

# Node list test
echo "3. Testing Node list..."
pve node list > /dev/null && echo "   PASS: Node list" || echo "   FAIL: Node list"

# Snapshot list test (using 7303)
echo "4. Testing Snapshot list..."
pve vm snapshot list --vmid 7303 > /dev/null && echo "   PASS: Snapshot list" || echo "   FAIL: Snapshot list"

# Backup storage test
echo "5. Testing Backup storage list..."
pve cluster storage list-backup > /dev/null && echo "   PASS: Backup storage" || echo "   FAIL: Backup storage"

echo ""
echo "=== Quick Test Complete ==="
```

---

## Appendix B: Test Data Setup

### Create Test Snapshots

```bash
# Create multiple test snapshots for delete tests
for i in {1..5}; do
    pve vm snapshot create --vmid 7303 --name "test-snap-$i" -y
done
```

### Create Test Backups

```bash
# Create test backups for delete tests
for i in {1..3}; do
    pve vm backup create --vmid 7303 --storage local --prefix "test-$i" -y
done
```

### Cleanup Test Data

```bash
# Remove all test snapshots
pve vm snapshot delete --vmid 7303 --all -y

# Remove all test backups (be careful!)
pve vm backup delete --vmid 7303 --pattern "*test*" -y
```
