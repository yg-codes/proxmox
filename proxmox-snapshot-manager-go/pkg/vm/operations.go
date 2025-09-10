package vm

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yg-codes/proxmox-snapshot-manager-go/pkg/api"
)

// VM represents a Proxmox virtual machine
type VM struct {
	VMID     string `json:"vmid"`
	Name     string `json:"name"`
	Node     string `json:"node"`
	Status   string `json:"status"`
	Running  bool   `json:"running"`
	Memory   int64  `json:"memory"`
	CPUs     int    `json:"cpus"`
	DiskSize int64  `json:"disksize"`
}

// Node represents a Proxmox node
type Node struct {
	Name   string `json:"node"`
	Status string `json:"status"`
	Online bool   `json:"online"`
}

// TaskStatus represents the status of a Proxmox task
type TaskStatus struct {
	UPID     string  `json:"upid"`
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	ExitCode string  `json:"exitstatus"`
	Progress float64 `json:"progress"`
}

// Operations handles VM operations
type Operations struct {
	client    *api.Client
	logger    *logrus.Logger
	nodeCache map[string]*Node
}

// NewOperations creates a new VM operations instance
func NewOperations(client *api.Client, logger *logrus.Logger) *Operations {
	if logger == nil {
		logger = logrus.New()
	}

	return &Operations{
		client:    client,
		logger:    logger,
		nodeCache: make(map[string]*Node),
	}
}

// GetNodes retrieves all nodes in the cluster
func (ops *Operations) GetNodes() ([]*Node, error) {
	if len(ops.nodeCache) > 0 {
		nodes := make([]*Node, 0, len(ops.nodeCache))
		for _, node := range ops.nodeCache {
			nodes = append(nodes, node)
		}
		return nodes, nil
	}

	resp, err := ops.client.Get("/nodes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var nodes []*Node
	if items, ok := resp["items"].([]interface{}); ok {
		for _, item := range items {
			if nodeMap, ok := item.(map[string]interface{}); ok {
				node := &Node{}
				if name, ok := nodeMap["node"].(string); ok {
					node.Name = name
				}
				if status, ok := nodeMap["status"].(string); ok {
					node.Status = status
					node.Online = status == "online"
				}
				nodes = append(nodes, node)
				ops.nodeCache[node.Name] = node
			}
		}
	}

	return nodes, nil
}

// FindVMNode finds which node a VM is located on
func (ops *Operations) FindVMNode(vmid string) (string, error) {
	nodes, err := ops.GetNodes()
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		// Try to get VM status on this node
		path := fmt.Sprintf("/nodes/%s/qemu/%s/status/current", node.Name, vmid)
		_, err := ops.client.Get(path, nil)
		if err == nil {
			return node.Name, nil
		}
	}

	return "", fmt.Errorf("VM %s not found on any node", vmid)
}

// GetAllVMs retrieves all VMs from all nodes
func (ops *Operations) GetAllVMs() ([]*VM, error) {
	nodes, err := ops.GetNodes()
	if err != nil {
		return nil, err
	}

	var allVMs []*VM
	for _, node := range nodes {
		vms, err := ops.getVMsFromNode(node.Name)
		if err != nil {
			ops.logger.Warnf("Failed to get VMs from node %s: %v", node.Name, err)
			continue
		}
		allVMs = append(allVMs, vms...)
	}

	return allVMs, nil
}

// getVMsFromNode retrieves VMs from a specific node
func (ops *Operations) getVMsFromNode(nodeName string) ([]*VM, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", nodeName)
	resp, err := ops.client.Get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs from node %s: %w", nodeName, err)
	}

	var vms []*VM
	if items, ok := resp["items"].([]interface{}); ok {
		for _, item := range items {
			if vmMap, ok := item.(map[string]interface{}); ok {
				vm := &VM{Node: nodeName}

				if vmid, ok := vmMap["vmid"].(float64); ok {
					vm.VMID = fmt.Sprintf("%.0f", vmid)
				}
				if name, ok := vmMap["name"].(string); ok {
					vm.Name = name
				}
				if status, ok := vmMap["status"].(string); ok {
					vm.Status = status
					vm.Running = status == "running"
				}
				if memory, ok := vmMap["memory"].(float64); ok {
					vm.Memory = int64(memory)
				}
				if cpus, ok := vmMap["cpus"].(float64); ok {
					vm.CPUs = int(cpus)
				}
				if disksize, ok := vmMap["disksize"].(float64); ok {
					vm.DiskSize = int64(disksize)
				}

				vms = append(vms, vm)
			}
		}
	}

	return vms, nil
}

// GetVMStatus retrieves the current status of a VM
func (ops *Operations) GetVMStatus(vmid string) (*VM, error) {
	node, err := ops.FindVMNode(vmid)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%s/status/current", node, vmid)
	resp, err := ops.client.Get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM %s status: %w", vmid, err)
	}

	vm := &VM{
		VMID: vmid,
		Node: node,
	}

	if name, ok := resp["name"].(string); ok {
		vm.Name = name
	}
	if status, ok := resp["status"].(string); ok {
		vm.Status = status
		vm.Running = status == "running"
	}
	if memory, ok := resp["memory"].(float64); ok {
		vm.Memory = int64(memory)
	}
	if cpus, ok := resp["cpus"].(float64); ok {
		vm.CPUs = int(cpus)
	}

	return vm, nil
}

// StartVM starts a virtual machine
func (ops *Operations) StartVM(vmid string) error {
	node, err := ops.FindVMNode(vmid)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%s/status/start", node, vmid)
	_, err = ops.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("failed to start VM %s: %w", vmid, err)
	}

	ops.logger.Infof("Started VM %s", vmid)
	return nil
}

// StopVM stops a virtual machine
func (ops *Operations) StopVM(vmid string) error {
	node, err := ops.FindVMNode(vmid)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%s/status/stop", node, vmid)
	_, err = ops.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("failed to stop VM %s: %w", vmid, err)
	}

	ops.logger.Infof("Stopped VM %s", vmid)
	return nil
}

// ShutdownVM gracefully shuts down a virtual machine
func (ops *Operations) ShutdownVM(vmid string) error {
	node, err := ops.FindVMNode(vmid)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%s/status/shutdown", node, vmid)
	_, err = ops.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("failed to shutdown VM %s: %w", vmid, err)
	}

	ops.logger.Infof("Shutdown VM %s", vmid)
	return nil
}

// ResetVM resets a virtual machine
func (ops *Operations) ResetVM(vmid string) error {
	node, err := ops.FindVMNode(vmid)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%s/status/reset", node, vmid)
	_, err = ops.client.Post(path, nil)
	if err != nil {
		return fmt.Errorf("failed to reset VM %s: %w", vmid, err)
	}

	ops.logger.Infof("Reset VM %s", vmid)
	return nil
}

// MonitorTask monitors a Proxmox task until completion
func (ops *Operations) MonitorTask(node, upid string) error {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", node, upid)

	for {
		resp, err := ops.client.Get(path, nil)
		if err != nil {
			return fmt.Errorf("failed to get task status: %w", err)
		}

		status := "running"
		if s, ok := resp["status"].(string); ok {
			status = s
		}

		exitStatus := ""
		if es, ok := resp["exitstatus"].(string); ok {
			exitStatus = es
		}

		if status == "stopped" {
			if exitStatus == "OK" || exitStatus == "" {
				return nil
			}
			return fmt.Errorf("task failed with exit status: %s", exitStatus)
		}

		time.Sleep(1 * time.Second)
	}
}

// GetTaskStatus gets the status of a Proxmox task
func (ops *Operations) GetTaskStatus(node, upid string) (*TaskStatus, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", node, upid)
	resp, err := ops.client.Get(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}

	task := &TaskStatus{UPID: upid}

	if status, ok := resp["status"].(string); ok {
		task.Status = status
	}
	if taskType, ok := resp["type"].(string); ok {
		task.Type = taskType
	}
	if exitCode, ok := resp["exitstatus"].(string); ok {
		task.ExitCode = exitCode
	}
	if progress, ok := resp["progress"].(float64); ok {
		task.Progress = progress
	}

	return task, nil
}

// VMExists checks if a VM exists
func (ops *Operations) VMExists(vmid string) bool {
	_, err := ops.FindVMNode(vmid)
	return err == nil
}

// ValidateVMID validates that a VM ID is numeric
func ValidateVMID(vmid string) error {
	if _, err := strconv.Atoi(vmid); err != nil {
		return fmt.Errorf("invalid VM ID '%s': must be numeric", vmid)
	}
	return nil
}
