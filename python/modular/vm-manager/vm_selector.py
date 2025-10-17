#!/usr/bin/env python3

"""
VM Selection Module
Handles VM selection parsing and filtering with flexible selection methods
"""

import re
from typing import List, Dict, Optional, Set


class VMSelector:
    """Handles VM selection parsing and filtering."""
    
    def __init__(self, vm_manager):
        self.vm_manager = vm_manager
    
    def find_vm_by_name_or_id(self, identifier: str, all_vms: List[Dict]) -> Optional[str]:
        """Find VM ID by either VM ID or VM name from the provided VM list."""
        # First, try direct VM ID match
        if any(vm['vmid'] == identifier for vm in all_vms):
            return identifier
        
        # Try exact name match (case-insensitive)
        for vm in all_vms:
            vm_name = vm.get('name', '')
            if vm_name.lower() == identifier.lower():
                return vm['vmid']
        
        # Try partial name matching (case-insensitive)
        matches = []
        for vm in all_vms:
            vm_name = vm.get('name', '')
            if identifier.lower() in vm_name.lower():
                matches.append(vm)
        
        # If exactly one partial match, return it
        if len(matches) == 1:
            return matches[0]['vmid']
        elif len(matches) > 1:
            # Multiple matches - show them and return None
            print(f"⚠️  Multiple VMs match '{identifier}':")
            for vm in matches:
                status = "🟢 running" if vm.get('running', False) else "🔴 stopped"
                print(f"  - VM {vm['vmid']}: {vm['name']} ({status})")
            print("Please be more specific.")
            return None
        
        return None
    
    def resolve_vm_identifiers(self, identifiers: List[str], all_vms: List[Dict]) -> List[str]:
        """Resolve a list of VM identifiers (IDs or names) to VM IDs."""
        resolved_ids = []
        failed_lookups = []
        
        for identifier in identifiers:
            vm_id = self.find_vm_by_name_or_id(identifier, all_vms)
            if vm_id:
                if vm_id not in resolved_ids:  # Avoid duplicates
                    resolved_ids.append(vm_id)
            else:
                failed_lookups.append(identifier)
        
        if failed_lookups:
            print(f"⚠️  Could not find VMs: {', '.join(failed_lookups)}")
        
        return resolved_ids
    
    def parse_selection(self, selection: str, all_vms: List[Dict]) -> List[str]:
        """Parse VM selection string and return list of VM IDs."""
        selection = selection.strip()
        
        # Handle special keywords
        if selection.lower() == '*' or selection.lower() == 'all':
            return [vm['vmid'] for vm in all_vms]
        elif selection.lower() == 'running':
            return [vm['vmid'] for vm in all_vms if vm.get('running', False)]
        elif selection.lower() == 'stopped':
            return [vm['vmid'] for vm in all_vms if not vm.get('running', False)]
        elif selection.lower() == 'i' or selection.lower() == 'interactive':
            return self.interactive_selection(all_vms)
        
        # Handle range selection (e.g., "7201-7205")
        if '-' in selection and len(selection.split('-')) == 2:
            try:
                start, end = selection.split('-')
                start_id, end_id = int(start.strip()), int(end.strip())
                vm_ids = []
                for vm in all_vms:
                    vm_id = int(vm['vmid'])
                    if start_id <= vm_id <= end_id:
                        vm_ids.append(vm['vmid'])
                return vm_ids
            except ValueError:
                pass
        
        # Handle comma-separated list (e.g., "7201,7203,smtp01,workstation03")
        if ',' in selection:
            identifiers = [item.strip() for item in selection.split(',') if item.strip()]
            return self.resolve_vm_identifiers(identifiers, all_vms)
        
        # Handle pattern matching (e.g., "72*", "smtp*", "*workstation*")
        if '*' in selection:
            pattern = selection.replace('*', '.*')
            vm_ids = []
            for vm in all_vms:
                # Check against VM ID
                if re.match(pattern, vm['vmid']):
                    vm_ids.append(vm['vmid'])
                # Also check against VM name
                elif re.match(pattern, vm.get('name', ''), re.IGNORECASE):
                    vm_ids.append(vm['vmid'])
            return vm_ids
        
        # Handle single VM ID or name
        vm_id = self.find_vm_by_name_or_id(selection, all_vms)
        if vm_id:
            return [vm_id]
        
        return []
    
    def interactive_selection(self, all_vms: List[Dict]) -> List[str]:
        """Interactive checkbox-style VM selection."""
        if not all_vms:
            print("No VMs available for selection")
            return []
        
        print("\n📋 Interactive VM Selection")
        print("=" * 50)
        print("Enter VM numbers to toggle selection (space-separated)")
        print("Commands: 'all' (select all), 'none' (clear all), 'done' (finish)")
        print()
        
        selected_vms: Set[str] = set()
        
        while True:
            # Display VMs with selection status
            print("\nAvailable VMs:")
            print(f"{'#':<3} {'✓':<3} {'VM ID':<8} {'Name':<25} {'Status':<15}")
            print("-" * 60)
            
            for i, vm in enumerate(all_vms, 1):
                selected = "✓" if vm['vmid'] in selected_vms else " "
                status = "🟢 running" if vm.get('running', False) else "🔴 stopped"
                name = vm.get('name', 'Unknown')[:24]
                print(f"{i:<3} {selected:<3} {vm['vmid']:<8} {name:<25} {status:<15}")
            
            print(f"\nSelected: {len(selected_vms)} VMs")
            choice = input("\nEnter selection (numbers, 'all', 'none', 'done'): ").strip().lower()
            
            if choice == 'done':
                break
            elif choice == 'all':
                selected_vms = {vm['vmid'] for vm in all_vms}
                print(f"✅ Selected all {len(selected_vms)} VMs")
            elif choice == 'none':
                selected_vms.clear()
                print("✅ Cleared all selections")
            else:
                # Parse numbers
                for num_str in choice.split():
                    try:
                        num = int(num_str)
                        if 1 <= num <= len(all_vms):
                            vm_id = all_vms[num-1]['vmid']
                            if vm_id in selected_vms:
                                selected_vms.remove(vm_id)
                                print(f"✅ Deselected VM {vm_id}")
                            else:
                                selected_vms.add(vm_id)
                                print(f"✅ Selected VM {vm_id}")
                        else:
                            print(f"⚠️ Invalid number: {num}")
                    except ValueError:
                        print(f"⚠️ Invalid input: {num_str}")
        
        return list(selected_vms)
    
    def display_selection_help(self):
        """Display help for VM selection formats."""
        print("\n📚 VM Selection Help")
        print("=" * 40)
        print("Selection formats:")
        print("  *                    - All VMs")
        print("  running              - All running VMs")
        print("  stopped              - All stopped VMs")
        print("  7201-7205            - Range of VM IDs")
        print("  7201,7203            - VM IDs (comma-separated)")
        print("  smtp01,workstation03 - VM names (comma-separated)")
        print("  7201,smtp01          - Mixed IDs and names")
        print("  72*                  - Pattern matching VM IDs")
        print("  smtp*                - Pattern matching VM names")
        print("  *workstation*        - Pattern matching (contains)")
        print("  i                    - Interactive selection")
        print("  7201                 - Single VM ID")
        print("  xsf-dev-smtp01       - Single VM name")
        print("  smtp01               - Partial VM name")
        print()
        print("Examples with your VMs:")
        print("  centos*              - All CentOS VMs (751,752,753)")
        print("  *smtp*               - All SMTP VMs (7204,7206)")
        print("  workstation*         - All workstation VMs")
        print("  7201,smtp01,tacacs01 - Mixed selection")
        print()