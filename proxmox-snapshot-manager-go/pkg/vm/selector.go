package vm

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// Selector handles VM selection and filtering logic
type Selector struct {
	vmOps  *Operations
	logger *logrus.Logger
}

// NewSelector creates a new VM selector
func NewSelector(vmOps *Operations, logger *logrus.Logger) *Selector {
	if logger == nil {
		logger = logrus.New()
	}

	return &Selector{
		vmOps:  vmOps,
		logger: logger,
	}
}

// SelectionResult represents the result of VM selection
type SelectionResult struct {
	VMs     []*VM
	Matched []string
	Failed  []string
}

// FindVMByNameOrID finds a VM by either VM ID or VM name from the provided VM list
func (s *Selector) FindVMByNameOrID(identifier string, allVMs []*VM) string {
	// First, try direct VM ID match
	for _, vm := range allVMs {
		if vm.VMID == identifier {
			return identifier
		}
	}

	// Try exact name match (case-insensitive)
	for _, vm := range allVMs {
		if strings.EqualFold(vm.Name, identifier) {
			return vm.VMID
		}
	}

	// Try partial name matching (case-insensitive)
	var matches []*VM
	lowerIdentifier := strings.ToLower(identifier)
	for _, vm := range allVMs {
		if strings.Contains(strings.ToLower(vm.Name), lowerIdentifier) {
			matches = append(matches, vm)
		}
	}

	// If exactly one partial match, return it
	if len(matches) == 1 {
		return matches[0].VMID
	} else if len(matches) > 1 {
		// Multiple matches - show them
		s.logger.Warnf("Multiple VMs match '%s':", identifier)
		for _, vm := range matches {
			status := "🟢 running"
			if !vm.Running {
				status = "🔴 stopped"
			}
			s.logger.Warnf("  - VM %s: %s (%s)", vm.VMID, vm.Name, status)
		}
		s.logger.Warn("Please be more specific.")
		return ""
	}

	return ""
}

// ParseVMSelection parses various VM selection formats and returns VM IDs
func (s *Selector) ParseVMSelection(selection string, allVMs []*VM) (*SelectionResult, error) {
	result := &SelectionResult{
		VMs:     []*VM{},
		Matched: []string{},
		Failed:  []string{},
	}

	// Handle special keywords
	switch strings.ToLower(selection) {
	case "all":
		result.VMs = allVMs
		for _, vm := range allVMs {
			result.Matched = append(result.Matched, vm.VMID)
		}
		return result, nil
	case "running":
		for _, vm := range allVMs {
			if vm.Running {
				result.VMs = append(result.VMs, vm)
				result.Matched = append(result.Matched, vm.VMID)
			}
		}
		return result, nil
	case "stopped":
		for _, vm := range allVMs {
			if !vm.Running {
				result.VMs = append(result.VMs, vm)
				result.Matched = append(result.Matched, vm.VMID)
			}
		}
		return result, nil
	}

	// Handle wildcard patterns (e.g., "72*", "*web*")
	if strings.Contains(selection, "*") {
		return s.parseWildcardPattern(selection, allVMs)
	}

	// Handle ranges (e.g., "7201-7205")
	if strings.Contains(selection, "-") && !strings.Contains(selection, ",") {
		return s.parseRange(selection, allVMs)
	}

	// Handle comma-separated lists (e.g., "7201,7203,7205" or "web01,web02")
	if strings.Contains(selection, ",") {
		return s.parseCommaSeparated(selection, allVMs)
	}

	// Handle single VM ID or name
	vmid := s.FindVMByNameOrID(selection, allVMs)
	if vmid != "" {
		for _, vm := range allVMs {
			if vm.VMID == vmid {
				result.VMs = []*VM{vm}
				result.Matched = []string{vmid}
				return result, nil
			}
		}
	}

	result.Failed = []string{selection}
	return result, fmt.Errorf("VM '%s' not found", selection)
}

// parseWildcardPattern handles wildcard patterns like "72*" or "*web*"
func (s *Selector) parseWildcardPattern(pattern string, allVMs []*VM) (*SelectionResult, error) {
	result := &SelectionResult{
		VMs:     []*VM{},
		Matched: []string{},
		Failed:  []string{},
	}

	// Convert wildcard pattern to regex
	regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*")
	regex, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return result, fmt.Errorf("invalid wildcard pattern '%s': %w", pattern, err)
	}

	for _, vm := range allVMs {
		// Check against VM ID
		if regex.MatchString(vm.VMID) {
			result.VMs = append(result.VMs, vm)
			result.Matched = append(result.Matched, vm.VMID)
			continue
		}

		// Check against VM name
		if regex.MatchString(vm.Name) {
			result.VMs = append(result.VMs, vm)
			result.Matched = append(result.Matched, vm.VMID)
		}
	}

	if len(result.VMs) == 0 {
		result.Failed = []string{pattern}
		return result, fmt.Errorf("no VMs match pattern '%s'", pattern)
	}

	return result, nil
}

// parseRange handles range patterns like "7201-7205"
func (s *Selector) parseRange(rangeStr string, allVMs []*VM) (*SelectionResult, error) {
	result := &SelectionResult{
		VMs:     []*VM{},
		Matched: []string{},
		Failed:  []string{},
	}

	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return result, fmt.Errorf("invalid range format '%s': expected 'start-end'", rangeStr)
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return result, fmt.Errorf("invalid range start '%s': %w", parts[0], err)
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return result, fmt.Errorf("invalid range end '%s': %w", parts[1], err)
	}

	if start > end {
		return result, fmt.Errorf("invalid range '%s': start must be <= end", rangeStr)
	}

	for _, vm := range allVMs {
		if vmidNum, err := strconv.Atoi(vm.VMID); err == nil {
			if vmidNum >= start && vmidNum <= end {
				result.VMs = append(result.VMs, vm)
				result.Matched = append(result.Matched, vm.VMID)
			}
		}
	}

	if len(result.VMs) == 0 {
		result.Failed = []string{rangeStr}
		return result, fmt.Errorf("no VMs found in range '%s'", rangeStr)
	}

	return result, nil
}

// parseCommaSeparated handles comma-separated lists like "7201,7203,web01"
func (s *Selector) parseCommaSeparated(listStr string, allVMs []*VM) (*SelectionResult, error) {
	result := &SelectionResult{
		VMs:     []*VM{},
		Matched: []string{},
		Failed:  []string{},
	}

	items := strings.Split(listStr, ",")
	vmidSet := make(map[string]bool) // To avoid duplicates

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		vmid := s.FindVMByNameOrID(item, allVMs)
		if vmid != "" && !vmidSet[vmid] {
			for _, vm := range allVMs {
				if vm.VMID == vmid {
					result.VMs = append(result.VMs, vm)
					result.Matched = append(result.Matched, vmid)
					vmidSet[vmid] = true
					break
				}
			}
		} else if vmid == "" {
			result.Failed = append(result.Failed, item)
		}
	}

	if len(result.Failed) > 0 {
		return result, fmt.Errorf("VMs not found: %s", strings.Join(result.Failed, ", "))
	}

	return result, nil
}

// InteractiveSelect provides an interactive VM selection interface
func (s *Selector) InteractiveSelect(allVMs []*VM, prompt string) ([]*VM, error) {
	if len(allVMs) == 0 {
		return nil, fmt.Errorf("no VMs available")
	}

	fmt.Printf("\n%s\n", prompt)
	fmt.Println("Available VMs:")
	fmt.Println("============")

	// Sort VMs by ID for consistent display
	sortedVMs := make([]*VM, len(allVMs))
	copy(sortedVMs, allVMs)
	sort.Slice(sortedVMs, func(i, j int) bool {
		return sortedVMs[i].VMID < sortedVMs[j].VMID
	})

	for i, vm := range sortedVMs {
		status := "🟢 running"
		if !vm.Running {
			status = "🔴 stopped"
		}
		fmt.Printf("%3d. VM %-4s: %-20s (%s)\n", i+1, vm.VMID, vm.Name, status)
	}

	fmt.Println("\nSelection options:")
	fmt.Println("  • Enter numbers (e.g., '1,3,5' or '1-5')")
	fmt.Println("  • Enter VM IDs (e.g., '7201,7203')")
	fmt.Println("  • Enter VM names (e.g., 'web01,db01')")
	fmt.Println("  • Use wildcards (e.g., '72*' or '*web*')")
	fmt.Println("  • Use keywords: 'all', 'running', 'stopped'")
	fmt.Print("\nSelect VMs: ")

	var selection string
	fmt.Scanln(&selection)

	// Handle numeric selection (menu indices)
	if isNumericSelection(selection) {
		return s.parseNumericSelection(selection, sortedVMs)
	}

	// Handle other selection formats
	result, err := s.ParseVMSelection(selection, allVMs)
	if err != nil {
		return nil, err
	}

	return result.VMs, nil
}

// isNumericSelection checks if the selection contains only numbers, commas, and dashes
func isNumericSelection(selection string) bool {
	for _, char := range selection {
		if char != '0' && char != '1' && char != '2' && char != '3' && char != '4' &&
			char != '5' && char != '6' && char != '7' && char != '8' && char != '9' &&
			char != ',' && char != '-' && char != ' ' {
			return false
		}
	}
	return true
}

// parseNumericSelection parses numeric menu selections like "1,3,5" or "1-5"
func (s *Selector) parseNumericSelection(selection string, sortedVMs []*VM) ([]*VM, error) {
	var selectedVMs []*VM
	selectedSet := make(map[string]bool)

	// Handle ranges and individual numbers
	parts := strings.Split(selection, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Handle range like "1-5"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start number: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end number: %s", rangeParts[1])
			}

			for i := start; i <= end; i++ {
				if i < 1 || i > len(sortedVMs) {
					return nil, fmt.Errorf("number %d is out of range (1-%d)", i, len(sortedVMs))
				}
				vm := sortedVMs[i-1]
				if !selectedSet[vm.VMID] {
					selectedVMs = append(selectedVMs, vm)
					selectedSet[vm.VMID] = true
				}
			}
		} else {
			// Handle individual number
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}

			if num < 1 || num > len(sortedVMs) {
				return nil, fmt.Errorf("number %d is out of range (1-%d)", num, len(sortedVMs))
			}

			vm := sortedVMs[num-1]
			if !selectedSet[vm.VMID] {
				selectedVMs = append(selectedVMs, vm)
				selectedSet[vm.VMID] = true
			}
		}
	}

	return selectedVMs, nil
}

// DisplayVMInfo displays information about VMs
func (s *Selector) DisplayVMInfo(vms []*VM) {
	if len(vms) == 0 {
		fmt.Println("No VMs found.")
		return
	}

	fmt.Printf("\nFound %d VM(s):\n", len(vms))
	fmt.Println("================")

	for _, vm := range vms {
		status := "🟢 running"
		if !vm.Running {
			status = "🔴 stopped"
		}

		fmt.Printf("VM %s: %s\n", vm.VMID, vm.Name)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Node: %s\n", vm.Node)
		if vm.Memory > 0 {
			fmt.Printf("  Memory: %d MB\n", vm.Memory)
		}
		if vm.CPUs > 0 {
			fmt.Printf("  CPUs: %d\n", vm.CPUs)
		}
		fmt.Println()
	}
}
