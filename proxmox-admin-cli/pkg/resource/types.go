package resource

import "time"

// ClusterResources represents cluster-wide resource information
type ClusterResources struct {
	Nodes   []*NodeResource
	VMs     []*VMResource
	Storage []*StorageResource
}

// NodeResource represents node resource usage
type NodeResource struct {
	Node   string
	Type   string
	Status string
	Online bool

	// CPU metrics
	CPU        float64 // Current CPU usage (0-1)
	MaxCPU     int     // Number of CPU cores
	CPUPercent float64 // CPU usage percentage

	// Memory metrics
	Memory     int64   // Used memory in bytes
	MaxMemory  int64   // Total memory in bytes
	MemPercent float64 // Memory usage percentage

	// Disk metrics
	Disk        int64   // Used disk in bytes
	MaxDisk     int64   // Total disk in bytes
	DiskPercent float64 // Disk usage percentage

	// Network metrics
	NetIn  int64 // Network input in bytes
	NetOut int64 // Network output in bytes

	// System info
	Uptime        int64
	LoadAverage   []float64 // 1, 5, 15 minute load averages
	KernelVersion string
	PVEVersion    string
}

// VMResource represents VM resource usage
type VMResource struct {
	VMID   int
	Name   string
	Node   string
	Type   string // qemu or lxc
	Status string

	// CPU metrics
	CPU        float64 // Current CPU usage (0-1)
	MaxCPU     int     // Number of vCPUs
	CPUPercent float64

	// Memory metrics
	Memory     int64 // Used memory in bytes
	MaxMemory  int64 // Allocated memory in bytes
	MemPercent float64

	// Disk metrics
	Disk        int64 // Used disk in bytes
	MaxDisk     int64 // Allocated disk in bytes
	DiskPercent float64

	// Network metrics
	NetIn  int64 // Network input in bytes
	NetOut int64 // Network output in bytes

	// Runtime info
	Uptime int64
	PID    int
}

// StorageResource represents storage resource usage
type StorageResource struct {
	Storage string
	Node    string
	Type    string
	Content string
	Status  string
	Active  bool

	// Disk metrics
	Used       int64   // Used space in bytes
	Total      int64   // Total space in bytes
	Available  int64   // Available space in bytes
	UsePercent float64 // Usage percentage
}

// ResourceHistory represents historical resource data
type ResourceHistory struct {
	Time    time.Time
	CPU     float64
	Memory  float64
	Disk    float64
	NetIn   float64
	NetOut  float64
	IODelay float64
	LoadAvg float64
}

// ResourceStats represents aggregated statistics
type ResourceStats struct {
	// Cluster totals
	TotalNodes  int
	OnlineNodes int
	TotalVMs    int
	RunningVMs  int
	StoppedVMs  int

	// Cluster resources
	TotalCPU   int
	UsedCPU    float64
	CPUPercent float64

	TotalMemory   int64
	UsedMemory    int64
	MemoryPercent float64

	TotalDisk   int64
	UsedDisk    int64
	DiskPercent float64

	// Storage stats
	TotalStorage     int
	AvailableStorage int
	UsedStorage      int64
	TotalStorageSize int64
	StoragePercent   float64
}

// RRDData represents RRD (Round Robin Database) data for graphs
type RRDData struct {
	TimeFrame  string // hour, day, week, month, year
	DataPoints []*ResourceHistory
}

// ResourceFilter for filtering resources
type ResourceFilter struct {
	Type   string // node, vm, storage, qemu, lxc
	Node   string // Filter by specific node
	Status string // running, stopped, online, offline
}
