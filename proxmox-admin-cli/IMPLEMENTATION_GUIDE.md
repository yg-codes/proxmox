# Backup Operations Implementation Guide

## ✅ Completed Core Packages

### 1. pkg/storage/operations.go
**Purpose**: Storage discovery and management

**Key Functions:**
- `GetVMStorages()` - Get storages suitable for VM disks
- `GetBackupStorages()` - Get backup-capable storages
- `DisplayVMStorages()` - Display VM storages in formatted table
- `DisplayBackupStorages()` - Display backup storages in formatted table
- `ValidateStorage()` - Validate storage exists and is active

**Usage Example:**
```go
storageOps := storage.NewOperations(apiClient, logger)
storages, err := storageOps.GetBackupStorages()
storageOps.DisplayBackupStorages()
```

### 2. pkg/backup/operations.go
**Purpose**: Complete backup lifecycle management

**Key Functions:**
- `CreateBackup(vmid, storage, mode, compress)` - Create VM backup
  - Modes: "snapshot", "suspend", "stop"
  - Compression: "zstd", "gzip", "lzo"
- `ListBackupsForVM(vmid, storage)` - List all backups for a VM
- `DisplayBackups(vmid, storage)` - Display backups in formatted table
- `RestoreBackup(vmid, volid, node, storage)` - Restore VM from backup
- `DeleteBackup(backup)` - Delete single backup
- `DeleteBackupsByPattern(vmid, storage, pattern)` - Delete by wildcard pattern
- `DeleteOldBackups(vmid, storage, keepCount, maxAgeDays)` - Retention-based cleanup

**Backup Struct:**
```go
type Backup struct {
    VolID       string  // e.g., "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst"
    VMID        string
    Storage     string
    Node        string
    Size        float64 // Size in GB
    CreatedTime int64
    Format      string
    Content     string
}
```

**Usage Example:**
```go
backupOps := backup.NewOperations(apiClient, vmOps, logger)
err := backupOps.CreateBackup("7303", "local-zfs", backup.BackupModeSnapshot, "zstd")
backups, err := backupOps.ListBackupsForVM("7303", "")
err = backupOps.DisplayBackups("7303", "")
```

### 3. pkg/protection/operations.go
**Purpose**: VM protection handling

**Key Functions:**
- `IsProtected(vmid)` - Check if VM is protected
- `CheckAndWarn(vmid, operation)` - Check and warn user about protection
- `SetProtection(vmid, protect)` - Enable/disable VM protection

**Usage Example:**
```go
protectionOps := protection.NewOperations(apiClient, vmOps, logger)
protected, err := protectionOps.IsProtected("7303")
protectionOps.CheckAndWarn("7303", "restore")
```

---

## 🔨 CLI Commands to Implement

### In cmd/main.go

#### 1. Add Backup Command

```go
var backupCmd = &cobra.Command{
    Use:   "backup",
    Short: "Create VM backup",
    Long:  "Create a backup of specified VM(s) to storage",
    Run: func(cmd *cobra.Command, args []string) {
        runBackupCommand(cmd, args)
    },
}

func init() {
    backupCmd.Flags().StringSliceVar(&vmIDs, "vmid", []string{}, "VM ID(s) (comma-separated, range, or pattern)")
    backupCmd.Flags().StringSliceVar(&vmNames, "vmname", []string{}, "VM name(s)")
    backupCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage for backup (required)")
    backupCmd.Flags().StringVar(&modeFlag, "mode", "snapshot", "Backup mode: snapshot, suspend, or stop")
    backupCmd.Flags().StringVar(&compressFlag, "compress", "zstd", "Compression: zstd, gzip, or lzo")
    backupCmd.Flags().BoolVarP(&batchMode, "batch", "b", false, "Batch mode (no confirmations)")
    backupCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Automatic yes to prompts")
    backupCmd.MarkFlagRequired("storage")

    rootCmd.AddCommand(backupCmd)
}

func runBackupCommand(cmd *cobra.Command, args []string) error {
    // Initialize
    if err := initializeApp(cmd, args); err != nil {
        return err
    }

    // Resolve VMs
    vms, err := resolveVMs(vmIDs, vmNames)
    if err != nil {
        return err
    }

    // Validate storage
    storageOps := storage.NewOperations(apiClient, logger)
    if err := storageOps.ValidateStorage(storageFlag); err != nil {
        return err
    }

    // Initialize backup operations
    backupOps := backup.NewOperations(apiClient, vmOps, logger)

    // Validate mode
    var mode backup.BackupMode
    switch modeFlag {
    case "snapshot":
        mode = backup.BackupModeSnapshot
    case "suspend":
        mode = backup.BackupModeSuspend
    case "stop":
        mode = backup.BackupModeStop
    default:
        return fmt.Errorf("invalid mode: %s (must be snapshot, suspend, or stop)", modeFlag)
    }

    // Single or bulk operation
    if len(vms) == 1 {
        return backupOps.CreateBackup(vms[0].VMID, storageFlag, mode, compressFlag)
    }

    // Bulk backup (can be added to bulk package)
    fmt.Printf("\nCreating backups for %d VMs\n", len(vms))
    for _, vm := range vms {
        if err := backupOps.CreateBackup(vm.VMID, storageFlag, mode, compressFlag); err != nil {
            logger.Errorf("Failed to backup VM %s: %v", vm.VMID, err)
        }
    }
    return nil
}
```

#### 2. Add List-Backups Command

```go
var listBackupsCmd = &cobra.Command{
    Use:   "list-backups",
    Short: "List VM backups",
    Long:  "List all backups for specified VM(s)",
    Run: func(cmd *cobra.Command, args []string) {
        runListBackupsCommand(cmd, args)
    },
}

func init() {
    listBackupsCmd.Flags().StringSliceVar(&vmIDs, "vmid", []string{}, "VM ID(s)")
    listBackupsCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage to check (optional, checks all if not specified)")

    rootCmd.AddCommand(listBackupsCmd)
}

func runListBackupsCommand(cmd *cobra.Command, args []string) error {
    if err := initializeApp(cmd, args); err != nil {
        return err
    }

    vms, err := resolveVMs(vmIDs, vmNames)
    if err != nil {
        return err
    }

    backupOps := backup.NewOperations(apiClient, vmOps, logger)

    for _, vm := range vms {
        if err := backupOps.DisplayBackups(vm.VMID, storageFlag); err != nil {
            logger.Errorf("Failed to list backups for VM %s: %v", vm.VMID, err)
        }
    }
    return nil
}
```

#### 3. Add Restore Command

```go
var restoreCmd = &cobra.Command{
    Use:   "restore",
    Short: "Restore VM from backup",
    Long:  "Restore a VM from a backup file",
    Run: func(cmd *cobra.Command, args []string) {
        runRestoreCommand(cmd, args)
    },
}

func init() {
    restoreCmd.Flags().StringVar(&vmIDFlag, "vmid", "", "Target VM ID (required)")
    restoreCmd.Flags().StringVar(&backupFileFlag, "backup-file", "", "Backup volid (required)")
    restoreCmd.Flags().StringVar(&nodeFlag, "node", "", "Node name (required)")
    restoreCmd.Flags().StringVar(&storageFlag, "storage", "", "Target storage (optional)")
    restoreCmd.Flags().BoolVarP(&batchMode, "batch", "b", false, "Batch mode")
    restoreCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Automatic yes to prompts")
    restoreCmd.MarkFlagRequired("vmid")
    restoreCmd.MarkFlagRequired("backup-file")
    restoreCmd.MarkFlagRequired("node")

    rootCmd.AddCommand(restoreCmd)
}

func runRestoreCommand(cmd *cobra.Command, args []string) error {
    if err := initializeApp(cmd, args); err != nil {
        return err
    }

    // Protection check
    protectionOps := protection.NewOperations(apiClient, vmOps, logger)
    protectionOps.CheckAndWarn(vmIDFlag, "restore")

    // Confirm if not batch mode
    if !batchMode && !yesFlag {
        if !confirmOperation(fmt.Sprintf("Restore VM %s from %s? This will OVERWRITE the existing VM!", vmIDFlag, backupFileFlag)) {
            fmt.Println("Restore cancelled")
            return nil
        }
    }

    backupOps := backup.NewOperations(apiClient, vmOps, logger)
    return backupOps.RestoreBackup(vmIDFlag, backupFileFlag, nodeFlag, storageFlag)
}
```

#### 4. Add Delete-Backups Command

```go
var deleteBackupsCmd = &cobra.Command{
    Use:   "delete-backups",
    Short: "Delete VM backups",
    Long:  "Delete specific backup(s) or cleanup old backups",
    Run: func(cmd *cobra.Command, args []string) {
        runDeleteBackupsCommand(cmd, args)
    },
}

func init() {
    deleteBackupsCmd.Flags().StringVar(&vmIDFlag, "vmid", "", "VM ID (required)")
    deleteBackupsCmd.Flags().StringVar(&backupFileFlag, "backup-file", "", "Specific backup volid to delete")
    deleteBackupsCmd.Flags().StringVar(&patternFlag, "pattern", "", "Delete backups matching pattern (e.g., '*2024*')")
    deleteBackupsCmd.Flags().IntVar(&keepCountFlag, "keep-count", 0, "Keep only N most recent backups")
    deleteBackupsCmd.Flags().IntVar(&maxAgeDaysFlag, "max-age-days", 0, "Delete backups older than N days")
    deleteBackupsCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage to check")
    deleteBackupsCmd.Flags().BoolVarP(&batchMode, "batch", "b", false, "Batch mode")
    deleteBackupsCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Automatic yes to prompts")
    deleteBackupsCmd.MarkFlagRequired("vmid")

    rootCmd.AddCommand(deleteBackupsCmd)
}

func runDeleteBackupsCommand(cmd *cobra.Command, args []string) error {
    if err := initializeApp(cmd, args); err != nil {
        return err
    }

    backupOps := backup.NewOperations(apiClient, vmOps, logger)

    // Specific backup deletion
    if backupFileFlag != "" {
        if !batchMode && !yesFlag {
            if !confirmOperation(fmt.Sprintf("Delete backup %s?", backupFileFlag)) {
                fmt.Println("Deletion cancelled")
                return nil
            }
        }

        backups, err := backupOps.ListBackupsForVM(vmIDFlag, storageFlag)
        if err != nil {
            return err
        }

        for _, backup := range backups {
            if backup.VolID == backupFileFlag {
                return backupOps.DeleteBackup(backup)
            }
        }
        return fmt.Errorf("backup not found: %s", backupFileFlag)
    }

    // Pattern-based deletion
    if patternFlag != "" {
        if !batchMode && !yesFlag {
            if !confirmOperation(fmt.Sprintf("Delete backups matching pattern '%s'?", patternFlag)) {
                fmt.Println("Deletion cancelled")
                return nil
            }
        }
        deleted, err := backupOps.DeleteBackupsByPattern(vmIDFlag, storageFlag, patternFlag)
        if err != nil {
            return err
        }
        fmt.Printf("✅ Deleted %d backup(s)\n", deleted)
        return nil
    }

    // Retention-based cleanup
    if keepCountFlag > 0 || maxAgeDaysFlag > 0 {
        if !batchMode && !yesFlag {
            if !confirmOperation(fmt.Sprintf("Cleanup old backups (keep=%d, max-age=%d days)?", keepCountFlag, maxAgeDaysFlag)) {
                fmt.Println("Cleanup cancelled")
                return nil
            }
        }
        deleted, err := backupOps.DeleteOldBackups(vmIDFlag, storageFlag, keepCountFlag, maxAgeDaysFlag)
        if err != nil {
            return err
        }
        fmt.Printf("✅ Cleaned up %d backup(s)\n", deleted)
        return nil
    }

    return fmt.Errorf("must specify --backup-file, --pattern, --keep-count, or --max-age-days")
}
```

#### 5. Add Shutdown Command

```go
var shutdownCmd = &cobra.Command{
    Use:   "shutdown",
    Short: "Gracefully shutdown VM(s)",
    Long:  "Send ACPI shutdown signal to VM(s)",
    Run: func(cmd *cobra.Command, args []string) {
        runShutdownCommand(cmd, args)
    },
}

func init() {
    shutdownCmd.Flags().StringSliceVar(&vmIDs, "vmid", []string{}, "VM ID(s)")
    shutdownCmd.Flags().StringSliceVar(&vmNames, "vmname", []string{}, "VM name(s)")
    shutdownCmd.Flags().BoolVarP(&batchMode, "batch", "b", false, "Batch mode")
    shutdownCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Automatic yes to prompts")

    rootCmd.AddCommand(shutdownCmd)
}

func runShutdownCommand(cmd *cobra.Command, args []string) error {
    if err := initializeApp(cmd, args); err != nil {
        return err
    }

    vms, err := resolveVMs(vmIDs, vmNames)
    if err != nil {
        return err
    }

    if !batchMode && !yesFlag {
        if !confirmOperation(fmt.Sprintf("Gracefully shutdown %d VM(s)?", len(vms))) {
            fmt.Println("Shutdown cancelled")
            return nil
        }
    }

    for _, vm := range vms {
        if err := vmOps.ShutdownVM(vm.VMID); err != nil {
            logger.Errorf("Failed to shutdown VM %s: %v", vm.VMID, err)
        }
    }
    return nil
}
```

---

## 📋 Required Global Variables

Add these to the top of `cmd/main.go`:

```go
var (
    // Existing variables...

    // Backup-related flags
    storageFlag     string
    modeFlag        string
    compressFlag    string
    backupFileFlag  string
    nodeFlag        string
    patternFlag     string
    keepCountFlag   int
    maxAgeDaysFlag  int
)
```

---

## 🔄 Integration Steps

1. **Import new packages** in `cmd/main.go`:
```go
import (
    // ... existing imports
    "proxmox-snapshot-manager/pkg/backup"
    "proxmox-snapshot-manager/pkg/storage"
    "proxmox-snapshot-manager/pkg/protection"
)
```

2. **Add commands** to `init()` function

3. **Update go.mod** if needed:
```bash
cd proxmox-snapshot-manager-go
go mod tidy
```

4. **Build and test**:
```bash
make build
./build/proxmox-snapshot-manager backup --help
```

---

## ✅ Test Commands

```bash
# List backup storages
./build/proxmox-snapshot-manager list-storages

# Create backup
./build/proxmox-snapshot-manager backup --vmid 7303 --storage local-zfs --mode snapshot

# List backups
./build/proxmox-snapshot-manager list-backups --vmid 7303

# Restore from backup
./build/proxmox-snapshot-manager restore --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --node pve

# Delete specific backup
./build/proxmox-snapshot-manager delete-backups --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --yes

# Cleanup old backups
./build/proxmox-snapshot-manager delete-backups --vmid 7303 --keep-count 5 --yes

# Graceful shutdown
./build/proxmox-snapshot-manager shutdown --vmid 7303,7304,7305 --yes
```

---

## 📊 Implementation Status

### ✅ Completed (Core Packages)
- [x] pkg/storage - Storage discovery and management
- [x] pkg/backup - Complete backup lifecycle (create, list, restore, delete)
- [x] pkg/protection - VM protection handling

### ✅ Completed (CLI Integration)
- [x] backup command
- [x] list-backups command
- [x] restore command
- [x] delete-backups command
- [x] shutdown command

### ✅ Completed (Future Enhancements)
- [x] Bulk operations interactive menu
- [x] Quick operations (quick-start-all, quick-stop-all, quick-backup-all)
- [x] Interactive backup selection
- [x] Global --dry-run flag for safety
- [x] Dry-run support for all operations

### ✅ Additional Features Implemented
- [x] Global --dry-run flag for all commands
- [x] Dry-run support for create, rollback, delete, start, stop, backup, delete-backups, shutdown commands
- [x] Dry-run support for all interactive menu operations
- [x] Quick operations: quick-start-all, quick-stop-all, quick-backup-all
- [x] Interactive menu integration for all backup operations

---

## 📝 Notes

1. **Error Handling**: All operations include comprehensive error handling and user feedback

2. **Batch Mode**: The `--batch` and `--yes` flags allow automation without prompts

3. **Protection Check**: The protection package warns users about protected VMs

4. **Storage Validation**: Storage is validated before backup operations

5. **Volid Format**: Backups use the full volid format: `<storage>:<type>/<path>`

6. **Concurrent Operations**: For bulk backups, consider using the existing `bulk` package

7. **Task Monitoring**: All operations use the existing task monitoring from vm.Operations

---

## 🚀 Next Steps

1. Add the CLI commands to `cmd/main.go` following the examples above
2. Add global variables for new flags
3. Import the new packages
4. Build and test each command
5. Update documentation (README.md, CLAUDE.md)
6. Run test specification tests (BKUP-001 to BKUP-016)
7. Update FUNCTIONAL_SPECIFICATION.md to mark features as implemented
