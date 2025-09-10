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
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/bulk"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/config"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/snapshot"
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

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&batchMode, "batch", false, "batch mode - no interactive prompts")
	rootCmd.PersistentFlags().BoolVarP(&autoConfirm, "yes", "y", false, "auto-confirm operations")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output")

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

	// Add commands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
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

func runInteractiveMode() {
	fmt.Println("🚀 Proxmox Snapshot Manager - Interactive Mode")
	fmt.Println("============================================")

	for {
		fmt.Println("\nAvailable operations:")
		fmt.Println("1. Create snapshots")
		fmt.Println("2. List snapshots")
		fmt.Println("3. Rollback snapshots")
		fmt.Println("4. Delete snapshots")
		fmt.Println("5. Start VMs")
		fmt.Println("6. Stop VMs")
		fmt.Println("0. Exit")

		fmt.Print("\nSelect operation (0-6): ")
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handleInterrupts(cancel, bulkMgr)

	if err := bulkMgr.StopVMs(ctx, vms); err != nil {
		logger.Errorf("VM stop failed: %v", err)
		return
	}

	bulkMgr.PrintSummary()
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
