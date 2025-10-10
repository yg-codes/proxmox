package backup

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yg-codes/proxmox-admin-cli/pkg/api"
	"github.com/yg-codes/proxmox-admin-cli/pkg/vm"
)

// Backup represents a VM backup
type Backup struct {
	VolID       string
	VMID        string
	Storage     string
	Node        string
	Size        float64 // Size in GB
	CreatedTime int64
	Format      string
	Content     string
}

// BackupMode defines backup operation mode
type BackupMode string

const (
	BackupModeSnapshot BackupMode = "snapshot" // VM snapshot (fastest)
	BackupModeSuspend  BackupMode = "suspend"  // Suspend VM during backup
	BackupModeStop     BackupMode = "stop"     // Stop VM during backup
)

// Operations handles backup operations
type Operations struct {
	client *api.Client
	vmOps  *vm.Operations
	logger *logrus.Logger
}

// NewOperations creates a new backup operations handler
func NewOperations(client *api.Client, vmOps *vm.Operations, logger *logrus.Logger) *Operations {
	return &Operations{
		client: client,
		vmOps:  vmOps,
		logger: logger,
	}
}

// CreateBackup creates a VM backup
func (ops *Operations) CreateBackup(vmid, storage string, mode BackupMode, compress string) error {
	ops.logger.Infof("Creating backup for VM %s", vmid)

	// Find VM node
	node, err := ops.vmOps.FindVMNode(vmid)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	fmt.Println("\n🔄 Creating backup...")
	fmt.Printf("  Storage: %s\n", storage)
	fmt.Printf("  Mode: %s\n", mode)
	fmt.Printf("  Compression: %s\n", compress)

	// Prepare backup data
	data := map[string]interface{}{
		"vmid":     vmid,
		"storage":  storage,
		"mode":     string(mode),
		"compress": compress,
		"remove":   "0", // Don't remove old backups
	}

	// Create backup
	resp, err := ops.client.Post(fmt.Sprintf("/nodes/%s/vzdump", node), data)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Extract task ID
	taskID, ok := resp["data"].(string)
	if !ok {
		return fmt.Errorf("invalid task response")
	}

	// Monitor task progress
	err = ops.vmOps.MonitorTask(node, taskID)
	if err != nil {
		fmt.Println("❌ Backup failed!")
		return err
	}

	fmt.Println("✅ Backup completed successfully!")

	// Try to get backup file info
	backups, err := ops.ListBackupsForVM(vmid, storage)
	if err == nil && len(backups) > 0 {
		// Find most recent backup
		var latest *Backup
		for _, backup := range backups {
			if latest == nil || backup.CreatedTime > latest.CreatedTime {
				latest = backup
			}
		}
		if latest != nil {
			fmt.Printf("  Backup file: %s\n", latest.VolID)
			fmt.Printf("  Size: %.2f GB\n", latest.Size)
		}
	}

	return nil
}

// ListBackupsForVM lists all backups for a specific VM
func (ops *Operations) ListBackupsForVM(vmid, storage string) ([]*Backup, error) {
	ops.logger.Debugf("Listing backups for VM %s", vmid)

	var storages []string
	if storage != "" {
		storages = []string{storage}
	} else {
		// Get all nodes to find all storages
		nodesResp, err := ops.client.Get("/nodes", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes: %w", err)
		}

		// API client wraps array responses in "items" key
		nodes, ok := nodesResp["items"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid nodes response")
		}

		storageMap := make(map[string]bool)
		for _, nodeData := range nodes {
			node, ok := nodeData.(map[string]interface{})
			if !ok {
				continue
			}
			nodeName := node["node"].(string)

			// Get storages for this node
			storagesResp, err := ops.client.Get(fmt.Sprintf("/nodes/%s/storage", nodeName), nil)
			if err != nil {
				continue
			}

			// API client wraps array responses in "items" key
		nodeStorages, ok := storagesResp["items"].([]interface{})
			if !ok {
				continue
			}

			for _, storageData := range nodeStorages {
				storageInfo, ok := storageData.(map[string]interface{})
				if !ok {
					continue
				}
				storageName := storageInfo["storage"].(string)
				storageMap[storageName] = true
			}
		}

		for s := range storageMap {
			storages = append(storages, s)
		}
	}

	var allBackups []*Backup
	seen := make(map[string]bool)

	// Get all nodes
	nodesResp, err := ops.client.Get("/nodes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// API client wraps array responses in "items" key
	nodes, ok := nodesResp["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid nodes response")
	}

	for _, storage := range storages {
		for _, nodeData := range nodes {
			node, ok := nodeData.(map[string]interface{})
			if !ok {
				continue
			}
			nodeName := node["node"].(string)

			// Try to get backup content
			params := map[string]string{"content": "backup"}
			contentsResp, err := ops.client.Get(
				fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage),
				params,
			)
			if err != nil {
				// Try without filter
				contentsResp, err = ops.client.Get(
					fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage),
					nil,
				)
				if err != nil {
					continue
				}
			}

			// API client wraps array responses in "items" key
		contents, ok := contentsResp["items"].([]interface{})
			if !ok {
				continue
			}

			for _, itemData := range contents {
				item, ok := itemData.(map[string]interface{})
				if !ok {
					continue
				}

				volid, ok := item["volid"].(string)
				if !ok {
					continue
				}

				// Skip if already seen
				if seen[volid] {
					continue
				}

				// Check if this backup belongs to the VM
				isBackup := false
				itemVMID := ""

				// Method 1: Direct VMID match
				if v, ok := item["vmid"]; ok {
					itemVMID = fmt.Sprintf("%v", v)
					if itemVMID == vmid {
						isBackup = true
					}
				}

				// Method 2: Check volid pattern
				backupPatterns := []string{
					fmt.Sprintf("vzdump-qemu-%s-", vmid),
					fmt.Sprintf("vzdump-lxc-%s-", vmid),
					fmt.Sprintf("backup-%s-", vmid),
					fmt.Sprintf("vm-%s-", vmid),
				}

				for _, pattern := range backupPatterns {
					if strings.Contains(volid, pattern) {
						isBackup = true
						if itemVMID == "" {
							itemVMID = vmid
						}
						break
					}
				}

				// Method 3: Parse volid for VMID
				if !isBackup && strings.Contains(volid, "vzdump") {
					parts := strings.Split(volid, "-")
					if len(parts) >= 3 && parts[2] == vmid {
						isBackup = true
						itemVMID = vmid
					}
				}

				if isBackup {
					backup := &Backup{
						VolID:   volid,
						VMID:    itemVMID,
						Storage: storage,
						Node:    nodeName,
						Content: "backup",
					}

					// Extract size
					if size, ok := item["size"].(float64); ok {
						backup.Size = size / (1024 * 1024 * 1024) // Convert to GB
					}

					// Extract creation time
					if ctime, ok := item["ctime"].(float64); ok {
						backup.CreatedTime = int64(ctime)
					}

					// Extract format
					if format, ok := item["format"].(string); ok {
						backup.Format = format
					}

					seen[volid] = true
					allBackups = append(allBackups, backup)
				}
			}
		}
	}

	return allBackups, nil
}

// DisplayBackups displays backups in a formatted table
func (ops *Operations) DisplayBackups(vmid, storage string) error {
	backups, err := ops.ListBackupsForVM(vmid, storage)
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found for VM %s\n", vmid)
		return nil
	}

	fmt.Printf("\nBackups for VM %s:\n", vmid)
	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("%-3s %-60s %-15s %-10s %-15s\n",
		"#", "Volume ID", "Storage", "Size", "Created")
	fmt.Println(strings.Repeat("-", 100))

	for i, backup := range backups {
		created := time.Unix(backup.CreatedTime, 0).Format("2006-01-02 15:04")
		size := fmt.Sprintf("%.2f GB", backup.Size)

		fmt.Printf("%-3d %-60s %-15s %-10s %-15s\n",
			i+1, backup.VolID, backup.Storage, size, created)
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("Total backups: %d\n", len(backups))

	return nil
}

// RestoreBackup restores a VM from backup
func (ops *Operations) RestoreBackup(vmid, volid, node, storage string) error {
	ops.logger.Infof("Restoring VM %s from backup %s", vmid, volid)

	fmt.Println("\n🔄 Restoring VM from backup...")
	fmt.Printf("  Backup: %s\n", volid)
	fmt.Printf("  Target VMID: %s\n", vmid)

	// Prepare restore data
	data := map[string]interface{}{
		"vmid":    vmid,
		"archive": volid,
		"force":   "1", // Overwrite existing VM
	}

	// If storage is specified, use it
	if storage != "" {
		data["storage"] = storage
	}

	// Execute restore
	resp, err := ops.client.Post(fmt.Sprintf("/nodes/%s/qemu", node), data)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Extract task ID
	taskID, ok := resp["data"].(string)
	if !ok {
		return fmt.Errorf("invalid task response")
	}

	// Monitor task progress
	err = ops.vmOps.MonitorTask(node, taskID)
	if err != nil {
		fmt.Println("❌ Restore failed!")
		return err
	}

	fmt.Println("✅ Restore completed successfully!")
	return nil
}

// DeleteBackup deletes a specific backup
func (ops *Operations) DeleteBackup(backup *Backup) error {
	ops.logger.Infof("Deleting backup %s", backup.VolID)

	fmt.Println("🗑️  Executing backup deletion...")

	// Delete backup via API
	resp, err := ops.client.Delete(
		fmt.Sprintf("/nodes/%s/storage/%s/content/%s",
			backup.Node, backup.Storage, backup.VolID),
	)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// Extract task ID if present
	if taskID, ok := resp["data"].(string); ok && taskID != "" {
		// Monitor task progress
		err = ops.vmOps.MonitorTask(backup.Node, taskID)
		if err != nil {
			fmt.Println("❌ Backup deletion failed!")
			return err
		}
	}

	fmt.Printf("✅ Backup %s deleted successfully!\n", backup.VolID)
	return nil
}

// DeleteBackupsByPattern deletes backups matching a pattern
func (ops *Operations) DeleteBackupsByPattern(vmid, storage, pattern string) (int, error) {
	ops.logger.Infof("Deleting backups for VM %s matching pattern %s", vmid, pattern)

	backups, err := ops.ListBackupsForVM(vmid, storage)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, backup := range backups {
		if pattern == "" || strings.Contains(backup.VolID, pattern) {
			err := ops.DeleteBackup(backup)
			if err != nil {
				ops.logger.Warnf("Failed to delete backup %s: %v", backup.VolID, err)
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

// DeleteOldBackups deletes backups older than specified days or keeps only N most recent
func (ops *Operations) DeleteOldBackups(vmid, storage string, keepCount int, maxAgeDays int) (int, error) {
	ops.logger.Infof("Cleaning up old backups for VM %s", vmid)

	backups, err := ops.ListBackupsForVM(vmid, storage)
	if err != nil {
		return 0, err
	}

	if len(backups) == 0 {
		return 0, nil
	}

	// Sort backups by creation time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].CreatedTime < backups[j].CreatedTime {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	deleted := 0
	now := time.Now().Unix()

	for i, backup := range backups {
		shouldDelete := false

		// Keep count logic
		if keepCount > 0 && i >= keepCount {
			shouldDelete = true
		}

		// Max age logic
		if maxAgeDays > 0 {
			age := (now - backup.CreatedTime) / (24 * 3600)
			if age > int64(maxAgeDays) {
				shouldDelete = true
			}
		}

		if shouldDelete {
			err := ops.DeleteBackup(backup)
			if err != nil {
				ops.logger.Warnf("Failed to delete backup %s: %v", backup.VolID, err)
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}
