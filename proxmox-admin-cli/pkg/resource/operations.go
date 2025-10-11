package resource

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yg-codes/proxmox-admin-cli/pkg/api"
)

// Operations handles resource monitoring operations
type Operations struct {
	client *api.Client
	logger *logrus.Logger
}

// NewOperations creates a new resource operations instance
func NewOperations(client *api.Client, logger *logrus.Logger) *Operations {
	if logger == nil {
		logger = logrus.New()
	}

	return &Operations{
		client: client,
		logger: logger,
	}
}

// GetClusterResources gets cluster-wide resource information
// API: GET /cluster/resources
func (ops *Operations) GetClusterResources(filter *ResourceFilter) (*ClusterResources, error) {
	ops.logger.Debug("Fetching cluster resources")

	params := make(map[string]string)
	if filter != nil && filter.Type != "" {
		params["type"] = filter.Type
	}

	resp, err := ops.client.Get("/cluster/resources", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster resources: %w", err)
	}

	resources := &ClusterResources{
		Nodes:   []*NodeResource{},
		VMs:     []*VMResource{},
		Storage: []*StorageResource{},
	}

	if items, ok := resp["items"].([]interface{}); ok {
		for _, item := range items {
			if resMap, ok := item.(map[string]interface{}); ok {
				resType, _ := resMap["type"].(string)

				switch resType {
				case "node":
					node := parseNodeResource(resMap)
					if filter == nil || matchesFilter(node, filter) {
						resources.Nodes = append(resources.Nodes, node)
					}
				case "qemu", "lxc":
					vm := parseVMResource(resMap)
					if filter == nil || matchesFilter(vm, filter) {
						resources.VMs = append(resources.VMs, vm)
					}
				case "storage":
					storage := parseStorageResource(resMap)
					if filter == nil || matchesFilter(storage, filter) {
						resources.Storage = append(resources.Storage, storage)
					}
				}
			}
		}
	}

	ops.logger.Infof("Retrieved %d nodes, %d VMs, %d storages",
		len(resources.Nodes), len(resources.VMs), len(resources.Storage))
	return resources, nil
}

// GetNodeResources gets resource usage for a specific node
// API: GET /nodes/{node}/status
func (ops *Operations) GetNodeResources(nodeName string) (*NodeResource, error) {
	ops.logger.Debugf("Fetching resources for node: %s", nodeName)

	path := fmt.Sprintf("/nodes/%s/status", nodeName)
	resp, err := ops.client.Get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get node resources: %w", err)
	}

	node := parseNodeResource(resp)
	node.Node = nodeName

	return node, nil
}

// GetVMResources gets resource usage for a specific VM
// API: GET /nodes/{node}/{type}/{vmid}/status/current
func (ops *Operations) GetVMResources(nodeName, vmType string, vmid int) (*VMResource, error) {
	ops.logger.Debugf("Fetching resources for VM %d on node %s", vmid, nodeName)

	path := fmt.Sprintf("/nodes/%s/%s/%d/status/current", nodeName, vmType, vmid)
	resp, err := ops.client.Get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM resources: %w", err)
	}

	vm := parseVMResource(resp)
	vm.VMID = vmid
	vm.Node = nodeName
	vm.Type = vmType

	return vm, nil
}

// GetNodeRRDData gets RRD data for a node
// API: GET /nodes/{node}/rrddata
func (ops *Operations) GetNodeRRDData(nodeName, timeframe string) (*RRDData, error) {
	ops.logger.Debugf("Fetching RRD data for node %s (timeframe: %s)", nodeName, timeframe)

	params := map[string]string{
		"timeframe": timeframe,
	}

	path := fmt.Sprintf("/nodes/%s/rrddata", nodeName)
	resp, err := ops.client.Get(path, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get node RRD data: %w", err)
	}

	rrdData := &RRDData{
		TimeFrame:  timeframe,
		DataPoints: []*ResourceHistory{},
	}

	if items, ok := resp["items"].([]interface{}); ok {
		for _, item := range items {
			if dataMap, ok := item.(map[string]interface{}); ok {
				point := parseRRDDataPoint(dataMap)
				rrdData.DataPoints = append(rrdData.DataPoints, point)
			}
		}
	}

	ops.logger.Infof("Retrieved %d RRD data points for node %s", len(rrdData.DataPoints), nodeName)
	return rrdData, nil
}

// GetVMRRDData gets RRD data for a VM
// API: GET /nodes/{node}/{type}/{vmid}/rrddata
func (ops *Operations) GetVMRRDData(nodeName, vmType string, vmid int, timeframe string) (*RRDData, error) {
	ops.logger.Debugf("Fetching RRD data for VM %d (timeframe: %s)", vmid, timeframe)

	params := map[string]string{
		"timeframe": timeframe,
	}

	path := fmt.Sprintf("/nodes/%s/%s/%d/rrddata", nodeName, vmType, vmid)
	resp, err := ops.client.Get(path, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM RRD data: %w", err)
	}

	rrdData := &RRDData{
		TimeFrame:  timeframe,
		DataPoints: []*ResourceHistory{},
	}

	if items, ok := resp["items"].([]interface{}); ok {
		for _, item := range items {
			if dataMap, ok := item.(map[string]interface{}); ok {
				point := parseRRDDataPoint(dataMap)
				rrdData.DataPoints = append(rrdData.DataPoints, point)
			}
		}
	}

	ops.logger.Infof("Retrieved %d RRD data points for VM %d", len(rrdData.DataPoints), vmid)
	return rrdData, nil
}

// GetClusterStats calculates aggregated cluster statistics
func (ops *Operations) GetClusterStats() (*ResourceStats, error) {
	ops.logger.Debug("Calculating cluster statistics")

	resources, err := ops.GetClusterResources(nil)
	if err != nil {
		return nil, err
	}

	stats := &ResourceStats{}

	// Count nodes
	stats.TotalNodes = len(resources.Nodes)
	for _, node := range resources.Nodes {
		if node.Online {
			stats.OnlineNodes++
		}
		stats.TotalCPU += node.MaxCPU
		stats.UsedCPU += node.CPU
		stats.TotalMemory += node.MaxMemory
		stats.UsedMemory += node.Memory
		stats.TotalDisk += node.MaxDisk
		stats.UsedDisk += node.Disk
	}

	// Calculate percentages
	if stats.TotalCPU > 0 {
		stats.CPUPercent = (stats.UsedCPU / float64(stats.TotalCPU)) * 100
	}
	if stats.TotalMemory > 0 {
		stats.MemoryPercent = (float64(stats.UsedMemory) / float64(stats.TotalMemory)) * 100
	}
	if stats.TotalDisk > 0 {
		stats.DiskPercent = (float64(stats.UsedDisk) / float64(stats.TotalDisk)) * 100
	}

	// Count VMs
	stats.TotalVMs = len(resources.VMs)
	for _, vm := range resources.VMs {
		if vm.Status == "running" {
			stats.RunningVMs++
		} else {
			stats.StoppedVMs++
		}
	}

	// Storage stats
	stats.TotalStorage = len(resources.Storage)
	for _, storage := range resources.Storage {
		if storage.Active {
			stats.AvailableStorage++
		}
		stats.UsedStorage += storage.Used
		stats.TotalStorageSize += storage.Total
	}

	if stats.TotalStorageSize > 0 {
		stats.StoragePercent = (float64(stats.UsedStorage) / float64(stats.TotalStorageSize)) * 100
	}

	ops.logger.Info("Cluster statistics calculated successfully")
	return stats, nil
}

// Helper functions

func parseNodeResource(resMap map[string]interface{}) *NodeResource {
	node := &NodeResource{}

	if name, ok := resMap["node"].(string); ok {
		node.Node = name
	}
	if resType, ok := resMap["type"].(string); ok {
		node.Type = resType
	}
	if status, ok := resMap["status"].(string); ok {
		node.Status = status
		node.Online = (status == "online")
	}

	// CPU metrics
	if cpu, ok := resMap["cpu"].(float64); ok {
		node.CPU = cpu
		if maxcpu, ok := resMap["maxcpu"].(float64); ok {
			node.MaxCPU = int(maxcpu)
			node.CPUPercent = (cpu / maxcpu) * 100
		}
	}

	// Memory metrics
	if mem, ok := resMap["mem"].(float64); ok {
		node.Memory = int64(mem)
	}
	if maxmem, ok := resMap["maxmem"].(float64); ok {
		node.MaxMemory = int64(maxmem)
		if node.MaxMemory > 0 {
			node.MemPercent = (float64(node.Memory) / float64(node.MaxMemory)) * 100
		}
	}

	// Disk metrics
	if disk, ok := resMap["disk"].(float64); ok {
		node.Disk = int64(disk)
	}
	if maxdisk, ok := resMap["maxdisk"].(float64); ok {
		node.MaxDisk = int64(maxdisk)
		if node.MaxDisk > 0 {
			node.DiskPercent = (float64(node.Disk) / float64(node.MaxDisk)) * 100
		}
	}

	// Network metrics
	if netin, ok := resMap["netin"].(float64); ok {
		node.NetIn = int64(netin)
	}
	if netout, ok := resMap["netout"].(float64); ok {
		node.NetOut = int64(netout)
	}

	// Uptime
	if uptime, ok := resMap["uptime"].(float64); ok {
		node.Uptime = int64(uptime)
	}

	return node
}

func parseVMResource(resMap map[string]interface{}) *VMResource {
	vm := &VMResource{}

	if vmid, ok := resMap["vmid"].(float64); ok {
		vm.VMID = int(vmid)
	}
	if name, ok := resMap["name"].(string); ok {
		vm.Name = name
	}
	if node, ok := resMap["node"].(string); ok {
		vm.Node = node
	}
	if vmType, ok := resMap["type"].(string); ok {
		vm.Type = vmType
	}
	if status, ok := resMap["status"].(string); ok {
		vm.Status = status
	}

	// CPU metrics
	if cpu, ok := resMap["cpu"].(float64); ok {
		vm.CPU = cpu
	}
	if maxcpu, ok := resMap["maxcpu"].(float64); ok {
		vm.MaxCPU = int(maxcpu)
		if vm.MaxCPU > 0 {
			vm.CPUPercent = (vm.CPU / float64(vm.MaxCPU)) * 100
		}
	}

	// Memory metrics
	if mem, ok := resMap["mem"].(float64); ok {
		vm.Memory = int64(mem)
	}
	if maxmem, ok := resMap["maxmem"].(float64); ok {
		vm.MaxMemory = int64(maxmem)
		if vm.MaxMemory > 0 {
			vm.MemPercent = (float64(vm.Memory) / float64(vm.MaxMemory)) * 100
		}
	}

	// Disk metrics
	if disk, ok := resMap["disk"].(float64); ok {
		vm.Disk = int64(disk)
	}
	if maxdisk, ok := resMap["maxdisk"].(float64); ok {
		vm.MaxDisk = int64(maxdisk)
		if vm.MaxDisk > 0 {
			vm.DiskPercent = (float64(vm.Disk) / float64(vm.MaxDisk)) * 100
		}
	}

	// Network metrics
	if netin, ok := resMap["netin"].(float64); ok {
		vm.NetIn = int64(netin)
	}
	if netout, ok := resMap["netout"].(float64); ok {
		vm.NetOut = int64(netout)
	}

	// Runtime info
	if uptime, ok := resMap["uptime"].(float64); ok {
		vm.Uptime = int64(uptime)
	}
	if pid, ok := resMap["pid"].(float64); ok {
		vm.PID = int(pid)
	}

	return vm
}

func parseStorageResource(resMap map[string]interface{}) *StorageResource {
	storage := &StorageResource{}

	if name, ok := resMap["storage"].(string); ok {
		storage.Storage = name
	}
	if node, ok := resMap["node"].(string); ok {
		storage.Node = node
	}
	if storageType, ok := resMap["type"].(string); ok {
		storage.Type = storageType
	}
	if content, ok := resMap["content"].(string); ok {
		storage.Content = content
	}
	if status, ok := resMap["status"].(string); ok {
		storage.Status = status
		storage.Active = (status == "available")
	}

	// Disk metrics
	if used, ok := resMap["used"].(float64); ok {
		storage.Used = int64(used)
	}
	if total, ok := resMap["total"].(float64); ok {
		storage.Total = int64(total)
	}
	if avail, ok := resMap["avail"].(float64); ok {
		storage.Available = int64(avail)
	}

	if storage.Total > 0 {
		storage.UsePercent = (float64(storage.Used) / float64(storage.Total)) * 100
	}

	return storage
}

func parseRRDDataPoint(dataMap map[string]interface{}) *ResourceHistory {
	point := &ResourceHistory{}

	if timestamp, ok := dataMap["time"].(float64); ok {
		point.Time = time.Unix(int64(timestamp), 0)
	}
	if cpu, ok := dataMap["cpu"].(float64); ok {
		point.CPU = cpu
	}
	if mem, ok := dataMap["mem"].(float64); ok {
		point.Memory = mem
	}
	if disk, ok := dataMap["disk"].(float64); ok {
		point.Disk = disk
	}
	if netin, ok := dataMap["netin"].(float64); ok {
		point.NetIn = netin
	}
	if netout, ok := dataMap["netout"].(float64); ok {
		point.NetOut = netout
	}
	if iodelay, ok := dataMap["iodelay"].(float64); ok {
		point.IODelay = iodelay
	}
	if loadavg, ok := dataMap["loadavg"].(float64); ok {
		point.LoadAvg = loadavg
	}

	return point
}

func matchesFilter(resource interface{}, filter *ResourceFilter) bool {
	if filter == nil {
		return true
	}

	switch r := resource.(type) {
	case *NodeResource:
		if filter.Node != "" && r.Node != filter.Node {
			return false
		}
		if filter.Status != "" && r.Status != filter.Status {
			return false
		}
	case *VMResource:
		if filter.Node != "" && r.Node != filter.Node {
			return false
		}
		if filter.Status != "" && r.Status != filter.Status {
			return false
		}
		if filter.Type != "" && r.Type != filter.Type {
			return false
		}
	case *StorageResource:
		if filter.Node != "" && r.Node != filter.Node {
			return false
		}
		if filter.Status != "" && r.Status != filter.Status {
			return false
		}
	}

	return true
}
