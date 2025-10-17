package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yg-codes/proxmox-admin-cli/pkg/backup"
	"github.com/yg-codes/proxmox-admin-cli/pkg/storage"
	"github.com/yg-codes/proxmox-admin-cli/pkg/vm"
)

// Flags for bulk operations
var (
	bulkStorageFlag  string
	bulkModeFlag     string
	bulkCompressFlag string
)

// bulkCmd represents the bulk operations command group
var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Bulk operations on all VMs",
	Long: `Perform operations on all VMs in the cluster.

Bulk operations automatically discover and act on all VMs that match
the operation criteria (e.g., all stopped VMs for start, all running VMs for stop).

Available bulk operations:
  start  - Start all stopped VMs
  stop   - Stop all running VMs
  backup - Backup all VMs`,
}

// bulkStartCmd starts all stopped VMs
var bulkStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start all stopped VMs",
	Long: `Start all stopped VMs in the cluster.

This command automatically finds all stopped VMs and starts them.

Examples:
  # Dry-run to see what would be started
  proxmox-admin-cli vm bulk start --dry-run

  # Start all stopped VMs with confirmation
  proxmox-admin-cli vm bulk start

  # Start all stopped VMs without confirmation
  proxmox-admin-cli vm bulk start --yes`,
	RunE: runBulkStartCommand,
}

// bulkStopCmd stops all running VMs
var bulkStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all running VMs",
	Long: `Stop all running VMs in the cluster.

This command automatically finds all running VMs and stops them.
WARNING: This is a force stop, not a graceful shutdown.

Examples:
  # Dry-run to see what would be stopped
  proxmox-admin-cli vm bulk stop --dry-run

  # Stop all running VMs with confirmation
  proxmox-admin-cli vm bulk stop

  # Stop all running VMs without confirmation
  proxmox-admin-cli vm bulk stop --yes`,
	RunE: runBulkStopCommand,
}

// bulkBackupCmd backs up all VMs
var bulkBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup all VMs",
	Long: `Backup all VMs in the cluster to specified storage.

This command automatically finds all VMs and backs them up.

Examples:
  # Dry-run to see what would be backed up
  proxmox-admin-cli vm bulk backup --storage local-zfs --dry-run

  # Backup all VMs with snapshot mode
  proxmox-admin-cli vm bulk backup --storage local-zfs

  # Backup all VMs with specific mode and compression
  proxmox-admin-cli vm bulk backup --storage local-zfs --mode suspend --compress gzip --yes`,
	RunE: runBulkBackupCommand,
}

// initBulkCommands sets up the bulk command hierarchy
func initBulkCommands() {
	// Add bulk subcommands
	bulkCmd.AddCommand(bulkStartCmd)
	bulkCmd.AddCommand(bulkStopCmd)
	bulkCmd.AddCommand(bulkBackupCmd)

	// Bulk backup flags
	bulkBackupCmd.Flags().StringVar(&bulkStorageFlag, "storage", "", "Storage for backup (required)")
	bulkBackupCmd.Flags().StringVar(&bulkModeFlag, "mode", "snapshot", "Backup mode: snapshot, suspend, or stop")
	bulkBackupCmd.Flags().StringVar(&bulkCompressFlag, "compress", "zstd", "Compression: zstd, gzip, or lzo")
	bulkBackupCmd.MarkFlagRequired("storage")
}

// runBulkStartCommand handles the bulk start operation
func runBulkStartCommand(cmd *cobra.Command, args []string) error {
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
		fmt.Printf("  • VM %s (%s) - stopped\n", vmInstance.VMID, vmInstance.Name)
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

// runBulkStopCommand handles the bulk stop operation
func runBulkStopCommand(cmd *cobra.Command, args []string) error {
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
	if !confirmOperation(fmt.Sprintf("Force stop %d running VM(s)?", len(runningVMs))) {
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

// runBulkBackupCommand handles the bulk backup operation
func runBulkBackupCommand(cmd *cobra.Command, args []string) error {
	printDryRunHeader()

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
		if err := storageOps.ValidateStorage(bulkStorageFlag); err != nil {
			return fmt.Errorf("storage validation failed: %w", err)
		}
	}

	// Validate mode
	var mode backup.BackupMode
	switch bulkModeFlag {
	case "snapshot":
		mode = backup.BackupModeSnapshot
	case "suspend":
		mode = backup.BackupModeSuspend
	case "stop":
		mode = backup.BackupModeStop
	default:
		return fmt.Errorf("invalid mode: %s (must be snapshot, suspend, or stop)", bulkModeFlag)
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		for _, vmInstance := range allVMs {
			details := fmt.Sprintf("storage=%s, mode=%s, compress=%s", bulkStorageFlag, bulkModeFlag, bulkCompressFlag)
			printDryRunAction("backup", vmInstance.VMID, vmInstance.Name, details)
		}
		printDryRunSummary("backup", len(allVMs))
		return nil
	}

	// Confirm operation
	confirmMsg := fmt.Sprintf("Backup %d VM(s) to storage '%s' with mode '%s'?",
		len(allVMs), bulkStorageFlag, bulkModeFlag)
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
		if err := backupOps.CreateBackup(vmInstance.VMID, bulkStorageFlag, mode, bulkCompressFlag); err != nil {
			logger.Errorf("Failed to backup VM %s: %v", vmInstance.VMID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✅ Successfully backed up %d/%d VMs\n", successCount, len(allVMs))
	return nil
}
