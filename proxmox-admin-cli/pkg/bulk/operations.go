package bulk

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yg-codes/proxmox-admin-cli/pkg/snapshot"
	"github.com/yg-codes/proxmox-admin-cli/pkg/vm"
)

// OperationResult represents the result of a bulk operation on a single VM
type OperationResult struct {
	VMID      string        `json:"vmid"`
	VMName    string        `json:"vm_name"`
	Operation string        `json:"operation"`
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Error     error         `json:"error,omitempty"`
}

// ProgressUpdate represents a progress update for bulk operations
type ProgressUpdate struct {
	Completed  int
	Total      int
	Successful int
	Failed     int
	Current    string
	Progress   float64
}

// Manager handles bulk operations with progress tracking and concurrent execution
type Manager struct {
	vmOps       *vm.Operations
	snapshotOps *snapshot.Operations
	logger      *logrus.Logger
	maxWorkers  int

	// Progress tracking
	results      []OperationResult
	resultsChan  chan OperationResult
	progressChan chan ProgressUpdate
	mu           sync.RWMutex
	cancelled    bool
	progressWG   sync.WaitGroup
}

// NewManager creates a new bulk operations manager
func NewManager(vmOps *vm.Operations, snapshotOps *snapshot.Operations, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}

	return &Manager{
		vmOps:        vmOps,
		snapshotOps:  snapshotOps,
		logger:       logger,
		maxWorkers:   3, // Default concurrent operations
		results:      make([]OperationResult, 0),
		resultsChan:  make(chan OperationResult, 100),
		progressChan: make(chan ProgressUpdate, 100),
	}
}

// SetMaxWorkers sets the maximum number of concurrent workers
func (m *Manager) SetMaxWorkers(maxWorkers int) {
	if maxWorkers > 0 {
		m.maxWorkers = maxWorkers
	}
}

// Cancel cancels ongoing operations
func (m *Manager) Cancel() {
	m.mu.Lock()
	m.cancelled = true
	m.mu.Unlock()
}

// IsCancelled checks if operations are cancelled
func (m *Manager) IsCancelled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cancelled
}

// GetResults returns a copy of the current results
func (m *Manager) GetResults() []OperationResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resultsCopy := make([]OperationResult, len(m.results))
	copy(resultsCopy, m.results)
	return resultsCopy
}

// GetProgress returns current progress statistics
func (m *Manager) GetProgress() (int, int, int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	completed := len(m.results)
	successful := 0
	failed := 0

	for _, result := range m.results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	return completed, successful, failed, completed
}

// CreateSnapshots creates snapshots for multiple VMs concurrently
func (m *Manager) CreateSnapshots(ctx context.Context, vms []*vm.VM, nameOrPrefix string, useExactName, saveVMState bool) error {
	if len(vms) == 0 {
		return fmt.Errorf("no VMs provided")
	}

	m.logger.Infof("Starting bulk snapshot creation for %d VMs", len(vms))
	m.resetResults()

	// Create worker pool
	jobs := make(chan *vm.VM, len(vms))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.maxWorkers; i++ {
		wg.Add(1)
		go m.createSnapshotWorker(ctx, &wg, jobs, nameOrPrefix, useExactName, saveVMState)
	}

	// Start progress monitor
	m.progressWG.Add(1)
	go m.progressMonitor(len(vms))

	// Send jobs
	for _, vm := range vms {
		select {
		case jobs <- vm:
		case <-ctx.Done():
			close(jobs)
			return ctx.Err()
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	m.logger.Debugf("All create workers completed, results expected: %d", len(vms))

	// Wait for all results to be processed before closing channel
	for {
		m.mu.RLock()
		resultCount := len(m.results)
		m.mu.RUnlock()

		if resultCount >= len(vms) {
			m.logger.Debugf("All results processed (%d/%d), closing channel", resultCount, len(vms))
			break
		}

		m.logger.Debugf("Waiting for results: %d/%d", resultCount, len(vms))
		time.Sleep(50 * time.Millisecond)
	}

	close(m.resultsChan)

	// Wait for progress monitor to finish processing all results
	m.logger.Debugf("Waiting for progress monitor to finish...")
	m.progressWG.Wait()
	m.logger.Debugf("Progress monitor finished, final results count: %d", len(m.results))

	return nil
}

// DeleteSnapshots deletes snapshots from multiple VMs concurrently
func (m *Manager) DeleteSnapshots(ctx context.Context, vms []*vm.VM, snapshotName string) error {
	if len(vms) == 0 {
		return fmt.Errorf("no VMs provided")
	}

	m.logger.Infof("Starting bulk snapshot deletion for %d VMs", len(vms))
	m.resetResults()

	// Create worker pool
	jobs := make(chan *vm.VM, len(vms))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.maxWorkers; i++ {
		wg.Add(1)
		go m.deleteSnapshotWorker(ctx, &wg, jobs, snapshotName)
	}

	// Start progress monitor
	m.progressWG.Add(1)
	go m.progressMonitor(len(vms))

	// Send jobs
	for _, vm := range vms {
		select {
		case jobs <- vm:
		case <-ctx.Done():
			close(jobs)
			return ctx.Err()
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(m.resultsChan)

	// Wait for progress monitor to finish processing all results
	m.progressWG.Wait()

	return nil
}

// RollbackSnapshots rolls back multiple VMs to a snapshot concurrently
func (m *Manager) RollbackSnapshots(ctx context.Context, vms []*vm.VM, snapshotName string) error {
	if len(vms) == 0 {
		return fmt.Errorf("no VMs provided")
	}

	m.logger.Infof("Starting bulk snapshot rollback for %d VMs", len(vms))
	m.resetResults()

	// Create worker pool
	jobs := make(chan *vm.VM, len(vms))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.maxWorkers; i++ {
		wg.Add(1)
		go m.rollbackSnapshotWorker(ctx, &wg, jobs, snapshotName)
	}

	// Start progress monitor
	m.progressWG.Add(1)
	go m.progressMonitor(len(vms))

	// Send jobs
	for _, vm := range vms {
		select {
		case jobs <- vm:
		case <-ctx.Done():
			close(jobs)
			return ctx.Err()
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(m.resultsChan)

	// Wait for progress monitor to finish processing all results
	m.progressWG.Wait()

	return nil
}

// StartVMs starts multiple VMs concurrently
func (m *Manager) StartVMs(ctx context.Context, vms []*vm.VM) error {
	if len(vms) == 0 {
		return fmt.Errorf("no VMs provided")
	}

	m.logger.Infof("Starting bulk VM start for %d VMs", len(vms))
	m.resetResults()

	// Create worker pool
	jobs := make(chan *vm.VM, len(vms))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.maxWorkers; i++ {
		wg.Add(1)
		go m.startVMWorker(ctx, &wg, jobs)
	}

	// Start progress monitor
	m.progressWG.Add(1)
	go m.progressMonitor(len(vms))

	// Send jobs
	for _, vm := range vms {
		select {
		case jobs <- vm:
		case <-ctx.Done():
			close(jobs)
			return ctx.Err()
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(m.resultsChan)

	// Wait for progress monitor to finish processing all results
	m.progressWG.Wait()

	return nil
}

// StopVMs stops multiple VMs concurrently
func (m *Manager) StopVMs(ctx context.Context, vms []*vm.VM) error {
	if len(vms) == 0 {
		return fmt.Errorf("no VMs provided")
	}

	m.logger.Infof("Starting bulk VM stop for %d VMs", len(vms))
	m.resetResults()

	// Create worker pool
	jobs := make(chan *vm.VM, len(vms))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < m.maxWorkers; i++ {
		wg.Add(1)
		go m.stopVMWorker(ctx, &wg, jobs)
	}

	// Start progress monitor
	m.progressWG.Add(1)
	go m.progressMonitor(len(vms))

	// Send jobs
	for _, vm := range vms {
		select {
		case jobs <- vm:
		case <-ctx.Done():
			close(jobs)
			return ctx.Err()
		}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()
	close(m.resultsChan)

	// Wait for progress monitor to finish processing all results
	m.progressWG.Wait()

	return nil
}

// Worker functions

func (m *Manager) createSnapshotWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *vm.VM, nameOrPrefix string, useExactName, saveVMState bool) {
	defer wg.Done()

	for vmInstance := range jobs {
		if m.IsCancelled() || ctx.Err() != nil {
			return
		}

		result := OperationResult{
			VMID:      vmInstance.VMID,
			VMName:    vmInstance.Name,
			Operation: "create_snapshot",
			StartTime: time.Now(),
		}

		err := m.snapshotOps.CreateSnapshot(vmInstance.VMID, nameOrPrefix, useExactName, saveVMState)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = err == nil
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else {
			result.Message = "Snapshot created successfully"
		}

		m.resultsChan <- result
	}
}

func (m *Manager) deleteSnapshotWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *vm.VM, snapshotName string) {
	defer wg.Done()

	for vmInstance := range jobs {
		if m.IsCancelled() || ctx.Err() != nil {
			return
		}

		result := OperationResult{
			VMID:      vmInstance.VMID,
			VMName:    vmInstance.Name,
			Operation: "delete_snapshot",
			StartTime: time.Now(),
		}

		err := m.snapshotOps.DeleteSnapshot(vmInstance.VMID, snapshotName)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = err == nil
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else {
			result.Message = fmt.Sprintf("Snapshot '%s' deleted successfully", snapshotName)
		}

		m.resultsChan <- result
	}
}

func (m *Manager) rollbackSnapshotWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *vm.VM, snapshotName string) {
	defer wg.Done()

	for vmInstance := range jobs {
		if m.IsCancelled() || ctx.Err() != nil {
			return
		}

		result := OperationResult{
			VMID:      vmInstance.VMID,
			VMName:    vmInstance.Name,
			Operation: "rollback_snapshot",
			StartTime: time.Now(),
		}

		err := m.snapshotOps.RollbackSnapshot(vmInstance.VMID, snapshotName)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = err == nil
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else {
			result.Message = fmt.Sprintf("Rolled back to snapshot '%s' successfully", snapshotName)
		}

		m.resultsChan <- result
	}
}

func (m *Manager) startVMWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *vm.VM) {
	defer wg.Done()

	for vmInstance := range jobs {
		if m.IsCancelled() || ctx.Err() != nil {
			return
		}

		result := OperationResult{
			VMID:      vmInstance.VMID,
			VMName:    vmInstance.Name,
			Operation: "start_vm",
			StartTime: time.Now(),
		}

		err := m.vmOps.StartVM(vmInstance.VMID)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = err == nil
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else {
			result.Message = "VM started successfully"
		}

		m.resultsChan <- result
	}
}

func (m *Manager) stopVMWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan *vm.VM) {
	defer wg.Done()

	for vmInstance := range jobs {
		if m.IsCancelled() || ctx.Err() != nil {
			return
		}

		result := OperationResult{
			VMID:      vmInstance.VMID,
			VMName:    vmInstance.Name,
			Operation: "stop_vm",
			StartTime: time.Now(),
		}

		err := m.vmOps.StopVM(vmInstance.VMID)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Success = err == nil
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else {
			result.Message = "VM stopped successfully"
		}

		m.resultsChan <- result
	}
}

// progressMonitor monitors progress and collects results
func (m *Manager) progressMonitor(total int) {
	defer m.progressWG.Done()

	for result := range m.resultsChan {
		m.mu.Lock()
		m.results = append(m.results, result)

		completed := len(m.results)
		successful := 0
		failed := 0

		for _, r := range m.results {
			if r.Success {
				successful++
			} else {
				failed++
			}
		}

		progress := float64(completed) / float64(total) * 100
		m.mu.Unlock()

		// Send progress update
		update := ProgressUpdate{
			Completed:  completed,
			Total:      total,
			Successful: successful,
			Failed:     failed,
			Current:    fmt.Sprintf("VM %s", result.VMID),
			Progress:   progress,
		}

		select {
		case m.progressChan <- update:
		default:
			// Progress channel full, skip
		}

		// Log result
		if result.Success {
			m.logger.Infof("✅ VM %s (%s): %s (%.2fs)", result.VMID, result.VMName, result.Message, result.Duration.Seconds())
		} else {
			m.logger.Errorf("❌ VM %s (%s): %s (%.2fs)", result.VMID, result.VMName, result.Message, result.Duration.Seconds())
		}
	}
}

// GetProgressChan returns the progress channel for real-time updates
func (m *Manager) GetProgressChan() <-chan ProgressUpdate {
	return m.progressChan
}

// PrintSummary prints a summary of the bulk operation results
func (m *Manager) PrintSummary() {
	results := m.GetResults()
	if len(results) == 0 {
		fmt.Println("No operations completed.")
		return
	}

	// Sort results by VM ID for consistent display
	sort.Slice(results, func(i, j int) bool {
		return results[i].VMID < results[j].VMID
	})

	completed, successful, failed, _ := m.GetProgress()

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("BULK OPERATION SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	fmt.Printf("Total Operations: %d\n", completed)
	fmt.Printf("Successful: %d (%.1f%%)\n", successful, float64(successful)/float64(completed)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", failed, float64(failed)/float64(completed)*100)

	if failed > 0 {
		fmt.Printf("\nFAILED OPERATIONS:\n")
		fmt.Printf(strings.Repeat("-", 30) + "\n")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("VM %s (%s): %s\n", result.VMID, result.VMName, result.Message)
			}
		}
	}

	if successful > 0 {
		fmt.Printf("\nSUCCESSFUL OPERATIONS:\n")
		fmt.Printf(strings.Repeat("-", 30) + "\n")
		for _, result := range results {
			if result.Success {
				fmt.Printf("VM %s (%s): %s (%.2fs)\n", result.VMID, result.VMName, result.Message, result.Duration.Seconds())
			}
		}
	}

	fmt.Printf(strings.Repeat("=", 60) + "\n")
}

// resetResults clears previous results
func (m *Manager) resetResults() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = make([]OperationResult, 0)
	m.cancelled = false

	// Drain old channels if they exist
	for len(m.resultsChan) > 0 {
		<-m.resultsChan
	}
	for len(m.progressChan) > 0 {
		<-m.progressChan
	}

	// Reset the WaitGroup to ensure clean state
	m.progressWG = sync.WaitGroup{}
}
