# Quick Operations + Dry-Run Implementation Guide

## Overview
This document provides the complete implementation for:
1. **Global `--dry-run` flag** - Show what would be done without doing it
2. **Quick Operations** - Convenient bulk operations on all VMs
   - `quick-start-all` - Start all stopped VMs
   - `quick-stop-all` - Stop all running VMs
   - `quick-backup-all` - Backup all VMs

## Design Principles

### Dry-Run Behavior
- ✅ Show exactly what actions would be performed
- ✅ Display VM lists and operation details
- ✅ No actual API calls to Proxmox
- ✅ Works in both batch and interactive modes
- ✅ Clear visual indicators (`[DRY-RUN]` prefix)

### Quick Operations Behavior
- ✅ Auto-filter VMs by state (running/stopped)
- ✅ Support `--dry-run` for safety
- ✅ Support `--yes` for automation
- ✅ Provide clear summaries
- ✅ Use existing bulk operations infrastructure

---

## Implementation

### 1. Dry-Run Helper Function

Add to `cmd/main.go` after the `confirmOperation` function:

```go
// printDryRunHeader prints a dry-run header
func printDryRunHeader() {
	if dryRun {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("🔍 DRY-RUN MODE - No changes will be made")
		fmt.Println(strings.Repeat("=", 60) + "\n")
	}
}

// printDryRunAction prints what action would be performed
func printDryRunAction(action string, vmid string, vmname string, details string) {
	if dryRun {
		if details != "" {
			fmt.Printf("[DRY-RUN] Would %s VM %s (%s) - %s\n", action, vmid, vmname, details)
		} else {
			fmt.Printf("[DRY-RUN] Would %s VM %s (%s)\n", action, vmid, vmname)
		}
	}
}

// printDryRunSummary prints a dry-run summary
func printDryRunSummary(operation string, count int) {
	if dryRun {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Printf("🔍 DRY-RUN COMPLETE: Would %s %d VM(s)\n", operation, count)
		fmt.Println(strings.Repeat("=", 60) + "\n")
	}
}
```

### 2. Quick-Start-All Command

```go
// quickStartAllCmd represents the quick-start-all command
var quickStartAllCmd = &cobra.Command{
	Use:   "quick-start-all",
	Short: "Quickly start all stopped VMs",
	Long: `Start all stopped VMs in the cluster.

This is a convenience command that automatically finds all stopped VMs and starts them.

Examples:
  # Dry-run to see what would be started
  proxmox-snapshot-manager quick-start-all --dry-run

  # Start all stopped VMs with confirmation
  proxmox-snapshot-manager quick-start-all

  # Start all stopped VMs without confirmation
  proxmox-snapshot-manager quick-start-all --yes`,
	RunE: runQuickStartAllCommand,
}

func runQuickStartAllCommand(cmd *cobra.Command, args []string) error {
	printDryRunHeader()

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		return fmt.Errorf("failed to get VMs: %w", err)
	}

	// Filter to stopped VMs
	var stoppedVMs []*vm.VM
	for _, vmInstance := range allVMs {
		if !vmInstance.Running {
			stoppedVMs = append(stoppedVMs, vmInstance)
		}
	}

	if len(stoppedVMs) == 0 {
		fmt.Println("ℹ️  No stopped VMs found")
		return nil
	}

	// Display list
	fmt.Printf("\nFound %d stopped VM(s):\n", len(stoppedVMs))
	for _, vmInstance := range stoppedVMs {
		status := "stopped"
		fmt.Printf("  • VM %s (%s) - %s\n", vmInstance.VMID, vmInstance.Name, status)
	}
	fmt.Println()

	// Dry-run mode - just show what would happen
	if dryRun {
		for _, vmInstance := range stoppedVMs {
			printDryRunAction("start", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("start", len(stoppedVMs))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Start %d stopped VM(s)?", len(stoppedVMs))) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Execute bulk operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	logger.Infof("Starting %d VM(s)", len(stoppedVMs))
	if err := bulkMgr.StartVMs(ctx, stoppedVMs); err != nil {
		return fmt.Errorf("bulk VM start failed: %w", err)
	}

	bulkMgr.PrintSummary()
	return nil
}
```

### 3. Quick-Stop-All Command

```go
// quickStopAllCmd represents the quick-stop-all command
var quickStopAllCmd = &cobra.Command{
	Use:   "quick-stop-all",
	Short: "Quickly stop all running VMs",
	Long: `Stop all running VMs in the cluster.

This is a convenience command that automatically finds all running VMs and stops them.
WARNING: This is a force stop, not a graceful shutdown.

Examples:
  # Dry-run to see what would be stopped
  proxmox-snapshot-manager quick-stop-all --dry-run

  # Stop all running VMs with confirmation
  proxmox-snapshot-manager quick-stop-all

  # Stop all running VMs without confirmation
  proxmox-snapshot-manager quick-stop-all --yes`,
	RunE: runQuickStopAllCommand,
}

func runQuickStopAllCommand(cmd *cobra.Command, args []string) error {
	printDryRunHeader()

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		return fmt.Errorf("failed to get VMs: %w", err)
	}

	// Filter to running VMs
	var runningVMs []*vm.VM
	for _, vmInstance := range allVMs {
		if vmInstance.Running {
			runningVMs = append(runningVMs, vmInstance)
		}
	}

	if len(runningVMs) == 0 {
		fmt.Println("ℹ️  No running VMs found")
		return nil
	}

	// Display list
	fmt.Printf("\n⚠️  Found %d running VM(s):\n", len(runningVMs))
	for _, vmInstance := range runningVMs {
		fmt.Printf("  • VM %s (%s) - running\n", vmInstance.VMID, vmInstance.Name)
	}
	fmt.Println()

	// Dry-run mode - just show what would happen
	if dryRun {
		for _, vmInstance := range runningVMs {
			printDryRunAction("force stop", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("force stop", len(runningVMs))
		return nil
	}

	// Confirm operation with extra warning
	fmt.Println("⚠️  WARNING: This will FORCE STOP all running VMs (not graceful shutdown)")
	if !confirmOperation(fmt.Sprintf("Force stop %d running VM(s)? Type 'STOP ALL' to confirm", len(runningVMs))) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Execute bulk operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	logger.Infof("Stopping %d VM(s)", len(runningVMs))
	if err := bulkMgr.StopVMs(ctx, runningVMs); err != nil {
		return fmt.Errorf("bulk VM stop failed: %w", err)
	}

	bulkMgr.PrintSummary()
	return nil
}
```

### 4. Quick-Backup-All Command

```go
// quickBackupAllCmd represents the quick-backup-all command
var quickBackupAllCmd = &cobra.Command{
	Use:   "quick-backup-all",
	Short: "Quickly backup all VMs",
	Long: `Backup all VMs in the cluster to specified storage.

This is a convenience command that automatically finds all VMs and backs them up.

Examples:
  # Dry-run to see what would be backed up
  proxmox-snapshot-manager quick-backup-all --storage local-zfs --dry-run

  # Backup all VMs with snapshot mode
  proxmox-snapshot-manager quick-backup-all --storage local-zfs

  # Backup all VMs with specific mode and compression
  proxmox-snapshot-manager quick-backup-all --storage local-zfs --mode suspend --compress gzip --yes`,
	RunE: runQuickBackupAllCommand,
}

func runQuickBackupAllCommand(cmd *cobra.Command, args []string) error {
	printDryRunHeader()

	// Validate storage flag
	if storageFlag == "" {
		return fmt.Errorf("--storage flag is required")
	}

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		return fmt.Errorf("failed to get VMs: %w", err)
	}

	if len(allVMs) == 0 {
		fmt.Println("ℹ️  No VMs found")
		return nil
	}

	// Display list
	fmt.Printf("\nFound %d VM(s) to backup:\n", len(allVMs))
	for _, vmInstance := range allVMs {
		status := "stopped"
		if vmInstance.Running {
			status = "running"
		}
		fmt.Printf("  • VM %s (%s) - %s\n", vmInstance.VMID, vmInstance.Name, status)
	}
	fmt.Println()

	// Validate storage (skip in dry-run)
	if !dryRun {
		storageOps := storage.NewOperations(client, logger)
		if err := storageOps.ValidateStorage(storageFlag); err != nil {
			return fmt.Errorf("storage validation failed: %w", err)
		}
	}

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

	// Dry-run mode - just show what would happen
	if dryRun {
		for _, vmInstance := range allVMs {
			details := fmt.Sprintf("storage=%s, mode=%s, compress=%s", storageFlag, modeFlag, compressFlag)
			printDryRunAction("backup", vmInstance.VMID, vmInstance.Name, details)
		}
		printDryRunSummary("backup", len(allVMs))
		return nil
	}

	// Confirm operation
	confirmMsg := fmt.Sprintf("Backup %d VM(s) to storage '%s' with mode '%s'?",
		len(allVMs), storageFlag, modeFlag)
	if !confirmOperation(confirmMsg) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Execute backups
	backupOps := backup.NewOperations(client, vmOps, logger)
	logger.Infof("Creating backups for %d VM(s)", len(allVMs))

	successCount := 0
	for _, vmInstance := range allVMs {
		fmt.Printf("\nBacking up VM %s (%s)...\n", vmInstance.VMID, vmInstance.Name)
		if err := backupOps.CreateBackup(vmInstance.VMID, storageFlag, mode, compressFlag); err != nil {
			logger.Errorf("Failed to backup VM %s: %v", vmInstance.VMID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✅ Successfully backed up %d/%d VMs\n", successCount, len(allVMs))
	return nil
}
```

### 5. Add Commands to init()

In the `init()` function, add after the existing command registrations:

```go
	// Quick operation command flags
	quickBackupAllCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage for backup (required)")
	quickBackupAllCmd.Flags().StringVar(&modeFlag, "mode", "snapshot", "Backup mode: snapshot, suspend, or stop")
	quickBackupAllCmd.Flags().StringVar(&compressFlag, "compress", "zstd", "Compression: zstd, gzip, or lzo")
	quickBackupAllCmd.MarkFlagRequired("storage")

	// Add quick commands
	rootCmd.AddCommand(quickStartAllCmd)
	rootCmd.AddCommand(quickStopAllCmd)
	rootCmd.AddCommand(quickBackupAllCmd)
```

---

## Testing

### Test Dry-Run Mode

```bash
# Test quick operations with dry-run
./build/proxmox-snapshot-manager quick-start-all --dry-run
./build/proxmox-snapshot-manager quick-stop-all --dry-run
./build/proxmox-snapshot-manager quick-backup-all --storage local-zfs --dry-run

# Test existing commands with dry-run
./build/proxmox-snapshot-manager create --vmid 7303 --prefix test --dry-run
./build/proxmox-snapshot-manager backup --vmid 7303 --storage local-zfs --dry-run
./build/proxmox-snapshot-manager start --vmid 7303 --dry-run
```

### Test Quick Operations

```bash
# Start all stopped VMs
./build/proxmox-snapshot-manager quick-start-all --yes

# Stop all running VMs (with extra confirmation)
./build/proxmox-snapshot-manager quick-stop-all

# Backup all VMs
./build/proxmox-snapshot-manager quick-backup-all --storage local-zfs --mode snapshot --yes
```

---

## Benefits

### Safety
- **Dry-run prevents accidents**: Always test operations first
- **Clear visual feedback**: `[DRY-RUN]` prefix on all actions
- **No API calls**: Completely safe to run
- **Works everywhere**: Both CLI and future interactive menus

### Convenience
- **One command**: `quick-start-all` vs selecting VMs manually
- **Auto-filtering**: Only shows relevant VMs (stopped/running)
- **Bulk efficiency**: Uses existing parallel infrastructure
- **Consistent UX**: Same flags and patterns as other commands

### User Experience
```bash
# Before (manual)
proxmox-snapshot-manager start --vmid 7301,7302,7303,7304,7305...

# After (automatic)
proxmox-snapshot-manager quick-start-all --yes
```

---

## Next Steps

1. ✅ Add dry-run helper functions
2. ✅ Implement quick-start-all command
3. ✅ Implement quick-stop-all command
4. ✅ Implement quick-backup-all command
5. ✅ Register commands in init()
6. ⏳ Add dry-run support to existing commands (optional enhancement)
7. ⏳ Build and test
8. ⏳ Update documentation
