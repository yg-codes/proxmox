package main

// initVMRootCommands sets up the VM command hierarchy
// The existing vmCmd from main.go becomes the top-level 'vm' command
func initVMRootCommands() {
	// Add the existing vmCmd to root
	rootCmd.AddCommand(vmCmd)

	// Add direct VM commands (list, details, start, stop, shutdown)
	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmDetailsCmd)
	vmCmd.AddCommand(vmStartCmd)
	vmCmd.AddCommand(vmStopCmd)
	vmCmd.AddCommand(vmShutdownCmd)

	// Add snapshot as subcommand of vm and initialize its subcommands
	vmCmd.AddCommand(snapshotCmd)
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRollbackCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)

	// Add backup as subcommand of vm and initialize its subcommands
	vmCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
}
