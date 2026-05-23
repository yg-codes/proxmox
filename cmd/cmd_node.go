package main

// initNodeRootCommands sets up the node command hierarchy
// The existing nodeCmd from node.go becomes the top-level 'node' command
func initNodeRootCommands() {
	// Add the existing nodeCmd to root
	rootCmd.AddCommand(nodeCmd)

	// Initialize the node commands (from node.go)
	initNodeCommands()

	// Add resource as subcommand of node
	nodeCmd.AddCommand(resourceCmd)

	// Initialize resource commands
	initResourceCommands()
}
