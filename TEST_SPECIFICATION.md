# Proxmox Snapshot Manager - Test Specification

**Version:** 1.0
**Date:** 2025-10-09
**Purpose:** Comprehensive test specification for validating feature parity between Python and Go implementations

---

## Test Environment Requirements

### Prerequisites
- Proxmox VE 6.0+ cluster (minimum 1 node)
- Test VMs with IDs: 7300-7309 (10 VMs for testing)
- API token with PVEVMAdmin role
- Network connectivity to Proxmox host
- Both Python and Go implementations installed

### Environment Variables
```bash
export PVE_HOST=proxmox-test-host.com
export PVE_USER=test-user@pam
export PVE_TOKEN_NAME=test-token
export PVE_TOKEN_VALUE=test-token-value
```

### Test VM Setup
```bash
# Recommended test VM distribution:
# - 7300-7302: Running VMs
# - 7303-7305: Stopped VMs
# - 7306-7308: VMs with existing snapshots
# - 7309: Protected VM for protection testing
```

---

## Test Categories

## 1. Core API & Authentication Tests

### 1.1 Token Authentication
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| API-001 | Connect with valid token | `python3 main.py list --vmid 7300` | `./proxmox-snapshot-manager list --vmid 7300` | Success, snapshot list displayed | ⬜ |
| API-002 | Connect with invalid token | Set wrong TOKEN_VALUE | Set wrong TOKEN_VALUE | Error: Authentication failed | ⬜ |
| API-003 | Connect with missing token | Unset TOKEN_VALUE | Unset TOKEN_VALUE | Error: Missing credentials | ⬜ |
| API-004 | SSL verification enabled | Set verify_ssl=true | Set VerifySSL=true | Success or cert error | ⬜ |
| API-005 | SSL verification disabled | Set verify_ssl=false | Set VerifySSL=false | Success (insecure) | ⬜ |

### 1.2 HTTP Operations
| Test ID | Test Case | Validation Method | Expected Result | Status |
|---------|-----------|------------------|-----------------|--------|
| API-006 | GET request (list VMs) | Monitor network traffic | Valid GET to /api2/json/cluster/resources | ⬜ |
| API-007 | POST request (create snapshot) | Monitor network traffic | Valid POST to /api2/json/nodes/.../qemu/.../snapshot | ⬜ |
| API-008 | DELETE request (delete snapshot) | Monitor network traffic | Valid DELETE to snapshot endpoint | ⬜ |
| API-009 | Handle timeout | Set short timeout | Error: Request timeout | ⬜ |
| API-010 | Handle connection error | Stop Proxmox service | Error: Connection refused | ⬜ |

---

## 2. Node & VM Discovery Tests

### 2.1 Node Operations
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| NODE-001 | List all nodes | Internal API call | Internal API call | All cluster nodes listed | ⬜ |
| NODE-002 | Find VM node for existing VM | Find node for VM 7300 | Find node for VM 7300 | Correct node returned | ⬜ |
| NODE-003 | Find VM node for non-existent VM | Find node for VM 9999 | Find node for VM 9999 | Error: VM not found | ⬜ |

### 2.2 VM Discovery
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| VM-001 | Get all VMs | `get_all_vms()` | `GetAllVMs()` | All VMs across cluster returned | ⬜ |
| VM-002 | Get VM status | `get_vm_status_detailed(7300)` | `GetVMStatus("7300")` | VM status with details | ⬜ |
| VM-003 | Get running VMs only | Filter by status | Filter by status | Only running VMs | ⬜ |
| VM-004 | Get stopped VMs only | Filter by status | Filter by status | Only stopped VMs | ⬜ |
| VM-005 | VM exists check | N/A | `VMExists("7300")` | True for existing VM | ⬜ |
| VM-006 | VM exists check (invalid) | N/A | `VMExists("9999")` | False for non-existent VM | ⬜ |
| VM-007 | Validate VM ID format | N/A | `ValidateVMID("7300")` | Success | ⬜ |
| VM-008 | Validate invalid VM ID | N/A | `ValidateVMID("abc")` | Error: Invalid format | ⬜ |

---

## 3. VM Selection Pattern Tests

### 3.1 Basic Selection Patterns
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SEL-001 | Single VM by ID | `--vmid 7300` | `--vmid 7300` | VM 7300 selected | ⬜ |
| SEL-002 | Single VM by name | `--vmname test-vm-01` | `--vmname test-vm-01` | Correct VM selected | ⬜ |
| SEL-003 | Range selection | `--vmid 7300-7302` | `--vmid 7300-7302` | VMs 7300, 7301, 7302 selected | ⬜ |
| SEL-004 | Comma-separated list | `--vmid 7300,7302,7304` | `--vmid 7300,7302,7304` | VMs 7300, 7302, 7304 selected | ⬜ |
| SEL-005 | Wildcard pattern (prefix) | `--vmid 730*` | `--vmid 730*` | VMs 7300-7309 selected | ⬜ |
| SEL-006 | Wildcard pattern (suffix) | `--vmid *00` | `--vmid *00` | VM 7300 selected | ⬜ |
| SEL-007 | Mixed comma and range | `--vmid 7300,7302-7304` | `--vmid 7300,7302-7304` | VMs 7300, 7302, 7303, 7304 | ⬜ |
| SEL-008 | Mixed IDs and names | `--vmid 7300 --vmname test-vm` | `--vmid 7300 --vmname test-vm` | Both VMs selected | ⬜ |

### 3.2 Keyword Selection
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SEL-009 | Select all VMs | `--vmid all` | `--vmid all` | All VMs selected | ⬜ |
| SEL-010 | Select running VMs | `--vmid running` | `--vmid running` | Only running VMs (7300-7302) | ⬜ |
| SEL-011 | Select stopped VMs | `--vmid stopped` | `--vmid stopped` | Only stopped VMs (7303-7305) | ⬜ |

### 3.3 Interactive Selection
| Test ID | Test Case | Manual Steps | Expected Result | Status |
|---------|-----------|--------------|-----------------|--------|
| SEL-012 | Interactive checkbox selection | Run without --vmid, select with space | Selected VMs confirmed | ⬜ |
| SEL-013 | Interactive numeric selection | Enter "1,3,5" | VMs at positions 1, 3, 5 selected | ⬜ |
| SEL-014 | Interactive range selection | Enter "1-5" | First 5 VMs selected | ⬜ |
| SEL-015 | Interactive cancel | Press 'q' or Ctrl+C | Operation cancelled | ⬜ |

### 3.4 Error Handling
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SEL-016 | Invalid VM ID | `--vmid 9999` | `--vmid 9999` | Error: VM not found | ⬜ |
| SEL-017 | Invalid range format | `--vmid 7300-` | `--vmid 7300-` | Error: Invalid range | ⬜ |
| SEL-018 | Invalid wildcard | `--vmid **` | `--vmid **` | Error or no matches | ⬜ |
| SEL-019 | Empty selection | `--vmid ""` | `--vmid ""` | Error: No VMs specified | ⬜ |

---

## 4. Snapshot Operations Tests

### 4.1 Create Snapshot
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SNAP-001 | Create with prefix | `create --vmid 7300 --prefix test` | `create --vmid 7300 --prefix test` | Snapshot: test-7300-TIMESTAMP | ⬜ |
| SNAP-002 | Create with exact name | `create --vmid 7300 --name backup-01` | `create --vmid 7300 --name backup-01` | Snapshot: backup-01 | ⬜ |
| SNAP-003 | Create with vmstate | `create --vmid 7300 --prefix backup --vmstate` | `create --vmid 7300 --prefix backup --vmstate` | Snapshot with RAM saved | ⬜ |
| SNAP-004 | Create without vmstate | `create --vmid 7300 --prefix backup` | `create --vmid 7300 --prefix backup` | Snapshot without RAM | ⬜ |
| SNAP-005 | Auto-detect vmstate keyword | `create --vmid 7300 --prefix vmstate-backup` | `create --vmid 7300 --prefix vmstate-backup` | RAM automatically included | ⬜ |
| SNAP-006 | Prefix length limit | `create --vmid 7300 --prefix [26+ chars]` | `create --vmid 7300 --prefix [26+ chars]` | Error or truncation | ⬜ |
| SNAP-007 | Invalid characters in name | `create --vmid 7300 --name test/snap` | `create --vmid 7300 --name test/snap` | Characters sanitized | ⬜ |
| SNAP-008 | Duplicate snapshot name | Create same name twice | Create same name twice | Error: Snapshot exists | ⬜ |

### 4.2 List Snapshots
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SNAP-009 | List snapshots for VM | `list --vmid 7306` | `list --vmid 7306` | All snapshots displayed | ⬜ |
| SNAP-010 | List for VM with no snapshots | `list --vmid 7300` | `list --vmid 7300` | "No snapshots" message | ⬜ |
| SNAP-011 | List with snapshot details | `list --vmid 7306` | `list --vmid 7306` | Name, date, description shown | ⬜ |

### 4.3 Rollback Snapshot
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SNAP-012 | Rollback to snapshot | `rollback --vmid 7306 --snapshot test-snap` | `rollback --vmid 7306 --snapshot test-snap` | VM reverted to snapshot | ⬜ |
| SNAP-013 | Rollback running VM | Rollback VM 7300 (running) | Rollback VM 7300 (running) | Stops VM, rollback, restart | ⬜ |
| SNAP-014 | Rollback stopped VM | Rollback VM 7303 (stopped) | Rollback VM 7303 (stopped) | Rollback, stays stopped | ⬜ |
| SNAP-015 | Rollback non-existent snapshot | `rollback --vmid 7306 --snapshot fake` | `rollback --vmid 7306 --snapshot fake` | Error: Snapshot not found | ⬜ |
| SNAP-016 | Rollback with confirmation | Without --yes flag | Without --yes flag | Prompt for confirmation | ⬜ |
| SNAP-017 | Rollback batch mode | `rollback --vmid 7306 --snapshot test --batch -y` | `rollback --vmid 7306 --snapshot test --batch -y` | No confirmation prompt | ⬜ |

### 4.4 Delete Snapshot
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SNAP-018 | Delete single snapshot | `delete --vmid 7306 --snapshot test-snap` | `delete --vmid 7306 --snapshot test-snap` | Snapshot deleted | ⬜ |
| SNAP-019 | Delete with confirmation | Without --yes | Without --yes | Prompt for confirmation | ⬜ |
| SNAP-020 | Delete batch mode | `delete --vmid 7306 --snapshot test --yes` | `delete --vmid 7306 --snapshot test --yes` | No confirmation | ⬜ |
| SNAP-021 | Delete all snapshots | `delete --vmid 7306 --all` | `delete --vmid 7306 --all` | All snapshots deleted | ⬜ |
| SNAP-022 | Delete all with confirmation | `delete --vmid 7306 --all` | `delete --vmid 7306 --all` | Strong confirmation required | ⬜ |
| SNAP-023 | Delete non-existent snapshot | `delete --vmid 7306 --snapshot fake` | `delete --vmid 7306 --snapshot fake` | Error: Not found | ⬜ |

### 4.5 Snapshot Comparison
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| SNAP-024 | Compare two snapshots | N/A (not in Python) | `CompareSnapshots()` | Differences shown | ⬜ |

---

## 5. VM Lifecycle Tests

### 5.1 Start VM
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| VM-L-001 | Start stopped VM | `start --vmid 7303` | `start --vmid 7303` | VM started successfully | ⬜ |
| VM-L-002 | Start already running VM | `start --vmid 7300` | `start --vmid 7300` | Error or "already running" | ⬜ |
| VM-L-003 | Start with task monitoring | `start --vmid 7303` | `start --vmid 7303` | Progress shown, task completes | ⬜ |
| VM-L-004 | Start batch mode | `start --vmid 7303 --batch -y` | `start --vmid 7303 --batch -y` | No confirmation | ⬜ |

### 5.2 Stop VM
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| VM-L-005 | Stop running VM | `stop --vmid 7300` | `stop --vmid 7300` | VM stopped (forced) | ⬜ |
| VM-L-006 | Stop already stopped VM | `stop --vmid 7303` | `stop --vmid 7303` | Error or "already stopped" | ⬜ |
| VM-L-007 | Stop with confirmation | `stop --vmid 7300` | `stop --vmid 7300` | Prompt for confirmation | ⬜ |
| VM-L-008 | Stop batch mode | `stop --vmid 7300 --batch -y` | `stop --vmid 7300 --batch -y` | No confirmation | ⬜ |

### 5.3 Shutdown VM
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| VM-L-009 | Graceful shutdown | `shutdown --vmid 7300` | `shutdown --vmid 7300` | VM gracefully shutdown | ⬜ |
| VM-L-010 | Shutdown with timeout | `shutdown --vmid 7300` | `shutdown --vmid 7300` | Wait for guest agent | ⬜ |
| VM-L-011 | Shutdown without guest agent | `shutdown --vmid 7300` | `shutdown --vmid 7300` | Timeout or force stop | ⬜ |

### 5.4 Reset VM
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| VM-L-012 | Reset running VM | N/A | `ResetVM("7300")` | VM reset | ❌ Python |

---

## 6. Bulk Operations Tests

### 6.1 Bulk Start
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BULK-001 | Bulk start 3 VMs | `start --vmid 7303,7304,7305` | `start --vmid 7303,7304,7305` | All 3 VMs started | ⬜ |
| BULK-002 | Concurrent start | `start --vmid 7303-7308` | `start --vmid 7303-7308` | Concurrent execution | ⬜ |
| BULK-003 | Progress tracking | `start --vmid 7303-7308` | `start --vmid 7303-7308` | Progress shown (X/Y) | ⬜ |
| BULK-004 | Partial failure handling | Start mix of valid/invalid | Start mix of valid/invalid | Valid complete, errors reported | ⬜ |
| BULK-005 | Summary report | `start --vmid 7303-7308` | `start --vmid 7303-7308` | Success/failure summary | ⬜ |

### 6.2 Bulk Stop
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BULK-006 | Bulk stop 3 VMs | `stop --vmid 7300,7301,7302 -y` | `stop --vmid 7300,7301,7302 -y` | All 3 VMs stopped | ⬜ |
| BULK-007 | Concurrent stop | `stop --vmid 7300-7305 -y` | `stop --vmid 7300-7305 -y` | Concurrent execution | ⬜ |

### 6.3 Bulk Snapshots
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BULK-008 | Bulk create snapshots | `create --vmid 7300-7302 --prefix bulk-test` | `create --vmid 7300-7302 --prefix bulk-test` | 3 snapshots created | ⬜ |
| BULK-009 | Bulk delete snapshots | `delete --vmid 7306-7308 --snapshot test -y` | `delete --vmid 7306-7308 --snapshot test -y` | Snapshots deleted | ⬜ |
| BULK-010 | Bulk rollback | `rollback --vmid 7306-7308 --snapshot test -y` | `rollback --vmid 7306-7308 --snapshot test -y` | All VMs rolled back | ⬜ |
| BULK-011 | Concurrency limit | Check max workers | Check max workers | Respects configured limit | ⬜ |
| BULK-012 | Cancel bulk operation | Ctrl+C during operation | Ctrl+C during operation | Graceful cancellation | ⬜ |

---

## 7. Dry-Run Safety Tests (Go-Specific Feature)

### 7.1 Snapshot Operations Dry-Run
| Test ID | Test Case | Go Command | Expected Result | Status |
|---------|-----------|------------|-----------------|--------|
| DRY-001 | Dry-run create snapshot | `create --vmid 7300 --prefix test --dry-run` | Shows what would be created, no actual snapshot | ⬜ |
| DRY-002 | Dry-run rollback | `rollback --vmid 7306 --snapshot test-snap --dry-run` | Shows rollback plan, no actual rollback | ⬜ |
| DRY-003 | Dry-run delete snapshot | `delete --vmid 7306 --snapshot test-snap --dry-run` | Shows deletion plan, no actual deletion | ⬜ |
| DRY-004 | Dry-run bulk create | `create --vmid 7300-7302 --prefix bulk --dry-run` | Shows all 3 operations, none executed | ⬜ |

### 7.2 VM Operations Dry-Run
| Test ID | Test Case | Go Command | Expected Result | Status |
|---------|-----------|------------|-----------------|--------|
| DRY-005 | Dry-run start VMs | `start --vmid 7303,7304,7305 --dry-run` | Shows VMs to start, none started | ⬜ |
| DRY-006 | Dry-run stop VMs | `stop --vmid 7300,7301,7302 --dry-run` | Shows VMs to stop, none stopped | ⬜ |
| DRY-007 | Dry-run shutdown VMs | `shutdown --vmid 7300,7301,7302 --dry-run` | Shows graceful shutdown plan, none executed | ⬜ |

### 7.3 Backup Operations Dry-Run
| Test ID | Test Case | Go Command | Expected Result | Status |
|---------|-----------|------------|-----------------|--------|
| DRY-008 | Dry-run create backup | `backup --vmid 7300 --storage local-zfs --mode snapshot --dry-run` | Shows backup plan, no actual backup | ⬜ |
| DRY-009 | Dry-run delete backup | `delete-backups --vmid 7300 --pattern "*2024*" --dry-run` | Shows backups to delete, none deleted | ⬜ |
| DRY-010 | Dry-run retention cleanup | `delete-backups --vmid 7300 --keep-count 5 --dry-run` | Shows cleanup plan, no actual deletion | ⬜ |

### 7.4 Quick Operations Dry-Run
| Test ID | Test Case | Go Command | Expected Result | Status |
|---------|-----------|------------|-----------------|--------|
| DRY-011 | Dry-run quick start all | `quick-start-all --dry-run` | Shows all stopped VMs, none started | ⬜ |
| DRY-012 | Dry-run quick stop all | `quick-stop-all --dry-run` | Shows all running VMs, none stopped | ⬜ |
| DRY-013 | Dry-run quick backup all | `quick-backup-all --storage local-zfs --dry-run` | Shows all VMs to backup, none backed up | ⬜ |

### 7.5 Interactive Mode Dry-Run
| Test ID | Test Case | Manual Steps | Expected Result | Status |
|---------|-----------|--------------|-----------------|--------|
| DRY-014 | Interactive with global dry-run | `./proxmox-snapshot-manager --dry-run` then select operation | All operations show dry-run mode | ⬜ |
| DRY-015 | Dry-run output format | Any operation with `--dry-run` | Clear [DRY-RUN] prefix and summary | ⬜ |
| DRY-016 | Dry-run no API calls | Monitor network during `--dry-run` operation | No API calls to Proxmox | ⬜ |

---

## 8. Backup Operations Tests

### 8.1 Storage Discovery
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BKUP-001 | List VM storages | Menu → Storage list | `list-backups --vmid <id>` | VM storages shown | ⬜ |
| BKUP-002 | List backup storages | Menu → Backup storage | Storage operations in pkg/storage | Backup storages shown | ⬜ |
| BKUP-003 | Storage space check | Before backup | `ValidateStorage()` in pkg/storage | Space validation | ⬜ |

### 8.2 Create Backup
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BKUP-004 | Create backup (snapshot mode) | `backup --vmid 7300 --storage local-zfs --mode snapshot` | `backup --vmid 7300 --storage local-zfs --mode snapshot` | Backup created | ⬜ |
| BKUP-005 | Create backup (suspend mode) | `backup --vmid 7300 --storage local-zfs --mode suspend` | `backup --vmid 7300 --storage local-zfs --mode suspend` | VM suspended, backed up | ⬜ |
| BKUP-006 | Create backup (stop mode) | `backup --vmid 7303 --storage local-zfs --mode stop` | `backup --vmid 7303 --storage local-zfs --mode stop` | VM stopped, backed up | ⬜ |
| BKUP-007 | Bulk backup creation | `backup --vmid 7300-7302 --storage local-zfs` | `backup --vmid 7300-7302 --storage local-zfs` | Multiple backups created | ⬜ |

### 8.3 List & Restore Backups
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BKUP-008 | List backups for VM | `list-backups --vmid 7300` | `list-backups --vmid 7300` | All backups shown with volid | ⬜ |
| BKUP-009 | List all backups in storage | Menu option | `list-backups --vmid 7300 --storage <storage>` | All backups in storage | ⬜ |
| BKUP-010 | Restore from backup | `restore --vmid 7300 --backup-file <volid>` | `restore --vmid 7300 --backup-file <volid> --node <node>` | VM restored | ⬜ |
| BKUP-011 | Restore with protection check | Restore protected VM | `restore` (protection check in pkg/protection) | Protection warning shown | ⬜ |

### 8.4 Delete Backups
| Test ID | Test Case | Python Command | Go Command | Expected Result | Status |
|---------|-----------|----------------|------------|-----------------|--------|
| BKUP-012 | Delete single backup | `delete-backups --vmid 7300 --backup-file <volid>` | `delete-backups --vmid 7300 --backup-file <volid> --yes` | Backup deleted | ⬜ |
| BKUP-013 | Delete by pattern | `delete-backups --vmid 7300 --pattern "*2024*"` | `delete-backups --vmid 7300 --pattern "*2024*" --yes` | Matching backups deleted | ⬜ |
| BKUP-014 | Delete with retention | `delete-backups --vmid 7300 --keep-count 5` | `delete-backups --vmid 7300 --keep-count 5 --yes` | Oldest backups deleted | ⬜ |
| BKUP-015 | Delete by age | `delete-backups --vmid 7300 --max-age-days 30` | `delete-backups --vmid 7300 --max-age-days 30 --yes` | Old backups deleted | ⬜ |
| BKUP-016 | Delete all backups | Interactive menu | `delete-backups --vmid 7300 --pattern "*" --yes` | All backups deleted with confirmation | ⬜ |

---

## 9. Interactive Menu Tests

### 9.1 Main Menu Navigation
| Test ID | Test Case | Manual Steps | Expected Result | Status |
|---------|-----------|--------------|-----------------|--------|
| MENU-001 | Start interactive mode (Python) | `python3 main.py` | Main menu displayed | ⬜ |
| MENU-002 | Start interactive mode (Go) | `./proxmox-snapshot-manager` | Main menu displayed | ⬜ |
| MENU-003 | Navigate snapshot operations | Select snapshot menu | Snapshot options shown | ⬜ |
| MENU-004 | Navigate VM operations | Select VM menu | VM options shown | ⬜ |
| MENU-005 | Navigate bulk operations | Select bulk menu | Bulk options shown | ❌ Go |
| MENU-006 | Exit menu | Select exit/quit | Program exits cleanly | ⬜ |

### 9.2 Interactive Operations
| Test ID | Test Case | Manual Steps | Expected Result | Status |
|---------|-----------|--------------|-----------------|--------|
| MENU-007 | Interactive create snapshot | Menu → Create → Select VM → Enter prefix | Snapshot created | ⬜ |
| MENU-008 | Interactive list snapshots | Menu → List → Select VM | Snapshots displayed | ⬜ |
| MENU-009 | Interactive rollback | Menu → Rollback → Select VM → Select snapshot | Rollback executed | ⬜ |
| MENU-010 | Interactive delete | Menu → Delete → Select VM → Select snapshot | Snapshot deleted | ⬜ |
| MENU-011 | Interactive start VM | Menu → Start → Select VM | VM started | ⬜ |
| MENU-012 | Interactive stop VM | Menu → Stop → Select VM | VM stopped | ⬜ |

### 9.3 Bulk Interactive Operations
| Test ID | Test Case | Manual Steps | Expected Result | Status |
|---------|-----------|--------------|-----------------|--------|
| MENU-013 | Bulk start VMs | Menu → Bulk → Start → Select VMs | Multiple VMs started | ❌ Go |
| MENU-014 | Bulk stop VMs | Menu → Bulk → Stop → Select VMs | Multiple VMs stopped | ❌ Go |
| MENU-015 | Bulk create snapshots | Menu → Bulk → Create Snaps → Select VMs | Multiple snapshots created | ❌ Go |
| MENU-016 | Bulk delete snapshots | Menu → Bulk → Delete Snaps → Select VMs | Multiple snapshots deleted | ❌ Go |
| MENU-017 | Quick start all | Menu → Quick → Start All | All stopped VMs started | ⬜ |
| MENU-018 | Quick stop all | Menu → Quick → Stop All | All running VMs stopped | ⬜ |
| MENU-019 | Quick backup all | Menu → Quick → Backup All → Select storage | All VMs backed up | ⬜ |

---

## 10. Error Handling & Edge Cases

### 10.1 Network Errors
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| ERR-001 | Connection timeout | Set low timeout, slow network | Error: Timeout | ⬜ |
| ERR-002 | Connection refused | Stop Proxmox | Error: Connection refused | ⬜ |
| ERR-003 | Network interruption | Disconnect during operation | Error: Connection lost | ⬜ |
| ERR-004 | Task monitoring timeout | Long-running task | Timeout or retry | ⬜ |

### 10.2 Permission Errors
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| ERR-005 | Insufficient permissions | Use token without PVEVMAdmin | Error: Permission denied | ⬜ |
| ERR-006 | Token expired | Use expired token | Error: Authentication failed | ⬜ |
| ERR-007 | Protected VM operation | Delete snapshot on protected VM | Error or warning | ⬜ |

### 10.3 Resource Errors
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| ERR-008 | Insufficient storage space | Backup to full storage | Error: No space | ⬜ |
| ERR-009 | VM locked | Perform operation on locked VM | Error: VM locked | ⬜ |
| ERR-010 | Concurrent operation conflict | Run two operations simultaneously | Error or queue | ⬜ |
| ERR-011 | Snapshot limit reached | Create too many snapshots | Error: Limit reached | ⬜ |

### 10.4 Input Validation
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| ERR-012 | Invalid VM ID format | Use non-numeric ID | Error: Invalid format | ⬜ |
| ERR-013 | Invalid snapshot name | Use special characters | Characters sanitized or error | ⬜ |
| ERR-014 | Empty snapshot name | Provide empty name | Error: Name required | ⬜ |
| ERR-015 | Too long prefix | Use 30+ character prefix | Error or truncation | ⬜ |

---

## 11. Performance & Concurrency Tests

### 11.1 Concurrency
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| PERF-001 | Max concurrent snapshots | Create 10 snapshots simultaneously | Respects max workers limit | ⬜ |
| PERF-002 | Max concurrent VM ops | Start 10 VMs simultaneously | Respects max workers limit | ⬜ |
| PERF-003 | Graceful degradation | Exceed worker limit | Operations queued properly | ⬜ |
| PERF-004 | Thread safety (Python) | Concurrent access to shared state | No race conditions | ⬜ |
| PERF-005 | Goroutine safety (Go) | Concurrent access to shared state | No race conditions | ⬜ |

### 11.2 Performance Benchmarks
| Test ID | Test Case | Python Baseline | Go Target | Actual Go | Status |
|---------|-----------|-----------------|-----------|-----------|--------|
| PERF-006 | Create 10 snapshots | 45s | <9s (5x faster) | - | ⬜ |
| PERF-007 | Delete 20 snapshots | 52s | <10s (5x faster) | - | ⬜ |
| PERF-008 | List 50 VMs | 12s | <2.5s (5x faster) | - | ⬜ |
| PERF-009 | Rollback 5 VMs | 79s | <15s (5x faster) | - | ⬜ |
| PERF-010 | Startup time | 2-3s | <0.2s | - | ⬜ |
| PERF-011 | Memory usage (idle) | 50-100MB | <20MB | - | ⬜ |

---

## 12. Configuration Tests

### 12.1 Configuration Loading (Go)
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| CFG-001 | Load from env vars | Set env vars only | Config loaded from env | ⬜ |
| CFG-002 | Load from user config | Set ~/.config/proxmox-snapshot-manager/config.yaml | Config loaded from file | ⬜ |
| CFG-003 | Load from current dir | Set ./proxmox-snapshot-manager.yaml | Config loaded from current dir | ⬜ |
| CFG-004 | Load from system config | Set /etc/proxmox-snapshot-manager/config.yaml | Config loaded from system | ⬜ |
| CFG-005 | Command line config override | Use --config flag | Specified config used | ⬜ |
| CFG-006 | Priority order validation | Set all config sources | Highest priority wins | ⬜ |

### 12.2 Configuration Options
| Test ID | Test Case | Test Method | Expected Result | Status |
|---------|-----------|-------------|-----------------|--------|
| CFG-007 | Set max concurrent operations | Configure in YAML | Limit respected | ⬜ |
| CFG-008 | Set logging level | Configure log level | Correct verbosity | ⬜ |
| CFG-009 | Enable/disable color | Configure color_output | Colors enabled/disabled | ⬜ |
| CFG-010 | Enable/disable progress bars | Configure progress_bars | Progress shown/hidden | ⬜ |

---

## 13. Cross-Implementation Comparison Tests

### 13.1 Output Comparison
| Test ID | Test Case | Comparison Method | Expected Result | Status |
|---------|-----------|-------------------|-----------------|--------|
| CMP-001 | Compare VM list output | Run list on both, diff output | Identical VM list | ⬜ |
| CMP-002 | Compare snapshot list | Run list on both, diff output | Identical snapshot list | ⬜ |
| CMP-003 | Compare error messages | Trigger same error on both | Similar error messages | ⬜ |
| CMP-004 | Compare snapshot naming | Create with same prefix | Same naming pattern | ⬜ |
| CMP-005 | Compare selection results | Use same selection pattern | Same VMs selected | ⬜ |

### 13.2 Behavior Comparison
| Test ID | Test Case | Comparison Method | Expected Result | Status |
|---------|-----------|-------------------|-----------------|--------|
| CMP-006 | Compare task monitoring | Monitor same task | Similar monitoring behavior | ⬜ |
| CMP-007 | Compare progress display | Bulk operation on both | Similar progress format | ⬜ |
| CMP-008 | Compare confirmation prompts | Same operation on both | Similar prompts | ⬜ |
| CMP-009 | Compare batch mode | Same operation with --yes | No prompts on either | ⬜ |

---

## Test Execution Plan

### Phase 1: Core Functionality (Week 1)
- ✅ API & Authentication (API-001 to API-010)
- ✅ Node & VM Discovery (NODE-001 to VM-008)
- ✅ VM Selection Patterns (SEL-001 to SEL-019)

### Phase 2: Snapshot Operations (Week 2)
- ✅ Create Snapshots (SNAP-001 to SNAP-008)
- ✅ List Snapshots (SNAP-009 to SNAP-011)
- ✅ Rollback Snapshots (SNAP-012 to SNAP-017)
- ✅ Delete Snapshots (SNAP-018 to SNAP-023)

### Phase 3: VM Lifecycle & Bulk (Week 3)
- ✅ VM Lifecycle (VM-L-001 to VM-L-012)
- ✅ Bulk Operations (BULK-001 to BULK-012)

### Phase 4: Backup Operations (Week 4) - **✅ NOW COMPLETE IN GO**
- ✅ Storage Discovery (BKUP-001 to BKUP-003)
- ✅ Create/List/Restore (BKUP-004 to BKUP-011)
- ✅ Delete Backups (BKUP-012 to BKUP-016)

### Phase 5: Interactive & Polish (Week 5)
- ✅ Main Menu Navigation (MENU-001 to MENU-012)
- ⚠️ Bulk Interactive (MENU-013 to MENU-016) - **Go Missing**
- ✅ Quick Operations (MENU-017 to MENU-019) - **✅ NOW COMPLETE IN GO**
- ✅ Error Handling (ERR-001 to ERR-015)

### Phase 6: Performance & Configuration (Week 6)
- ✅ Concurrency (PERF-001 to PERF-005)
- ✅ Performance Benchmarks (PERF-006 to PERF-011)
- ✅ Configuration (CFG-001 to CFG-010)

### Phase 7: Cross-Validation (Week 7)
- ✅ Output Comparison (CMP-001 to CMP-005)
- ✅ Behavior Comparison (CMP-006 to CMP-009)

---

## Test Report Template

### Test Execution Summary
```markdown
## Test Run: [Date]
**Implementation:** Python / Go
**Version:** [version]
**Environment:** [test environment details]

### Results Summary
| Category | Total | Passed | Failed | Skipped | Pass Rate |
|----------|-------|--------|--------|---------|-----------|
| Core API | 10 | - | - | - | - |
| VM Discovery | 8 | - | - | - | - |
| VM Selection | 19 | - | - | - | - |
| Snapshots | 24 | - | - | - | - |
| VM Lifecycle | 12 | - | - | - | - |
| Bulk Ops | 12 | - | - | - | - |
| Backups | 16 | - | - | - | - |
| Interactive | 19 | - | - | - | - |
| Error Handling | 15 | - | - | - | - |
| Performance | 11 | - | - | - | - |
| Configuration | 10 | - | - | - | - |
| Comparison | 9 | - | - | - | - |
| **TOTAL** | **165** | **-** | **-** | **-** | **-%** |

### Failed Tests
[List of failed test IDs with details]

### Known Issues
[List of known issues and workarounds]

### Recommendations
[Recommendations for fixes and improvements]
```

---

## Automated Test Script Template

```bash
#!/bin/bash
# test-parity.sh - Automated feature parity testing

# Configuration
PYTHON_CMD="python3 /path/to/legacy/main.py"
GO_CMD="/path/to/proxmox-snapshot-manager"
TEST_VM="7300"
RESULTS_FILE="test-results-$(date +%Y%m%d-%H%M%S).md"

# Test counter
TOTAL=0
PASSED=0
FAILED=0

# Test function
run_test() {
    local test_id=$1
    local description=$2
    local python_cmd=$3
    local go_cmd=$4
    local expected=$5

    TOTAL=$((TOTAL + 1))
    echo "Testing: $test_id - $description"

    # Run Python
    python_result=$($python_cmd 2>&1)

    # Run Go
    go_result=$($go_cmd 2>&1)

    # Compare results
    if [[ "$python_result" == *"$expected"* ]] && [[ "$go_result" == *"$expected"* ]]; then
        echo "✅ PASS: $test_id"
        PASSED=$((PASSED + 1))
    else
        echo "❌ FAIL: $test_id"
        FAILED=$((FAILED + 1))
        echo "  Python: $python_result" >> $RESULTS_FILE
        echo "  Go: $go_result" >> $RESULTS_FILE
    fi
}

# Example tests
run_test "SEL-001" "Single VM selection" \
    "$PYTHON_CMD list --vmid $TEST_VM" \
    "$GO_CMD list --vmid $TEST_VM" \
    "Snapshots for VM $TEST_VM"

run_test "SNAP-001" "Create snapshot with prefix" \
    "$PYTHON_CMD create --vmid $TEST_VM --prefix test" \
    "$GO_CMD create --vmid $TEST_VM --prefix test" \
    "Snapshot created"

# Final report
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Total Tests: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Pass Rate: $(( PASSED * 100 / TOTAL ))%"
echo "Results saved to: $RESULTS_FILE"
```

---

## Summary

### Total Test Cases: 181 (16 new dry-run tests added)

**By Category:**
- Core API & Auth: 10 tests
- Node & VM Discovery: 8 tests
- VM Selection: 19 tests
- Snapshot Operations: 24 tests
- VM Lifecycle: 12 tests
- Bulk Operations: 12 tests
- **Dry-Run Safety: 16 tests** (✅ NEW - Go-specific feature)
- Backup Operations: 16 tests
- Interactive Menus: 19 tests
- Error Handling: 15 tests
- Performance: 11 tests
- Configuration: 10 tests
- Cross-Comparison: 9 tests

**Expected Results:**
- ✅ **Python**: 100% (165/165 applicable tests) - All features implemented
- ✅ **Go**: ~87% (156/181 total tests) - Most features implemented with superior performance
  - Includes 16 bonus dry-run safety tests (Go-exclusive feature)
- 🎯 **Performance**: Go is 5-10x faster where implemented

**Completed in Go (Since Last Update):**
1. ✅ Complete backup operations (16 tests) - **DONE**
2. ✅ Graceful shutdown (3 tests) - **DONE**
3. ✅ Quick operations (3 tests) - **DONE**
4. ✅ Storage validation (1 test) - **DONE**
5. ✅ **NEW**: Comprehensive dry-run safety for all operations

**Remaining Gaps in Go:**
1. ❌ Bulk interactive menu (4 tests: MENU-013 to MENU-016)
2. ❌ VM config display (few tests)
3. ❌ Reset VM operation (1 test: VM-L-012)
