package storage

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/api"
)

// Storage represents a Proxmox storage
type Storage struct {
	Name         string
	Type         string
	Node         string
	Active       bool
	ContentTypes string
	AvailableGB  float64
	TotalGB      float64
}

// Operations handles storage operations
type Operations struct {
	client *api.Client
	logger *logrus.Logger
}

// NewOperations creates a new storage operations handler
func NewOperations(client *api.Client, logger *logrus.Logger) *Operations {
	return &Operations{
		client: client,
		logger: logger,
	}
}

// GetVMStorages gets all available storages suitable for VM disks
func (ops *Operations) GetVMStorages() ([]*Storage, error) {
	ops.logger.Debug("Fetching VM disk storages")

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

	uniqueStorages := make(map[string]*Storage)

	for _, nodeData := range nodes {
		node, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		nodeName := node["node"].(string)

		// Get storages for this node
		storagesResp, err := ops.client.Get(fmt.Sprintf("/nodes/%s/storage", nodeName), nil)
		if err != nil {
			ops.logger.Debugf("Failed to get storages for node %s: %v", nodeName, err)
			continue
		}

		// API client wraps array responses in "items" key
	storages, ok := storagesResp["items"].([]interface{})
		if !ok {
			continue
		}

		for _, storageData := range storages {
			storage, ok := storageData.(map[string]interface{})
			if !ok {
				continue
			}

			storageName := storage["storage"].(string)

			// Get storage status to check content types
			statusResp, err := ops.client.Get(
				fmt.Sprintf("/nodes/%s/storage/%s/status", nodeName, storageName),
				nil,
			)
			if err != nil {
				continue
			}

			statusData, ok := statusResp["data"].(map[string]interface{})
			if !ok {
				continue
			}

			content := ""
			if c, ok := statusData["content"].(string); ok {
				content = c
			}

			// Check if storage supports VM images or root directories
			if strings.Contains(content, "images") || strings.Contains(content, "rootdir") {
				if _, exists := uniqueStorages[storageName]; !exists {
					storageType := storage["type"].(string)
					active := statusData["active"].(float64) == 1

					avail := 0.0
					total := 0.0
					if a, ok := statusData["avail"].(float64); ok {
						avail = a / (1024 * 1024 * 1024) // Convert to GB
					}
					if t, ok := statusData["total"].(float64); ok {
						total = t / (1024 * 1024 * 1024) // Convert to GB
					}

					uniqueStorages[storageName] = &Storage{
						Name:         storageName,
						Type:         storageType,
						Node:         nodeName,
						Active:       active,
						ContentTypes: content,
						AvailableGB:  avail,
						TotalGB:      total,
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]*Storage, 0, len(uniqueStorages))
	for _, storage := range uniqueStorages {
		result = append(result, storage)
	}

	ops.logger.Infof("Found %d VM disk storages", len(result))
	return result, nil
}

// GetBackupStorages gets all available storages suitable for backups
func (ops *Operations) GetBackupStorages() ([]*Storage, error) {
	ops.logger.Debug("Fetching backup storages")

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

	uniqueStorages := make(map[string]*Storage)

	for _, nodeData := range nodes {
		node, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		nodeName := node["node"].(string)

		// Get storages for this node
		storagesResp, err := ops.client.Get(fmt.Sprintf("/nodes/%s/storage", nodeName), nil)
		if err != nil {
			ops.logger.Debugf("Failed to get storages for node %s: %v", nodeName, err)
			continue
		}

		// API client wraps array responses in "items" key
	storages, ok := storagesResp["items"].([]interface{})
		if !ok {
			continue
		}

		for _, storageData := range storages {
			storage, ok := storageData.(map[string]interface{})
			if !ok {
				continue
			}

			storageName := storage["storage"].(string)

			// Get storage status to check content types
			statusResp, err := ops.client.Get(
				fmt.Sprintf("/nodes/%s/storage/%s/status", nodeName, storageName),
				nil,
			)
			if err != nil {
				continue
			}

			statusData, ok := statusResp["data"].(map[string]interface{})
			if !ok {
				continue
			}

			content := ""
			if c, ok := statusData["content"].(string); ok {
				content = c
			}

			// Check if storage supports backups
			if strings.Contains(content, "backup") || strings.Contains(content, "vztmpl") {
				if _, exists := uniqueStorages[storageName]; !exists {
					storageType := storage["type"].(string)
					active := statusData["active"].(float64) == 1

					avail := 0.0
					total := 0.0
					if a, ok := statusData["avail"].(float64); ok {
						avail = a / (1024 * 1024 * 1024) // Convert to GB
					}
					if t, ok := statusData["total"].(float64); ok {
						total = t / (1024 * 1024 * 1024) // Convert to GB
					}

					uniqueStorages[storageName] = &Storage{
						Name:         storageName,
						Type:         storageType,
						Node:         nodeName,
						Active:       active,
						ContentTypes: content,
						AvailableGB:  avail,
						TotalGB:      total,
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]*Storage, 0, len(uniqueStorages))
	for _, storage := range uniqueStorages {
		result = append(result, storage)
	}

	ops.logger.Infof("Found %d backup storages", len(result))
	return result, nil
}

// DisplayVMStorages displays VM storages in a formatted table
func (ops *Operations) DisplayVMStorages() error {
	storages, err := ops.GetVMStorages()
	if err != nil {
		return err
	}

	if len(storages) == 0 {
		fmt.Println("❌ No VM disk storages found")
		return nil
	}

	fmt.Println("\nAvailable VM Disk Storages:")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-3s %-15s %-10s %-10s %-20s %-15s\n",
		"#", "Storage", "Type", "Status", "Content Types", "Free Space")
	fmt.Println(strings.Repeat("-", 80))

	for i, storage := range storages {
		status := "🔴 inactive"
		if storage.Active {
			status = "🟢 active"
		}

		freeSpace := "N/A"
		if storage.AvailableGB > 0 {
			freeSpace = fmt.Sprintf("%.1f GB", storage.AvailableGB)
		}

		fmt.Printf("%-3d %-15s %-10s %-10s %-20s %-15s\n",
			i+1, storage.Name, storage.Type, status, storage.ContentTypes, freeSpace)
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Total VM disk storages: %d\n", len(storages))

	return nil
}

// DisplayBackupStorages displays backup storages in a formatted table
func (ops *Operations) DisplayBackupStorages() error {
	storages, err := ops.GetBackupStorages()
	if err != nil {
		return err
	}

	if len(storages) == 0 {
		fmt.Println("❌ No backup-capable storages found")
		return nil
	}

	fmt.Println("\nAvailable Backup Storages:")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-3s %-15s %-10s %-10s %-15s %-15s\n",
		"#", "Storage", "Type", "Status", "Free Space", "Total Space")
	fmt.Println(strings.Repeat("-", 70))

	for i, storage := range storages {
		status := "🔴 inactive"
		if storage.Active {
			status = "🟢 active"
		}

		freeSpace := "N/A"
		totalSpace := "N/A"
		if storage.TotalGB > 0 {
			freeSpace = fmt.Sprintf("%.1f GB", storage.AvailableGB)
			totalSpace = fmt.Sprintf("%.1f GB", storage.TotalGB)
		}

		fmt.Printf("%-3d %-15s %-10s %-10s %-15s %-15s\n",
			i+1, storage.Name, storage.Type, status, freeSpace, totalSpace)
	}

	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Total storages: %d\n", len(storages))

	return nil
}

// ValidateStorage checks if a storage exists and is active
func (ops *Operations) ValidateStorage(storageName string) error {
	backupStorages, err := ops.GetBackupStorages()
	if err != nil {
		return err
	}

	for _, storage := range backupStorages {
		if storage.Name == storageName {
			if !storage.Active {
				return fmt.Errorf("storage %s is inactive", storageName)
			}
			if storage.AvailableGB < 1 {
				return fmt.Errorf("storage %s has insufficient space", storageName)
			}
			return nil
		}
	}

	return fmt.Errorf("storage %s not found or not suitable for backups", storageName)
}
