package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/api"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/backup"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/bulk"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/config"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/protection"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/snapshot"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/storage"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/vm"
)

var (
	cfg        *config.Config
	logger     *logrus.Logger
	client     *api.Client
	vmOps      *vm.Operations
	vmSelector *vm.Selector
	snapOps    *snapshot.Operations
	bulkMgr    *bulk.Manager

	// Global flags
	configPath  string
	batchMode   bool
	autoConfirm bool
	verbose     bool
	quiet       bool
	dryRun      bool

	// Backup-related flags
	storageFlag    string
	modeFlag       string
	compressFlag   string
	backupFileFlag string
	nodeFlag       string
	patternFlag    string
	keepCountFlag  int
	maxAgeDaysFlag int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "proxmox-snapshot-manager",
	Short: "Proxmox VM Snapshot Management Tool",
	Long: `A comprehensive Proxmox VM snapshot management tool written in Go.

Provides powerful snapshot management capabilities including:
- Create snapshots with intelligent naming
- Rollback to previous snapshots
- List and manage existing snapshots
- Delete snapshots with safety checks
- Bulk snapshot operations with concurrent execution
- Real-time task monitoring and progress tracking

Authentication can be done via API tokens (recommended) or username/password.
Set environment variables: PVE_HOST, PVE_USER, PVE_TOKEN_NAME, PVE_TOKEN_VALUE`,
	PersistentPreRunE: initializeApp,
	Run: func(cmd *cobra.Command, args []string) {
		if batchMode {
			fmt.Println("No command specified. Available commands: create, list, rollback, delete, start, stop")
			fmt.Println("Use --help for detailed usage information.")
			os.Exit(1)
		} else {
			// Interactive mode
			runInteractiveMode()
		}
	},
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create VM snapshots",
	Long: `Create snapshots for one or more VMs with intelligent naming and optional VM state.

Examples:
  # Create snapshot with prefix for single VM
  proxmox-snapshot-manager create --vmid 7303 --prefix backup
  
  # Create snapshot with VM state (RAM)
  proxmox-snapshot-manager create --vmname web01 --prefix backup --vmstate
  
  # Create snapshots for multiple VMs
  proxmox-snapshot-manager create --vmid 7301,7302,7303 --prefix pre-update --batch -y
  
  # Create with exact snapshot name
  proxmox-snapshot-manager create --vmid 7303 --name backup-20240101-1200`,
	RunE: runCreateCommand,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List VM snapshots",
	Long: `List snapshots for one or more VMs with detailed information.

Examples:
  # List snapshots for single VM
  proxmox-snapshot-manager list --vmid 7303
  
  # List snapshots for multiple VMs
  proxmox-snapshot-manager list --vmname web01,web02`,
	RunE: runListCommand,
}

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback VMs to snapshots",
	Long: `Rollback one or more VMs to a specific snapshot.

This operation will revert all changes made after the snapshot was created.

Examples:
  # Rollback single VM
  proxmox-snapshot-manager rollback --vmid 7303 --snapshot backup-20240101-1200
  
  # Rollback multiple VMs
  proxmox-snapshot-manager rollback --vmid 7301,7302 --snapshot pre-update --batch -y`,
	RunE: runRollbackCommand,
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete VM snapshots",
	Long: `Delete one or more snapshots from VMs.

Examples:
  # Delete specific snapshot
  proxmox-snapshot-manager delete --vmid 7303 --snapshot backup-20240101-1200
  
  # Delete all snapshots from VM
  proxmox-snapshot-manager delete --vmid 7303 --all --batch -y
  
  # Delete snapshots from multiple VMs
  proxmox-snapshot-manager delete --vmid 7301,7302 --snapshot pre-update --batch -y`,
	RunE: runDeleteCommand,
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start VMs",
	Long: `Start one or more virtual machines.

Examples:
  # Start single VM
  proxmox-snapshot-manager start --vmid 7303
  
  # Start multiple VMs
  proxmox-snapshot-manager start --vmid 7301,7302,7303 --batch`,
	RunE: runStartCommand,
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop VMs",
	Long: `Stop one or more virtual machines.

Examples:
  # Stop single VM
  proxmox-snapshot-manager stop --vmid 7303

  # Stop multiple VMs
  proxmox-snapshot-manager stop --vmid 7301,7302,7303 --batch -y`,
	RunE: runStopCommand,
}

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create VM backup",
	Long: `Create a backup of specified VM(s) to storage.

Examples:
  # Create backup with snapshot mode
  proxmox-snapshot-manager backup --vmid 7303 --storage local-zfs

  # Create backup with suspend mode
  proxmox-snapshot-manager backup --vmid 7303 --storage local-zfs --mode suspend

  # Create backup with specific compression
  proxmox-snapshot-manager backup --vmid 7303 --storage local-zfs --compress gzip`,
	RunE: runBackupCommand,
}

// listBackupsCmd represents the list-backups command
var listBackupsCmd = &cobra.Command{
	Use:   "list-backups",
	Short: "List VM backups",
	Long: `List all backups for specified VM(s).

Examples:
  # List backups for single VM
  proxmox-snapshot-manager list-backups --vmid 7303

  # List backups from specific storage
  proxmox-snapshot-manager list-backups --vmid 7303 --storage local-zfs`,
	RunE: runListBackupsCommand,
}

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore VM from backup",
	Long: `Restore a VM from a backup file.

Examples:
  # Restore from backup
  proxmox-snapshot-manager restore --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --node pve`,
	RunE: runRestoreCommand,
}

// deleteBackupsCmd represents the delete-backups command
var deleteBackupsCmd = &cobra.Command{
	Use:   "delete-backups",
	Short: "Delete VM backups",
	Long: `Delete specific backup(s) or cleanup old backups.

Examples:
  # Delete specific backup
  proxmox-snapshot-manager delete-backups --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --yes

  # Delete backups matching pattern
  proxmox-snapshot-manager delete-backups --vmid 7303 --pattern "*2024*" --yes

  # Keep only 5 most recent backups
  proxmox-snapshot-manager delete-backups --vmid 7303 --keep-count 5 --yes

  # Delete backups older than 30 days
  proxmox-snapshot-manager delete-backups --vmid 7303 --max-age-days 30 --yes`,
	RunE: runDeleteBackupsCommand,
}

// shutdownCmd represents the shutdown command
var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Gracefully shutdown VM(s)",
	Long: `Send ACPI shutdown signal to VM(s).

Examples:
  # Shutdown single VM
  proxmox-snapshot-manager shutdown --vmid 7303

  # Shutdown multiple VMs
  proxmox-snapshot-manager shutdown --vmid 7301,7302,7303 --yes`,
	RunE: runShutdownCommand,
}

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

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&batchMode, "batch", false, "batch mode - no interactive prompts")
	rootCmd.PersistentFlags().BoolVarP(&autoConfirm, "yes", "y", false, "auto-confirm operations")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without actually doing it")

	// Create command flags
	createCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	createCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")
	createCmd.Flags().String("prefix", "", "snapshot prefix")
	createCmd.Flags().String("name", "", "exact snapshot name")
	createCmd.Flags().Bool("vmstate", false, "include VM state (RAM)")

	// List command flags
	listCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	listCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")

	// Rollback command flags
	rollbackCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	rollbackCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")
	rollbackCmd.Flags().String("snapshot", "", "snapshot name to rollback to")
	rollbackCmd.MarkFlagRequired("snapshot")

	// Delete command flags
	deleteCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	deleteCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")
	deleteCmd.Flags().String("snapshot", "", "snapshot name to delete")
	deleteCmd.Flags().Bool("all", false, "delete all snapshots")

	// Start command flags
	startCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	startCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")

	// Stop command flags
	stopCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	stopCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")

	// Backup command flags
	backupCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	backupCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")
	backupCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage for backup (required)")
	backupCmd.Flags().StringVar(&modeFlag, "mode", "snapshot", "Backup mode: snapshot, suspend, or stop")
	backupCmd.Flags().StringVar(&compressFlag, "compress", "zstd", "Compression: zstd, gzip, or lzo")
	backupCmd.MarkFlagRequired("storage")

	// List-backups command flags
	listBackupsCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	listBackupsCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")
	listBackupsCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage to check (optional, checks all if not specified)")

	// Restore command flags
	restoreCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated, typically one)")
	restoreCmd.Flags().StringVar(&backupFileFlag, "backup-file", "", "Backup volid (required)")
	restoreCmd.Flags().StringVar(&nodeFlag, "node", "", "Node name (required)")
	restoreCmd.Flags().StringVar(&storageFlag, "storage", "", "Target storage (optional)")
	restoreCmd.MarkFlagRequired("vmid")
	restoreCmd.MarkFlagRequired("backup-file")
	restoreCmd.MarkFlagRequired("node")

	// Delete-backups command flags
	deleteBackupsCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated, typically one)")
	deleteBackupsCmd.Flags().StringVar(&backupFileFlag, "backup-file", "", "Specific backup volid to delete")
	deleteBackupsCmd.Flags().StringVar(&patternFlag, "pattern", "", "Delete backups matching pattern (e.g., '*2024*')")
	deleteBackupsCmd.Flags().IntVar(&keepCountFlag, "keep-count", 0, "Keep only N most recent backups")
	deleteBackupsCmd.Flags().IntVar(&maxAgeDaysFlag, "max-age-days", 0, "Delete backups older than N days")
	deleteBackupsCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage to check")
	deleteBackupsCmd.MarkFlagRequired("vmid")

	// Shutdown command flags
	shutdownCmd.Flags().StringSlice("vmid", []string{}, "VM IDs (comma-separated)")
	shutdownCmd.Flags().StringSlice("vmname", []string{}, "VM names (comma-separated)")

	// Quick operation command flags
	quickBackupAllCmd.Flags().StringVar(&storageFlag, "storage", "", "Storage for backup (required)")
	quickBackupAllCmd.Flags().StringVar(&modeFlag, "mode", "snapshot", "Backup mode: snapshot, suspend, or stop")
	quickBackupAllCmd.Flags().StringVar(&compressFlag, "compress", "zstd", "Compression: zstd, gzip, or lzo")
	quickBackupAllCmd.MarkFlagRequired("storage")

	// Add commands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(listBackupsCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(deleteBackupsCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(quickStartAllCmd)
	rootCmd.AddCommand(quickStopAllCmd)
	rootCmd.AddCommand(quickBackupAllCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// initializeApp initializes the application
func initializeApp(cmd *cobra.Command, args []string) error {
	var err error

	// Load configuration
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with command-line flags
	if batchMode {
		cfg.CLI.BatchMode = true
	}
	if autoConfirm {
		cfg.CLI.AutoConfirm = true
	}
	if verbose {
		cfg.Logging.Level = "debug"
	}
	if quiet {
		cfg.Logging.Level = "error"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Setup logger
	logger = cfg.SetupLogger()

	// Check for required environment variables in batch mode
	if cfg.IsBatchMode() {
		requiredVars := []string{"PVE_HOST", "PVE_USER"}
		if cfg.Proxmox.TokenName == "" || cfg.Proxmox.TokenValue == "" {
			requiredVars = append(requiredVars, "PVE_TOKEN_NAME", "PVE_TOKEN_VALUE")
		}

		var missingVars []string
		for _, envVar := range requiredVars {
			if os.Getenv(envVar) == "" {
				missingVars = append(missingVars, envVar)
			}
		}

		if len(missingVars) > 0 {
			return fmt.Errorf("batch mode: missing required environment variables: %s", strings.Join(missingVars, ", "))
		}
	}

	// Initialize API client
	clientConfig := &api.ClientConfig{
		Host:       cfg.Proxmox.Host,
		Port:       cfg.Proxmox.Port,
		Username:   cfg.Proxmox.Username,
		Password:   cfg.Proxmox.Password,
		TokenName:  cfg.Proxmox.TokenName,
		TokenValue: cfg.Proxmox.TokenValue,
		VerifySSL:  cfg.Proxmox.VerifySSL,
		Timeout:    cfg.Proxmox.Timeout,
		Logger:     logger,
	}

	client = api.NewClient(clientConfig)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to Proxmox API: %w", err)
	}

	if !cfg.IsBatchMode() {
		logger.Info("✅ Connected to Proxmox API successfully")
	}

	// Initialize components
	vmOps = vm.NewOperations(client, logger)
	vmSelector = vm.NewSelector(vmOps, logger)
	snapOps = snapshot.NewOperations(client, vmOps, logger)
	bulkMgr = bulk.NewManager(vmOps, snapOps, logger)

	// Configure bulk manager
	bulkMgr.SetMaxWorkers(cfg.GetMaxConcurrentOperations("snapshot"))

	return nil
}

// resolveVMs resolves VM identifiers to VM objects
func resolveVMs(vmids, vmnames []string) ([]*vm.VM, error) {
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs: %w", err)
	}

	var selectedVMs []*vm.VM
	vmSet := make(map[string]*vm.VM)

	// Process VM IDs
	for _, vmid := range vmids {
		resolvedID := vmSelector.FindVMByNameOrID(vmid, allVMs)
		if resolvedID == "" {
			return nil, fmt.Errorf("VM '%s' not found", vmid)
		}

		for _, vmInstance := range allVMs {
			if vmInstance.VMID == resolvedID {
				vmSet[resolvedID] = vmInstance
				break
			}
		}
	}

	// Process VM names
	for _, vmname := range vmnames {
		resolvedID := vmSelector.FindVMByNameOrID(vmname, allVMs)
		if resolvedID == "" {
			return nil, fmt.Errorf("VM '%s' not found", vmname)
		}

		for _, vmInstance := range allVMs {
			if vmInstance.VMID == resolvedID {
				vmSet[resolvedID] = vmInstance
				break
			}
		}
	}

	// Convert map to slice
	for _, vmInstance := range vmSet {
		selectedVMs = append(selectedVMs, vmInstance)
	}

	return selectedVMs, nil
}

// confirmOperation asks for user confirmation unless auto-confirm is enabled
func confirmOperation(message string) bool {
	if cfg.IsAutoConfirm() {
		return true
	}

	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

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

// Command handlers

func runCreateCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")
	prefix, _ := cmd.Flags().GetString("prefix")
	name, _ := cmd.Flags().GetString("name")
	vmstate, _ := cmd.Flags().GetBool("vmstate")

	// Validate arguments
	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	if prefix == "" && name == "" {
		return fmt.Errorf("either --prefix or --name must be specified")
	}

	if prefix != "" && name != "" {
		return fmt.Errorf("cannot specify both --prefix and --name")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	// Determine naming
	useExactName := name != ""
	nameOrPrefix := name
	if !useExactName {
		nameOrPrefix = prefix
	}

	logger.Infof("Creating snapshots for %d VM(s) with %s '%s' (vmstate: %v)",
		len(vms),
		map[bool]string{true: "name", false: "prefix"}[useExactName],
		nameOrPrefix,
		vmstate)

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			vmstateStr := ""
			if vmstate {
				vmstateStr = "with VM state"
			}
			printDryRunAction("create snapshot", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("name='%s' %s", nameOrPrefix, vmstateStr))
		}
		printDryRunSummary("create snapshot", len(vms))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Create snapshots for %d VM(s)?", len(vms))) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	go handleInterrupts(cancel, bulkMgr)

	// Execute bulk operation
	if err := bulkMgr.CreateSnapshots(ctx, vms, nameOrPrefix, useExactName, vmstate); err != nil {
		return fmt.Errorf("bulk snapshot creation failed: %w", err)
	}

	// Print summary
	bulkMgr.PrintSummary()
	return nil
}

func runListCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	// List snapshots for each VM
	for i, vmInstance := range vms {
		if err := snapOps.ListSnapshots(vmInstance.VMID); err != nil {
			logger.Errorf("Failed to list snapshots for VM %s: %v", vmInstance.VMID, err)
			continue
		}

		if i < len(vms)-1 {
			fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
		}
	}

	return nil
}

func runRollbackCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")
	snapshotName, _ := cmd.Flags().GetString("snapshot")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	logger.Infof("Rolling back %d VM(s) to snapshot '%s'", len(vms), snapshotName)

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("rollback", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("to snapshot '%s'", snapshotName))
		}
		printDryRunSummary("rollback", len(vms))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Rollback %d VM(s) to snapshot '%s'? This will revert all changes after the snapshot.", len(vms), snapshotName)) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	// Execute bulk operation
	if err := bulkMgr.RollbackSnapshots(ctx, vms, snapshotName); err != nil {
		return fmt.Errorf("bulk snapshot rollback failed: %w", err)
	}

	bulkMgr.PrintSummary()
	return nil
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")
	snapshotName, _ := cmd.Flags().GetString("snapshot")
	deleteAll, _ := cmd.Flags().GetBool("all")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	if snapshotName == "" && !deleteAll {
		return fmt.Errorf("either --snapshot or --all must be specified")
	}

	if snapshotName != "" && deleteAll {
		return fmt.Errorf("cannot specify both --snapshot and --all")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			if deleteAll {
				printDryRunAction("delete all snapshots", vmInstance.VMID, vmInstance.Name, "")
			} else {
				printDryRunAction("delete snapshot", vmInstance.VMID, vmInstance.Name,
					fmt.Sprintf("snapshot '%s'", snapshotName))
			}
		}
		operation := "delete all snapshots"
		if !deleteAll {
			operation = fmt.Sprintf("delete snapshot '%s'", snapshotName)
		}
		printDryRunSummary(operation, len(vms))
		return nil
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			if deleteAll {
				printDryRunAction("delete all snapshots", vmInstance.VMID, vmInstance.Name, "")
			} else {
				printDryRunAction("delete snapshot", vmInstance.VMID, vmInstance.Name,
					fmt.Sprintf("snapshot '%s'", snapshotName))
			}
		}
		operation := "delete all snapshots"
		if !deleteAll {
			operation = fmt.Sprintf("delete snapshot '%s'", snapshotName)
		}
		printDryRunSummary(operation, len(vms))
		return nil
	}

	var confirmMsg string
	if deleteAll {
		confirmMsg = fmt.Sprintf("Delete ALL snapshots from %d VM(s)? This cannot be undone.", len(vms))
		if !cfg.IsAutoConfirm() {
			fmt.Print("Type 'DELETE ALL' to confirm: ")
			var response string
			fmt.Scanln(&response)
			if response != "DELETE ALL" {
				fmt.Println("Operation cancelled")
				return nil
			}
		}
	} else {
		confirmMsg = fmt.Sprintf("Delete snapshot '%s' from %d VM(s)?", snapshotName, len(vms))
		if !confirmOperation(confirmMsg) {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	// Execute operation
	if deleteAll {
		// Delete all snapshots from each VM
		for _, vmInstance := range vms {
			if err := snapOps.DeleteAllSnapshots(vmInstance.VMID); err != nil {
				logger.Errorf("Failed to delete all snapshots from VM %s: %v", vmInstance.VMID, err)
			}
		}
	} else {
		// Delete specific snapshot
		if err := bulkMgr.DeleteSnapshots(ctx, vms, snapshotName); err != nil {
			return fmt.Errorf("bulk snapshot deletion failed: %w", err)
		}
		bulkMgr.PrintSummary()
	}

	return nil
}

func runStartCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	logger.Infof("Starting %d VM(s)", len(vms))

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("start", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("start", len(vms))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Start %d VM(s)?", len(vms))) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	// Execute bulk operation
	if err := bulkMgr.StartVMs(ctx, vms); err != nil {
		return fmt.Errorf("bulk VM start failed: %w", err)
	}

	bulkMgr.PrintSummary()
	return nil
}

func runStopCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	logger.Infof("Stopping %d VM(s)", len(vms))

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("stop", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("stop", len(vms))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Stop %d VM(s)?", len(vms))) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	// Execute bulk operation
	if err := bulkMgr.StopVMs(ctx, vms); err != nil {
		return fmt.Errorf("bulk VM stop failed: %w", err)
	}

	bulkMgr.PrintSummary()
	return nil
}

func runBackupCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	// Resolve VMs
	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	// Validate storage (skip in dry-run)
	if !dryRun {
		storageOps := storage.NewOperations(client, logger)
		if err := storageOps.ValidateStorage(storageFlag); err != nil {
			return fmt.Errorf("storage validation failed: %w", err)
		}
	}

	// Initialize backup operations
	backupOps := backup.NewOperations(client, vmOps, logger)

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
		printDryRunHeader()
		for _, vmInstance := range vms {
			details := fmt.Sprintf("storage=%s, mode=%s, compress=%s", storageFlag, modeFlag, compressFlag)
			printDryRunAction("backup", vmInstance.VMID, vmInstance.Name, details)
		}
		printDryRunSummary("backup", len(vms))
		return nil
	}

	// Confirm operation
	if !confirmOperation(fmt.Sprintf("Create backup for %d VM(s) with mode '%s'?", len(vms), modeFlag)) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Single or bulk operation
	if len(vms) == 1 {
		return backupOps.CreateBackup(vms[0].VMID, storageFlag, mode, compressFlag)
	}

	// Bulk backup
	fmt.Printf("\nCreating backups for %d VMs\n", len(vms))
	for _, vmInstance := range vms {
		if err := backupOps.CreateBackup(vmInstance.VMID, storageFlag, mode, compressFlag); err != nil {
			logger.Errorf("Failed to backup VM %s: %v", vmInstance.VMID, err)
		}
	}
	return nil
}

func runListBackupsCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	backupOps := backup.NewOperations(client, vmOps, logger)

	for _, vmInstance := range vms {
		if err := backupOps.DisplayBackups(vmInstance.VMID, storageFlag); err != nil {
			logger.Errorf("Failed to list backups for VM %s: %v", vmInstance.VMID, err)
		}
	}
	return nil
}

func runRestoreCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")

	if len(vmids) == 0 {
		return fmt.Errorf("--vmid must be specified")
	}

	// For restore, we typically work with a single VM
	vmid := vmids[0]

	// Protection check
	protectionOps := protection.NewOperations(client, vmOps, logger)
	protectionOps.CheckAndWarn(vmid, "restore")

	// Confirm if not auto-confirm
	if !confirmOperation(fmt.Sprintf("Restore VM %s from %s? This will OVERWRITE the existing VM!", vmid, backupFileFlag)) {
		fmt.Println("Restore cancelled")
		return nil
	}

	backupOps := backup.NewOperations(client, vmOps, logger)
	return backupOps.RestoreBackup(vmid, backupFileFlag, nodeFlag, storageFlag)
}

func runDeleteBackupsCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")

	if len(vmids) == 0 {
		return fmt.Errorf("--vmid must be specified")
	}

	// For delete, we typically work with a single VM
	vmid := vmids[0]

	backupOps := backup.NewOperations(client, vmOps, logger)

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		
		// Specific backup deletion
		if backupFileFlag != "" {
			printDryRunAction("delete backup", vmid, "", fmt.Sprintf("backup='%s'", backupFileFlag))
			printDryRunSummary(fmt.Sprintf("delete backup '%s'", backupFileFlag), 1)
			return nil
		}

		// Pattern-based deletion
		if patternFlag != "" {
			printDryRunAction("delete backups", vmid, "", fmt.Sprintf("matching pattern='%s'", patternFlag))
			printDryRunSummary(fmt.Sprintf("delete backups matching pattern '%s'", patternFlag), 1)
			return nil
		}

		// Retention-based cleanup
		if keepCountFlag > 0 || maxAgeDaysFlag > 0 {
			printDryRunAction("cleanup backups", vmid, "", 
				fmt.Sprintf("keep=%d, max-age=%d days", keepCountFlag, maxAgeDaysFlag))
			printDryRunSummary(fmt.Sprintf("cleanup backups (keep=%d, max-age=%d days)", keepCountFlag, maxAgeDaysFlag), 1)
			return nil
		}

		return fmt.Errorf("must specify --backup-file, --pattern, --keep-count, or --max-age-days")
	}

	// Specific backup deletion
	if backupFileFlag != "" {
		if !confirmOperation(fmt.Sprintf("Delete backup %s?", backupFileFlag)) {
			fmt.Println("Deletion cancelled")
			return nil
		}

		backups, err := backupOps.ListBackupsForVM(vmid, storageFlag)
		if err != nil {
			return err
		}

		for _, bkp := range backups {
			if bkp.VolID == backupFileFlag {
				return backupOps.DeleteBackup(bkp)
			}
		}
		return fmt.Errorf("backup not found: %s", backupFileFlag)
	}

	// Pattern-based deletion
	if patternFlag != "" {
		if !confirmOperation(fmt.Sprintf("Delete backups matching pattern '%s'?", patternFlag)) {
			fmt.Println("Deletion cancelled")
			return nil
		}
		deleted, err := backupOps.DeleteBackupsByPattern(vmid, storageFlag, patternFlag)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Deleted %d backup(s)\n", deleted)
		return nil
	}

	// Retention-based cleanup
	if keepCountFlag > 0 || maxAgeDaysFlag > 0 {
		if !confirmOperation(fmt.Sprintf("Cleanup old backups (keep=%d, max-age=%d days)?", keepCountFlag, maxAgeDaysFlag)) {
			fmt.Println("Cleanup cancelled")
			return nil
		}
		deleted, err := backupOps.DeleteOldBackups(vmid, storageFlag, keepCountFlag, maxAgeDaysFlag)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Cleaned up %d backup(s)\n", deleted)
		return nil
	}

	return fmt.Errorf("must specify --backup-file, --pattern, --keep-count, or --max-age-days")
}

func runShutdownCommand(cmd *cobra.Command, args []string) error {
	vmids, _ := cmd.Flags().GetStringSlice("vmid")
	vmnames, _ := cmd.Flags().GetStringSlice("vmname")

	if len(vmids) == 0 && len(vmnames) == 0 {
		return fmt.Errorf("either --vmid or --vmname must be specified")
	}

	vms, err := resolveVMs(vmids, vmnames)
	if err != nil {
		return err
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("gracefully shutdown", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("gracefully shutdown", len(vms))
		return nil
	}

	if !confirmOperation(fmt.Sprintf("Gracefully shutdown %d VM(s)?", len(vms))) {
		fmt.Println("Shutdown cancelled")
		return nil
	}

	for _, vmInstance := range vms {
		if err := vmOps.ShutdownVM(vmInstance.VMID); err != nil {
			logger.Errorf("Failed to shutdown VM %s: %v", vmInstance.VMID, err)
		} else {
			fmt.Printf("✅ VM %s shutdown initiated\n", vmInstance.VMID)
		}
	}
	return nil
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

func runQuickBackupAllCommand(cmd *cobra.Command, args []string) error {
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

func runInteractiveMode() {
	fmt.Println("🚀 Proxmox Snapshot Manager - Interactive Mode")
	fmt.Println("============================================")

	for {
		fmt.Println("\nAvailable operations:")
		fmt.Println("1.  Create snapshots")
		fmt.Println("2.  List snapshots")
		fmt.Println("3.  Rollback snapshots")
		fmt.Println("4.  Delete snapshots")
		fmt.Println("5.  Start VMs")
		fmt.Println("6.  Stop VMs")
		fmt.Println("7.  Gracefully shutdown VMs")
		fmt.Println("8.  Create VM backups")
		fmt.Println("9.  List VM backups")
		fmt.Println("10. Restore VM from backup")
		fmt.Println("11. Delete VM backups")
		fmt.Println("12. Quick start all stopped VMs")
		fmt.Println("13. Quick stop all running VMs")
		fmt.Println("14. Quick backup all VMs")
		fmt.Println("0.  Exit")

		fmt.Print("\nSelect operation (0-14): ")
		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 0:
			fmt.Println("Goodbye!")
			return
		case 1:
			runInteractiveCreate()
		case 2:
			runInteractiveList()
		case 3:
			runInteractiveRollback()
		case 4:
			runInteractiveDelete()
		case 5:
			runInteractiveStart()
		case 6:
			runInteractiveStop()
		case 7:
			runInteractiveShutdown()
		case 8:
			runInteractiveBackup()
		case 9:
			runInteractiveListBackups()
		case 10:
			runInteractiveRestore()
		case 11:
			runInteractiveDeleteBackups()
		case 12:
			runInteractiveQuickStartAll()
		case 13:
			runInteractiveQuickStopAll()
		case 14:
			runInteractiveQuickBackupAll()
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

func runInteractiveCreate() {
	fmt.Println("\n📸 Create Snapshots")
	fmt.Println("==================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs for snapshot creation:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	fmt.Print("Enter snapshot prefix: ")
	var prefix string
	fmt.Scanln(&prefix)

	fmt.Print("Include VM state/RAM? (y/N): ")
	var vmstateInput string
	fmt.Scanln(&vmstateInput)
	vmstate := strings.ToLower(vmstateInput) == "y" || strings.ToLower(vmstateInput) == "yes"

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			vmstateStr := ""
			if vmstate {
				vmstateStr = "with VM state"
			}
			printDryRunAction("create snapshot", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("prefix='%s' %s", prefix, vmstateStr))
		}
		printDryRunSummary("create snapshot", len(vms))
		return
	}

	// Execute operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if err := bulkMgr.CreateSnapshots(ctx, vms, prefix, false, vmstate); err != nil {
		logger.Errorf("Snapshot creation failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveList() {
	fmt.Println("\n📋 List Snapshots")
	fmt.Println("=================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs to list snapshots:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	for i, vmInstance := range vms {
		if err := snapOps.ListSnapshots(vmInstance.VMID); err != nil {
			logger.Errorf("Failed to list snapshots for VM %s: %v", vmInstance.VMID, err)
		}

		if i < len(vms)-1 {
			fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
		}
	}
}

func runInteractiveRollback() {
	fmt.Println("\n⏪ Rollback Snapshots")
	fmt.Println("====================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs to rollback:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	fmt.Print("Enter snapshot name to rollback to: ")
	var snapshotName string
	fmt.Scanln(&snapshotName)

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("rollback", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("to snapshot '%s'", snapshotName))
		}
		printDryRunSummary("rollback", len(vms))
		return
	}

	// Execute operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if err := bulkMgr.RollbackSnapshots(ctx, vms, snapshotName); err != nil {
		logger.Errorf("Snapshot rollback failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveDelete() {
	fmt.Println("\n🗑️ Delete Snapshots")
	fmt.Println("===================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs to delete snapshots from:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	fmt.Print("Enter snapshot name to delete (or 'ALL' to delete all snapshots): ")
	var snapshotName string
	fmt.Scanln(&snapshotName)

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		deleteAll := strings.ToUpper(snapshotName) == "ALL"
		for _, vmInstance := range vms {
			if deleteAll {
				printDryRunAction("delete all snapshots", vmInstance.VMID, vmInstance.Name, "")
			} else {
				printDryRunAction("delete snapshot", vmInstance.VMID, vmInstance.Name,
					fmt.Sprintf("snapshot '%s'", snapshotName))
			}
		}
		operation := "delete all snapshots"
		if !deleteAll {
			operation = fmt.Sprintf("delete snapshot '%s'", snapshotName)
		}
		printDryRunSummary(operation, len(vms))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if strings.ToUpper(snapshotName) == "ALL" {
		for _, vmInstance := range vms {
			if err := snapOps.DeleteAllSnapshots(vmInstance.VMID); err != nil {
				logger.Errorf("Failed to delete all snapshots from VM %s: %v", vmInstance.VMID, err)
			}
		}
	} else {
		if err := bulkMgr.DeleteSnapshots(ctx, vms, snapshotName); err != nil {
			logger.Errorf("Snapshot deletion failed: %v", err)
			return
		}
		bulkMgr.PrintSummary()
	}
}

func runInteractiveStart() {
	fmt.Println("\n▶️ Start VMs")
	fmt.Println("============")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	// Filter to stopped VMs
	var stoppedVMs []*vm.VM
	for _, vmInstance := range allVMs {
		if !vmInstance.Running {
			stoppedVMs = append(stoppedVMs, vmInstance)
		}
	}

	if len(stoppedVMs) == 0 {
		fmt.Println("No stopped VMs found.")
		return
	}

	vms, err := vmSelector.InteractiveSelect(stoppedVMs, "Select VMs to start:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("start", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("start", len(vms))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if err := bulkMgr.StartVMs(ctx, vms); err != nil {
		logger.Errorf("VM start failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveStop() {
	fmt.Println("\n⏹️ Stop VMs")
	fmt.Println("===========")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	// Filter to running VMs
	var runningVMs []*vm.VM
	for _, vmInstance := range allVMs {
		if vmInstance.Running {
			runningVMs = append(runningVMs, vmInstance)
		}
	}

	if len(runningVMs) == 0 {
		fmt.Println("No running VMs found.")
		return
	}

	vms, err := vmSelector.InteractiveSelect(runningVMs, "Select VMs to stop:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("stop", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("stop", len(vms))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if err := bulkMgr.StopVMs(ctx, vms); err != nil {
		logger.Errorf("VM stop failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveShutdown() {
	fmt.Println("\n🔌 Gracefully Shutdown VMs")
	fmt.Println("=========================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	// Filter to running VMs
	var runningVMs []*vm.VM
	for _, vmInstance := range allVMs {
		if vmInstance.Running {
			runningVMs = append(runningVMs, vmInstance)
		}
	}

	if len(runningVMs) == 0 {
		fmt.Println("No running VMs found.")
		return
	}

	vms, err := vmSelector.InteractiveSelect(runningVMs, "Select VMs to gracefully shutdown:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			printDryRunAction("gracefully shutdown", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("gracefully shutdown", len(vms))
		return
	}

	// Confirm operation
	fmt.Printf("\n⚠️  Will gracefully shutdown %d VM(s):\n", len(vms))
	for _, vmInstance := range vms {
		fmt.Printf("  • VM %s (%s)\n", vmInstance.VMID, vmInstance.Name)
	}
	fmt.Println()

	fmt.Print("Continue with graceful shutdown? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Shutdown cancelled")
		return
	}

	// Execute shutdown
	for _, vmInstance := range vms {
		if err := vmOps.ShutdownVM(vmInstance.VMID); err != nil {
			logger.Errorf("Failed to shutdown VM %s: %v", vmInstance.VMID, err)
		} else {
			fmt.Printf("✅ VM %s shutdown initiated\n", vmInstance.VMID)
		}
	}
}

func runInteractiveBackup() {
	fmt.Println("\n💾 Create VM Backups")
	fmt.Println("====================")

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs to backup:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	// Get storage
	fmt.Print("Enter storage name (e.g., local-zfs): ")
	var storage string
	fmt.Scanln(&storage)
	if storage == "" {
		fmt.Println("Storage name is required")
		return
	}

	// Get mode
	fmt.Print("Enter backup mode (snapshot/suspend/stop) [snapshot]: ")
	var mode string
	fmt.Scanln(&mode)
	if mode == "" {
		mode = "snapshot"
	}

	// Get compression
	fmt.Print("Enter compression (zstd/gzip/lzo) [zstd]: ")
	var compress string
	fmt.Scanln(&compress)
	if compress == "" {
		compress = "zstd"
	}

	// Validate storage (skip in dry-run)
	if !dryRun {
		storageOps := storage.NewOperations(client, logger)
		if err := storageOps.ValidateStorage(storage); err != nil {
			logger.Errorf("Storage validation failed: %v", err)
			return
		}
	}

	// Initialize backup operations
	backupOps := backup.NewOperations(client, vmOps, logger)

	// Validate mode
	var backupMode backup.BackupMode
	switch mode {
	case "snapshot":
		backupMode = backup.BackupModeSnapshot
	case "suspend":
		backupMode = backup.BackupModeSuspend
	case "stop":
		backupMode = backup.BackupModeStop
	default:
		logger.Errorf("Invalid mode: %s (must be snapshot, suspend, or stop)", mode)
		return
	}

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range vms {
			details := fmt.Sprintf("storage=%s, mode=%s, compress=%s", storage, mode, compress)
			printDryRunAction("backup", vmInstance.VMID, vmInstance.Name, details)
		}
		printDryRunSummary("backup", len(vms))
		return
	}

	// Confirm operation
	fmt.Printf("\n⚠️  Will backup %d VM(s) to storage '%s' with mode '%s':\n", len(vms), storage, mode)
	for _, vmInstance := range vms {
		fmt.Printf("  • VM %s (%s)\n", vmInstance.VMID, vmInstance.Name)
	}
	fmt.Println()

	fmt.Print("Continue with backup? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Backup cancelled")
		return
	}

	// Execute backups
	fmt.Printf("\nCreating backups for %d VMs...\n", len(vms))
	successCount := 0
	for _, vmInstance := range vms {
		fmt.Printf("\nBacking up VM %s (%s)...\n", vmInstance.VMID, vmInstance.Name)
		if err := backupOps.CreateBackup(vmInstance.VMID, storage, backupMode, compress); err != nil {
			logger.Errorf("Failed to backup VM %s: %v", vmInstance.VMID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✅ Successfully backed up %d/%d VMs\n", successCount, len(vms))
}

func runInteractiveListBackups() {
	fmt.Println("\n📋 List VM Backups")
	fmt.Println("==================")

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		fmt.Println("[DRY-RUN] Would list backups for selected VMs")
		printDryRunSummary("list backups", 1)
		return
	}

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VMs to list backups for:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	// Get optional storage filter
	fmt.Print("Enter storage name to filter (optional, press Enter for all): ")
	var storage string
	fmt.Scanln(&storage)

	backupOps := backup.NewOperations(client, vmOps, logger)

	for _, vmInstance := range vms {
		fmt.Printf("\n📦 Backups for VM %s (%s):\n", vmInstance.VMID, vmInstance.Name)
		if err := backupOps.DisplayBackups(vmInstance.VMID, storage); err != nil {
			logger.Errorf("Failed to list backups for VM %s: %v", vmInstance.VMID, err)
		}
		if len(vms) > 1 {
			fmt.Println("\n" + strings.Repeat("=", 60))
		}
	}
}

func runInteractiveRestore() {
	fmt.Println("\n🔄 Restore VM from Backup")
	fmt.Println("==========================")

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		allVMs, err := vmOps.GetAllVMs()
		if err != nil {
			logger.Errorf("Failed to get VMs: %v", err)
			return
		}

		vms, err := vmSelector.InteractiveSelect(allVMs, "Select VM to restore:")
		if err != nil {
			logger.Errorf("VM selection failed: %v", err)
			return
		}

		if len(vms) != 1 {
			fmt.Println("Please select exactly one VM for restore")
			return
		}

		vmInstance := vms[0]

		// Get backup file
		fmt.Print("\nEnter backup volid to restore (e.g., local:backup/vzdump-qemu-7303-2025_08_06.vma.zst): ")
		var backupFile string
		fmt.Scanln(&backupFile)
		if backupFile == "" {
			fmt.Println("Backup file is required")
			return
		}

		// Get node
		fmt.Print("Enter node name: ")
		var node string
		fmt.Scanln(&node)
		if node == "" {
			fmt.Println("Node name is required")
			return
		}

		// Get optional target storage
		fmt.Print("Enter target storage (optional, press Enter to use original): ")
		var targetStorage string
		fmt.Scanln(&targetStorage)

		printDryRunAction("restore", vmInstance.VMID, vmInstance.Name,
			fmt.Sprintf("from backup '%s' on node '%s'", backupFile, node))
		printDryRunSummary("restore", 1)
		return
	}

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VM to restore:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	if len(vms) != 1 {
		fmt.Println("Please select exactly one VM for restore")
		return
	}

	vmInstance := vms[0]

	// List available backups first
	backupOps := backup.NewOperations(client, vmOps, logger)
	fmt.Printf("\n📦 Available backups for VM %s (%s):\n", vmInstance.VMID, vmInstance.Name)
	if err := backupOps.DisplayBackups(vmInstance.VMID, ""); err != nil {
		logger.Errorf("Failed to list backups: %v", err)
		return
	}

	// Get backup file
	fmt.Print("\nEnter backup volid to restore (e.g., local:backup/vzdump-qemu-7303-2025_08_06.vma.zst): ")
	var backupFile string
	fmt.Scanln(&backupFile)
	if backupFile == "" {
		fmt.Println("Backup file is required")
		return
	}

	// Get node
	fmt.Print("Enter node name: ")
	var node string
	fmt.Scanln(&node)
	if node == "" {
		fmt.Println("Node name is required")
		return
	}

	// Get optional target storage
	fmt.Print("Enter target storage (optional, press Enter to use original): ")
	var targetStorage string
	fmt.Scanln(&targetStorage)

	// Protection check
	protectionOps := protection.NewOperations(client, vmOps, logger)
	protectionOps.CheckAndWarn(vmInstance.VMID, "restore")

	// Confirm operation
	fmt.Printf("\n⚠️  WARNING: This will OVERWRITE VM %s (%s) with backup %s\n",
		vmInstance.VMID, vmInstance.Name, backupFile)
	fmt.Print("Type 'RESTORE' to confirm: ")
	var response string
	fmt.Scanln(&response)
	if response != "RESTORE" {
		fmt.Println("Restore cancelled")
		return
	}

	// Execute restore
	if err := backupOps.RestoreBackup(vmInstance.VMID, backupFile, node, targetStorage); err != nil {
		logger.Errorf("Restore failed: %v", err)
		return
	}

	fmt.Printf("\n✅ VM %s restored successfully\n", vmInstance.VMID)
}

func runInteractiveDeleteBackups() {
	fmt.Println("\n🗑️ Delete VM Backups")
	fmt.Println("===================")

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		allVMs, err := vmOps.GetAllVMs()
		if err != nil {
			logger.Errorf("Failed to get VMs: %v", err)
			return
		}

		vms, err := vmSelector.InteractiveSelect(allVMs, "Select VM to delete backups for:")
		if err != nil {
			logger.Errorf("VM selection failed: %v", err)
			return
		}

		if len(vms) != 1 {
			fmt.Println("Please select exactly one VM for backup deletion")
			return
		}

		vmInstance := vms[0]

		fmt.Println("\nDelete options:")
		fmt.Println("1. Delete specific backup")
		fmt.Println("2. Delete backups matching pattern")
		fmt.Println("3. Keep only N most recent backups")
		fmt.Println("4. Delete backups older than N days")

		fmt.Print("\nSelect delete option (1-4): ")
		var option int
		fmt.Scanln(&option)

		switch option {
		case 1:
			fmt.Print("Enter backup volid to delete: ")
			var backupFile string
			fmt.Scanln(&backupFile)
			if backupFile == "" {
				fmt.Println("Backup file is required")
				return
			}
			printDryRunAction("delete backup", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("backup='%s'", backupFile))

		case 2:
			fmt.Print("Enter pattern (e.g., '*2024*'): ")
			var pattern string
			fmt.Scanln(&pattern)
			if pattern == "" {
				fmt.Println("Pattern is required")
				return
			}
			printDryRunAction("delete backups", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("matching pattern='%s'", pattern))

		case 3:
			fmt.Print("Enter number of most recent backups to keep: ")
			var keepCount int
			fmt.Scanln(&keepCount)
			if keepCount <= 0 {
				fmt.Println("Keep count must be positive")
				return
			}
			printDryRunAction("cleanup backups", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("keep=%d most recent", keepCount))

		case 4:
			fmt.Print("Enter maximum age in days: ")
			var maxAgeDays int
			fmt.Scanln(&maxAgeDays)
			if maxAgeDays <= 0 {
				fmt.Println("Max age must be positive")
				return
			}
			printDryRunAction("cleanup backups", vmInstance.VMID, vmInstance.Name,
				fmt.Sprintf("older than %d days", maxAgeDays))

		default:
			fmt.Println("Invalid option")
			return
		}

		printDryRunSummary("delete backups", 1)
		return
	}

	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	vms, err := vmSelector.InteractiveSelect(allVMs, "Select VM to delete backups for:")
	if err != nil {
		logger.Errorf("VM selection failed: %v", err)
		return
	}

	if len(vms) != 1 {
		fmt.Println("Please select exactly one VM for backup deletion")
		return
	}

	vmInstance := vms[0]

	// List available backups first
	backupOps := backup.NewOperations(client, vmOps, logger)
	fmt.Printf("\n📦 Available backups for VM %s (%s):\n", vmInstance.VMID, vmInstance.Name)
	if err := backupOps.DisplayBackups(vmInstance.VMID, ""); err != nil {
		logger.Errorf("Failed to list backups: %v", err)
		return
	}

	fmt.Println("\nDelete options:")
	fmt.Println("1. Delete specific backup")
	fmt.Println("2. Delete backups matching pattern")
	fmt.Println("3. Keep only N most recent backups")
	fmt.Println("4. Delete backups older than N days")

	fmt.Print("\nSelect delete option (1-4): ")
	var option int
	fmt.Scanln(&option)

	switch option {
	case 1:
		fmt.Print("Enter backup volid to delete: ")
		var backupFile string
		fmt.Scanln(&backupFile)
		if backupFile == "" {
			fmt.Println("Backup file is required")
			return
		}

		fmt.Printf("\n⚠️  Will delete backup: %s\n", backupFile)
		fmt.Print("Type 'DELETE' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "DELETE" {
			fmt.Println("Deletion cancelled")
			return
		}

		backups, err := backupOps.ListBackupsForVM(vmInstance.VMID, "")
		if err != nil {
			logger.Errorf("Failed to list backups: %v", err)
			return
		}

		for _, bkp := range backups {
			if bkp.VolID == backupFile {
				if err := backupOps.DeleteBackup(bkp); err != nil {
					logger.Errorf("Failed to delete backup: %v", err)
					return
				}
				fmt.Printf("\n✅ Backup %s deleted\n", backupFile)
				return
			}
		}
		logger.Errorf("Backup not found: %s", backupFile)

	case 2:
		fmt.Print("Enter pattern (e.g., '*2024*'): ")
		var pattern string
		fmt.Scanln(&pattern)
		if pattern == "" {
			fmt.Println("Pattern is required")
			return
		}

		fmt.Printf("\n⚠️  Will delete backups matching pattern: %s\n", pattern)
		fmt.Print("Type 'DELETE' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "DELETE" {
			fmt.Println("Deletion cancelled")
			return
		}

		deleted, err := backupOps.DeleteBackupsByPattern(vmInstance.VMID, "", pattern)
		if err != nil {
			logger.Errorf("Failed to delete backups: %v", err)
			return
		}
		fmt.Printf("\n✅ Deleted %d backup(s)\n", deleted)

	case 3:
		fmt.Print("Enter number of most recent backups to keep: ")
		var keepCount int
		fmt.Scanln(&keepCount)
		if keepCount <= 0 {
			fmt.Println("Keep count must be positive")
			return
		}

		fmt.Printf("\n⚠️  Will keep only %d most recent backups\n", keepCount)
		fmt.Print("Type 'CLEANUP' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "CLEANUP" {
			fmt.Println("Cleanup cancelled")
			return
		}

		deleted, err := backupOps.DeleteOldBackups(vmInstance.VMID, "", keepCount, 0)
		if err != nil {
			logger.Errorf("Failed to cleanup backups: %v", err)
			return
		}
		fmt.Printf("\n✅ Cleaned up %d backup(s)\n", deleted)

	case 4:
		fmt.Print("Enter maximum age in days: ")
		var maxAgeDays int
		fmt.Scanln(&maxAgeDays)
		if maxAgeDays <= 0 {
			fmt.Println("Max age must be positive")
			return
		}

		fmt.Printf("\n⚠️  Will delete backups older than %d days\n", maxAgeDays)
		fmt.Print("Type 'CLEANUP' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "CLEANUP" {
			fmt.Println("Cleanup cancelled")
			return
		}

		deleted, err := backupOps.DeleteOldBackups(vmInstance.VMID, "", 0, maxAgeDays)
		if err != nil {
			logger.Errorf("Failed to cleanup backups: %v", err)
			return
		}
		fmt.Printf("\n✅ Cleaned up %d backup(s)\n", deleted)

	default:
		fmt.Println("Invalid option")
	}
}

func runInteractiveQuickStartAll() {
	fmt.Println("\n🚀 Quick Start All Stopped VMs")
	fmt.Println("============================")

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
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
		return
	}

	// Display list
	fmt.Printf("\nFound %d stopped VM(s):\n", len(stoppedVMs))
	for _, vmInstance := range stoppedVMs {
		fmt.Printf("  • VM %s (%s) - stopped\n", vmInstance.VMID, vmInstance.Name)
	}
	fmt.Println()

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range stoppedVMs {
			printDryRunAction("start", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("start", len(stoppedVMs))
		return
	}

	// Confirm operation
	fmt.Print("Start all stopped VMs? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	// Execute bulk operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	logger.Infof("Starting %d VM(s)", len(stoppedVMs))
	if err := bulkMgr.StartVMs(ctx, stoppedVMs); err != nil {
		logger.Errorf("Bulk VM start failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveQuickStopAll() {
	fmt.Println("\n⏹️ Quick Stop All Running VMs")
	fmt.Println("============================")

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
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
		return
	}

	// Display list with warning
	fmt.Printf("\n⚠️  Found %d running VM(s):\n", len(runningVMs))
	for _, vmInstance := range runningVMs {
		fmt.Printf("  • VM %s (%s) - running\n", vmInstance.VMID, vmInstance.Name)
	}
	fmt.Println()

	// Dry-run mode - just show what would happen
	if dryRun {
		printDryRunHeader()
		for _, vmInstance := range runningVMs {
			printDryRunAction("force stop", vmInstance.VMID, vmInstance.Name, "")
		}
		printDryRunSummary("force stop", len(runningVMs))
		return
	}

	// Confirm operation with extra warning
	fmt.Println("⚠️  WARNING: This will FORCE STOP all running VMs (not graceful shutdown)")
	fmt.Print("Type 'FORCE STOP' to confirm: ")
	var response string
	fmt.Scanln(&response)
	if response != "FORCE STOP" {
		fmt.Println("Operation cancelled")
		return
	}

	// Execute bulk operation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	logger.Infof("Stopping %d VM(s)", len(runningVMs))
	if err := bulkMgr.StopVMs(ctx, runningVMs); err != nil {
		logger.Errorf("Bulk VM stop failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
}

func runInteractiveQuickBackupAll() {
	fmt.Println("\n💾 Quick Backup All VMs")
	fmt.Println("=======================")

	// Get all VMs
	allVMs, err := vmOps.GetAllVMs()
	if err != nil {
		logger.Errorf("Failed to get VMs: %v", err)
		return
	}

	if len(allVMs) == 0 {
		fmt.Println("ℹ️  No VMs found")
		return
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

	// Get storage
	fmt.Print("Enter storage name (e.g., local-zfs): ")
	var storage string
	fmt.Scanln(&storage)
	if storage == "" {
		fmt.Println("Storage name is required")
		return
	}

	// Get mode
	fmt.Print("Enter backup mode (snapshot/suspend/stop) [snapshot]: ")
	var mode string
	fmt.Scanln(&mode)
	if mode == "" {
		mode = "snapshot"
	}

	// Get compression
	fmt.Print("Enter compression (zstd/gzip/lzo) [zstd]: ")
	var compress string
	fmt.Scanln(&compress)
	if compress == "" {
		compress = "zstd"
	}

	// Validate storage
	storageOps := storage.NewOperations(client, logger)
	if err := storageOps.ValidateStorage(storage); err != nil {
		logger.Errorf("Storage validation failed: %v", err)
		return
	}

	// Validate mode
	var backupMode backup.BackupMode
	switch mode {
	case "snapshot":
		backupMode = backup.BackupModeSnapshot
	case "suspend":
		backupMode = backup.BackupModeSuspend
	case "stop":
		backupMode = backup.BackupModeStop
	default:
		logger.Errorf("Invalid mode: %s (must be snapshot, suspend, or stop)", mode)
		return
	}

	// Confirm operation
	confirmMsg := fmt.Sprintf("Backup %d VM(s) to storage '%s' with mode '%s'?",
		len(allVMs), storage, mode)
	fmt.Print(confirmMsg + " (y/N): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	// Execute backups
	backupOps := backup.NewOperations(client, vmOps, logger)
	logger.Infof("Creating backups for %d VM(s)", len(allVMs))

	successCount := 0
	for _, vmInstance := range allVMs {
		fmt.Printf("\nBacking up VM %s (%s)...\n", vmInstance.VMID, vmInstance.Name)
		if err := backupOps.CreateBackup(vmInstance.VMID, storage, backupMode, compress); err != nil {
			logger.Errorf("Failed to backup VM %s: %v", vmInstance.VMID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✅ Successfully backed up %d/%d VMs\n", successCount, len(allVMs))
}

// handleInterrupts handles SIGINT and SIGTERM to gracefully cancel operations
func handleInterrupts(cancel context.CancelFunc, bulkMgr *bulk.Manager) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	fmt.Println("\n🛑 Received interrupt signal. Cancelling operations...")
	bulkMgr.Cancel()
	cancel()

	// Give operations time to cancel gracefully
	time.Sleep(2 * time.Second)
	fmt.Println("Operations cancelled. Exiting...")
	os.Exit(1)
}
