#!/usr/bin/env python3

"""
Snapshot Integration Module
Bridge module that integrates with the existing proxmox-snapshot-manager
to provide unified snapshot and VM management capabilities
"""

import os
import sys
from typing import List, Dict, Optional

# Add the snapshot manager to the path for imports
SNAPSHOT_MANAGER_PATH = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'proxmox-snapshot-manager')
if os.path.exists(SNAPSHOT_MANAGER_PATH):
    sys.path.insert(0, SNAPSHOT_MANAGER_PATH)

try:
    from snapshot_operations import SnapshotOperations
    from bulk_operations import BulkSnapshotOperations
    SNAPSHOT_MANAGER_AVAILABLE = True
except ImportError:
    SNAPSHOT_MANAGER_AVAILABLE = False
    # Warning will be shown only when snapshot functionality is actually needed


class SnapshotIntegration:
    """Integrates VM manager with existing snapshot management functionality."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
        self.snapshot_ops = None
        self.bulk_snapshot_ops = None
        
        if SNAPSHOT_MANAGER_AVAILABLE:
            self.snapshot_ops = SnapshotOperations(api_client, vm_manager)
            self.bulk_snapshot_ops = BulkSnapshotOperations(api_client, vm_manager)
    
    def is_available(self) -> bool:
        """Check if snapshot integration is available."""
        return SNAPSHOT_MANAGER_AVAILABLE and self.snapshot_ops is not None
    
    def get_snapshots(self, vmid: str) -> List[Dict]:
        """Get list of snapshots for a VM."""
        if not self.is_available():
            return []
        return self.snapshot_ops.get_snapshots(vmid)
    
    def get_snapshot_config(self, vmid: str, snapshot_name: str) -> Dict:
        """Get configuration of a specific snapshot."""
        if not self.is_available():
            return {}
        return self.snapshot_ops.get_snapshot_config(vmid, snapshot_name)
    
    def create_snapshot(self, vmid: str, name_or_prefix: str, use_exact_name: bool = False, save_vmstate: bool = False) -> bool:
        """Create a snapshot for a VM."""
        if not self.is_available():
            print("❌ Snapshot functionality not available")
            return False
        return self.snapshot_ops.create_snapshot(vmid, name_or_prefix, use_exact_name, save_vmstate)
    
    def create_snapshot_silent(self, vmid: str, prefix: str, save_vmstate: bool = True) -> bool:
        """Create a snapshot without output (for bulk operations)."""
        if not self.is_available():
            return False
        return self.snapshot_ops.create_snapshot_silent(vmid, prefix, save_vmstate)
    
    def rollback_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        """Rollback VM to a specific snapshot."""
        if not self.is_available():
            print("❌ Snapshot functionality not available")
            return False
        return self.snapshot_ops.rollback_snapshot(vmid, snapshot_name)
    
    def delete_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        """Delete a specific snapshot."""
        if not self.is_available():
            print("❌ Snapshot functionality not available")
            return False
        return self.snapshot_ops.delete_snapshot(vmid, snapshot_name)
    
    def delete_snapshot_silent(self, vmid: str, snapshot_name: str) -> bool:
        """Delete a snapshot without output (for bulk operations)."""
        if not self.is_available():
            return False
        return self.snapshot_ops.delete_snapshot_silent(vmid, snapshot_name)
    
    def list_snapshots(self, vmid: str):
        """List all snapshots for a VM in a formatted table."""
        if not self.is_available():
            print("❌ Snapshot functionality not available")
            return
        self.snapshot_ops.list_snapshots(vmid)
    
    def delete_all_snapshots(self, vmid: str, snapshots: List[Dict]):
        """Delete all snapshots for a VM."""
        if not self.is_available():
            print("❌ Snapshot functionality not available")
            return
        self.snapshot_ops.delete_all_snapshots(vmid, snapshots)
    
    def bulk_create_snapshots(self, vm_ids: List[str], prefix: str, max_workers: int = 2):
        """Create snapshots for multiple VMs concurrently."""
        if not self.is_available():
            print("❌ Bulk snapshot functionality not available")
            return None
        return self.bulk_snapshot_ops.bulk_create_snapshots(vm_ids, prefix, max_workers)
    
    def bulk_delete_snapshots(self, snapshots_by_vm: Dict[str, List[str]], max_workers: int = 2):
        """Delete multiple snapshots concurrently."""
        if not self.is_available():
            print("❌ Bulk snapshot functionality not available")
            return None
        return self.bulk_snapshot_ops.bulk_delete_snapshots(snapshots_by_vm, max_workers)
    
    def display_snapshot_menu(self):
        """Display snapshot management menu."""
        if not self.is_available():
            print("⚠️  Warning: Snapshot manager modules not found. Snapshot functionality will be limited.")
            print("❌ Snapshot functionality not available")
            print("To enable snapshot features, ensure the proxmox-snapshot-manager is properly installed.")
            input("Press Enter to continue...")
            return
        
        while True:
            try:
                print("\n📸 Snapshot Management")
                print("=" * 40)
                print("1. List VM Snapshots")
                print("2. Create Snapshot")
                print("3. Rollback to Snapshot")
                print("4. Delete Snapshot")
                print("5. Bulk Create Snapshots")
                print("6. Bulk Delete Snapshots")
                print("7. Back to Main Menu")
                print()
                
                choice = input("Select option (1-7): ").strip()
                
                if choice == '1':
                    self.handle_list_snapshots()
                elif choice == '2':
                    self.handle_create_snapshot()
                elif choice == '3':
                    self.handle_rollback_snapshot()
                elif choice == '4':
                    self.handle_delete_snapshot()
                elif choice == '5':
                    self.handle_bulk_create_snapshots()
                elif choice == '6':
                    self.handle_bulk_delete_snapshots()
                elif choice == '7':
                    break
                else:
                    print("❌ Invalid choice. Please select 1-7")
                    
            except KeyboardInterrupt:
                print("\nReturning to main menu...")
                break
    
    def handle_list_snapshots(self):
        """Handle snapshot listing."""
        print("\n📸 List VM Snapshots")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.vm_manager.get_all_vms_info()
        vm_id = self.vm_manager.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if vm_id:
            self.list_snapshots(vm_id)
        else:
            print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_create_snapshot(self):
        """Handle snapshot creation."""
        print("\n📸 Create Snapshot")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.vm_manager.get_all_vms_info()
        vm_id = self.vm_manager.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # Get snapshot name/prefix
        print("\nSnapshot naming options:")
        print("1. Use prefix (recommended) - combines with VM name and timestamp")
        print("2. Use exact name - specify the complete snapshot name")
        
        naming_choice = input("Select naming option (1-2): ").strip()
        
        if naming_choice == '2':
            snapshot_name = input("Enter exact snapshot name: ").strip()
            if not snapshot_name:
                return
            use_exact_name = True
            name_input = snapshot_name
        else:
            prefix = input("Enter snapshot prefix (e.g., 'backup', 'pre-update'): ").strip()
            if not prefix:
                return
            use_exact_name = False
            name_input = prefix
        
        # VM state option
        vm_info = self.vm_manager.get_vm_info(vm_id)
        if vm_info and vm_info.get('running', False):
            include_vmstate = input("Include VM state (RAM)? This ensures perfect consistency but takes longer (y/N): ").strip().lower()
            save_vmstate = include_vmstate in ['y', 'yes']
        else:
            save_vmstate = False
            print("ℹ️  VM is not running - VM state will not be included")
        
        # Create snapshot
        self.create_snapshot(vm_id, name_input, use_exact_name, save_vmstate)
        
        input("Press Enter to continue...")
    
    def handle_rollback_snapshot(self):
        """Handle snapshot rollback."""
        print("\n⏪ Rollback to Snapshot")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.vm_manager.get_all_vms_info()
        vm_id = self.vm_manager.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # List available snapshots
        snapshots = self.get_snapshots(vm_id)
        available_snapshots = [s for s in snapshots if s.get('name') != 'current']
        
        if not available_snapshots:
            print(f"❌ No snapshots found for VM {vm_id}")
            input("Press Enter to continue...")
            return
        
        # Display snapshots
        print(f"\nAvailable snapshots for VM {vm_id}:")
        print("-" * 60)
        for i, snapshot in enumerate(available_snapshots, 1):
            name = snapshot.get('name', 'Unknown')
            desc = snapshot.get('description', 'No description')
            print(f"{i:2d}. {name}")
            if desc:
                print(f"    {desc}")
        print("-" * 60)
        
        # Select snapshot
        try:
            choice = input(f"Select snapshot to rollback to (1-{len(available_snapshots)}, or 'q' to quit): ").strip().lower()
            
            if choice == 'q':
                return
            
            snapshot_num = int(choice)
            if 1 <= snapshot_num <= len(available_snapshots):
                selected_snapshot = available_snapshots[snapshot_num - 1]
                snapshot_name = selected_snapshot['name']
                
                # Confirm rollback
                print(f"\n⚠️  ROLLBACK WARNING")
                print("=" * 40)
                print("This will:")
                print(f"  • Revert VM {vm_id} to snapshot '{snapshot_name}'")
                print("  • Permanently lose all changes made after this snapshot")
                print("  • This action cannot be undone!")
                print("=" * 40)
                
                confirm = input("Type 'ROLLBACK' to confirm this operation: ").strip()
                if confirm != 'ROLLBACK':
                    print("Rollback operation cancelled")
                    input("Press Enter to continue...")
                    return
                
                # Perform rollback
                self.rollback_snapshot(vm_id, snapshot_name)
            else:
                print(f"❌ Invalid choice. Please select 1-{len(available_snapshots)}")
        except ValueError:
            print("❌ Invalid input. Please enter a number")
        
        input("Press Enter to continue...")
    
    def handle_delete_snapshot(self):
        """Handle snapshot deletion."""
        print("\n🗑️  Delete Snapshot")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.vm_manager.get_all_vms_info()
        vm_id = self.vm_manager.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # List available snapshots
        snapshots = self.get_snapshots(vm_id)
        available_snapshots = [s for s in snapshots if s.get('name') != 'current']
        
        if not available_snapshots:
            print(f"❌ No snapshots found for VM {vm_id}")
            input("Press Enter to continue...")
            return
        
        # Display snapshots
        print(f"\nAvailable snapshots for VM {vm_id}:")
        print("-" * 60)
        for i, snapshot in enumerate(available_snapshots, 1):
            name = snapshot.get('name', 'Unknown')
            desc = snapshot.get('description', 'No description')
            print(f"{i:2d}. {name}")
            if desc:
                print(f"    {desc[:50]}{'...' if len(desc) > 50 else ''}")
        print("-" * 60)
        
        # Select snapshot
        try:
            choice = input(f"Select snapshot to delete (1-{len(available_snapshots)}, 'all' for all snapshots, or 'q' to quit): ").strip().lower()
            
            if choice == 'q':
                return
            elif choice == 'all':
                # Delete all snapshots
                self.delete_all_snapshots(vm_id, available_snapshots)
            else:
                snapshot_num = int(choice)
                if 1 <= snapshot_num <= len(available_snapshots):
                    selected_snapshot = available_snapshots[snapshot_num - 1]
                    snapshot_name = selected_snapshot['name']
                    
                    # Confirm deletion
                    confirm = input(f"Delete snapshot '{snapshot_name}'? (y/N): ").strip().lower()
                    if confirm in ['y', 'yes']:
                        self.delete_snapshot(vm_id, snapshot_name)
                    else:
                        print("Deletion cancelled")
                else:
                    print(f"❌ Invalid choice. Please select 1-{len(available_snapshots)}")
        except ValueError:
            print("❌ Invalid input. Please enter a number or 'all'")
        
        input("Press Enter to continue...")
    
    def handle_bulk_create_snapshots(self):
        """Handle bulk snapshot creation."""
        print("\n📸 Bulk Create Snapshots")
        print("=" * 40)
        
        all_vms = self.vm_manager.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_manager.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_manager.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        # Get snapshot prefix
        prefix = input("Enter snapshot prefix (e.g., 'backup', 'pre-update'): ").strip()
        if not prefix:
            return
        
        print(f"\nSelected VMs: {', '.join(vm_ids)}")
        print(f"Snapshot prefix: {prefix}")
        confirm = input(f"Create snapshots for {len(vm_ids)} VMs? (y/N): ").strip().lower()
        
        if confirm in ['y', 'yes']:
            self.bulk_create_snapshots(vm_ids, prefix)
        else:
            print("Operation cancelled")
        
        input("Press Enter to continue...")
    
    def handle_bulk_delete_snapshots(self):
        """Handle bulk snapshot deletion."""
        print("\n🗑️  Bulk Delete Snapshots")
        print("=" * 40)
        
        all_vms = self.vm_manager.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_manager.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_manager.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        # Get snapshot pattern or name
        pattern = input("Enter snapshot name pattern (e.g., 'backup-*', exact name, or 'all' for all snapshots): ").strip()
        if not pattern:
            return
        
        # Collect snapshots to delete
        snapshots_by_vm = {}
        total_snapshots = 0
        
        for vm_id in vm_ids:
            snapshots = self.get_snapshots(vm_id)
            available_snapshots = [s for s in snapshots if s.get('name') != 'current']
            
            if pattern.lower() == 'all':
                snapshots_to_delete = [s['name'] for s in available_snapshots]
            elif '*' in pattern:
                # Pattern matching
                import fnmatch
                snapshots_to_delete = [s['name'] for s in available_snapshots if fnmatch.fnmatch(s['name'], pattern)]
            else:
                # Exact match
                snapshots_to_delete = [s['name'] for s in available_snapshots if s['name'] == pattern]
            
            if snapshots_to_delete:
                snapshots_by_vm[vm_id] = snapshots_to_delete
                total_snapshots += len(snapshots_to_delete)
        
        if not snapshots_by_vm:
            print("❌ No matching snapshots found")
            input("Press Enter to continue...")
            return
        
        # Display summary
        print(f"\nSnapshots to delete:")
        for vm_id, snapshot_names in snapshots_by_vm.items():
            print(f"  VM {vm_id}: {len(snapshot_names)} snapshot(s)")
            for name in snapshot_names[:3]:  # Show first 3
                print(f"    - {name}")
            if len(snapshot_names) > 3:
                print(f"    ... and {len(snapshot_names) - 3} more")
        
        print(f"\nTotal: {total_snapshots} snapshots across {len(snapshots_by_vm)} VMs")
        
        # Confirm deletion
        if pattern.lower() == 'all':
            confirm_text = "DELETE ALL"
            confirm = input(f"⚠️  This will delete ALL snapshots! Type '{confirm_text}' to confirm: ").strip()
            if confirm != confirm_text:
                print("Operation cancelled")
                input("Press Enter to continue...")
                return
        else:
            confirm = input(f"Delete {total_snapshots} snapshots? (y/N): ").strip().lower()
            if confirm not in ['y', 'yes']:
                print("Operation cancelled")
                input("Press Enter to continue...")
                return
        
        # Perform bulk deletion
        self.bulk_delete_snapshots(snapshots_by_vm)
        
        input("Press Enter to continue...")