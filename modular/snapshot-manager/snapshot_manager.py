#!/usr/bin/env python3

"""
Main Snapshot Manager Module
Orchestrates all VM and snapshot operations with an integrated management interface
"""

import os
import sys
from datetime import datetime
from typing import List, Dict, Optional

from proxmox_api import ProxmoxAPI, ProxmoxAPIError
from vm_operations import VMOperations
from vm_selector import VMSelector
from snapshot_operations import SnapshotOperations
from bulk_operations import BulkSnapshotOperations, BulkVMOperations


class ProxmoxSnapshotManager:
    """Main manager class that orchestrates all Proxmox VM snapshot operations."""
    
    def __init__(self):
        self.api = None
        self.vm_ops = None
        self.snapshot_ops = None
        self.bulk_snapshot_ops = None
        self.bulk_vm_ops = None
        self.vm_selector = None
        self.save_vmstate = False
    
    def display_usage(self):
        """Display usage information."""
        usage_text = """
Proxmox VM Snapshot Management Tool (API Version)
================================================

This tool provides comprehensive VM snapshot management capabilities:
- Create snapshots with intelligent naming
- Rollback to previous snapshots
- List and manage existing snapshots
- Delete snapshots with safety checks
- Bulk snapshot operations
- Real-time task monitoring

API Authentication Options:
  1. Username/Password (prompted)
  2. API Token (set environment variables):
     export PVE_HOST=your-proxmox-host
     export PVE_USER=username@realm
     export PVE_TOKEN_NAME=token-name
     export PVE_TOKEN_VALUE=token-value

Usage: python3 pve_snapshot_manager.py
"""
        print(usage_text)
    
    def connect_to_proxmox(self, batch_mode: bool = False) -> bool:
        """Establish connection to Proxmox API and initialize components."""
        # Try environment variables first
        host = os.getenv('PVE_HOST')
        user = os.getenv('PVE_USER') 
        token_name = os.getenv('PVE_TOKEN_NAME')
        token_value = os.getenv('PVE_TOKEN_VALUE')
        
        # Debug: Show what environment variables are found (only if not in batch mode)
        if not batch_mode:
            print("🔍 Checking environment variables...")
            print(f"   PVE_HOST: {'✅ Set' if host else '⚠️ Not set'}")
            print(f"   PVE_USER: {'✅ Set' if user else '⚠️ Not set'}")
            print(f"   PVE_TOKEN_NAME: {'✅ Set' if token_name else '⚠️ Not set'}")
            print(f"   PVE_TOKEN_VALUE: {'✅ Set' if token_value else '⚠️ Not set'}")
        
        if host and user and token_name and token_value:
            if not batch_mode:
                print(f"🔗 Connecting to Proxmox API at {host} using token authentication...")
            try:
                self.api = ProxmoxAPI(host, user, token_name=token_name, token_value=token_value)
                if not batch_mode:
                    print("✅ Connected successfully using API token")
                self._initialize_components()
                return True
            except ProxmoxAPIError as e:
                if batch_mode:
                    print(f"❌ BATCH MODE: Token authentication failed: {e.message}")
                    print("❌ BATCH MODE: Cannot proceed without valid credentials")
                    return False
                else:
                    print(f"⚠️ Token authentication failed: {e.message}")
                    print("🔄 Falling back to password authentication...")
        else:
            if batch_mode:
                print("❌ BATCH MODE: Missing required environment variables")
                print("❌ BATCH MODE: Set PVE_HOST, PVE_USER, PVE_TOKEN_NAME, PVE_TOKEN_VALUE")
                return False
            else:
                print("⚠️  Missing environment variables, using interactive authentication")
        
        # Fall back to interactive authentication (only if not in batch mode)
        if batch_mode:
            print("❌ BATCH MODE: Interactive authentication not allowed")
            return False
            
        print("🔗 Interactive Proxmox API Connection")
        print("=" * 40)
        
        try:
            if not host:
                host = input("Proxmox host (IP or FQDN): ").strip()
            if not user:
                user = input("Username (user@realm): ").strip()
            
            if not host or not user:
                print("⚠️ Host and username are required")
                return False
            
            print(f"🔗 Connecting to {host}...")
            self.api = ProxmoxAPI(host, user)
            print("✅ Connected successfully")
            self._initialize_components()
            return True
            
        except ProxmoxAPIError as e:
            print(f"⚠️ Connection failed: {e.message}")
            return False
        except KeyboardInterrupt:
            print("\n⚠️ Connection cancelled")
            return False
    
    def _initialize_components(self):
        """Initialize all component modules after API connection is established."""
        self.vm_ops = VMOperations(self.api)
        self.snapshot_ops = SnapshotOperations(self.api, self.vm_ops)
        self.bulk_snapshot_ops = BulkSnapshotOperations(self.api, self.vm_ops)
        self.bulk_vm_ops = BulkVMOperations(self.api, self.vm_ops)
        self.vm_selector = VMSelector(self.vm_ops)
    
    # Delegate methods to appropriate components
    def get_nodes(self) -> List[Dict]:
        return self.vm_ops.get_nodes()
    
    def find_vm_node(self, vmid: str) -> Optional[str]:
        return self.vm_ops.find_vm_node(vmid)
    
    def get_all_vms(self) -> List[Dict]:
        return self.vm_ops.get_all_vms()
    
    def get_vm_info(self, vmid: str) -> Optional[Dict]:
        return self.vm_ops.get_vm_info(vmid)
    
    def get_all_vms_info(self) -> List[Dict]:
        return self.vm_ops.get_all_vms_info()
    
    def get_vm_status_detailed(self, vmid: str):
        return self.vm_ops.get_vm_status_detailed(vmid)
    
    def get_vm_name(self, vmid: str) -> Optional[str]:
        return self.vm_ops.get_vm_name(vmid)
    
    def get_full_vm_name(self, vmid: str) -> Optional[str]:
        return self.vm_ops.get_full_vm_name(vmid)
    
    def truncate_vm_name_intelligently(self, vm_name: str, max_length: int) -> str:
        return self.vm_ops.truncate_vm_name_intelligently(vm_name, max_length)
    
    def monitor_task(self, node: str, task_id: str, description: str = "Task") -> bool:
        return self.vm_ops.monitor_task(node, task_id, description)
    
    def _monitor_task_silent(self, node: str, task_id: str) -> bool:
        return self.vm_ops._monitor_task_silent(node, task_id)
    
    def start_vm(self, vmid: str) -> bool:
        return self.vm_ops.start_vm(vmid)
    
    def _start_vm_silent(self, vmid: str) -> bool:
        return self.vm_ops._start_vm_silent(vmid)
    
    def show_vm_details(self, vmid: str):
        # Add snapshot count to VM details
        self.vm_ops.show_vm_details(vmid)
        snapshots = self.snapshot_ops.get_snapshots(vmid)
        snapshot_count = len([s for s in snapshots if s.get('name') != 'current'])
        print(f"Snapshots: {snapshot_count}")
        print(f"{'='*60}\n")
    
    def display_vm_list_interactive(self):
        """Display enhanced VM list for interactive mode with snapshot counts."""
        self.vm_ops.display_vm_list_interactive(self.snapshot_ops)
    
    # Snapshot operations
    def get_snapshots(self, vmid: str) -> List[Dict]:
        return self.snapshot_ops.get_snapshots(vmid)
    
    def create_snapshot(self, vmid: str, name_or_prefix: str, use_exact_name: bool = False) -> bool:
        return self.snapshot_ops.create_snapshot(vmid, name_or_prefix, use_exact_name, self.save_vmstate)
    
    def _create_snapshot_silent(self, vmid: str, prefix: str) -> bool:
        return self.snapshot_ops.create_snapshot_silent(vmid, prefix, self.save_vmstate)
    
    def rollback_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        return self.snapshot_ops.rollback_snapshot(vmid, snapshot_name)
    
    def delete_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        return self.snapshot_ops.delete_snapshot(vmid, snapshot_name)
    
    def _delete_snapshot_silent(self, vmid: str, snapshot_name: str) -> bool:
        return self.snapshot_ops.delete_snapshot_silent(vmid, snapshot_name)
    
    def list_snapshots(self, vmid: str):
        self.snapshot_ops.list_snapshots(vmid)
    
    def delete_all_snapshots(self, vmid: str, snapshots: List[Dict]):
        self.snapshot_ops.delete_all_snapshots(vmid, snapshots)
    
    # Bulk operations
    def bulk_create_snapshots(self, vm_ids: List[str], prefix: str, max_workers: int = 2):
        return self.bulk_snapshot_ops.bulk_create_snapshots(vm_ids, prefix, max_workers)
    
    def bulk_delete_snapshots(self, snapshots_by_vm: Dict[str, List[str]], max_workers: int = 2):
        return self.bulk_snapshot_ops.bulk_delete_snapshots(snapshots_by_vm, max_workers)
    
    def bulk_start_vms(self, vm_ids: List[str], max_workers: int = 3):
        return self.bulk_vm_ops.bulk_start_vms(vm_ids, max_workers)
    
    # Interactive handlers
    def handle_create_snapshot(self, vmid: str):
        """Handle creating a snapshot for a VM."""
        print(f"\n📸 Create Snapshot for VM {vmid}")
        print("=" * 50)
        
        # Show VM details
        vm_info = self.get_vm_info(vmid)
        if not vm_info:
            print("⚠️ Could not get VM information")
            input("Press Enter to continue...")
            return
        
        # Show current VM status
        is_running, status_display, status_details = self.get_vm_status_detailed(vmid)
        print(f"VM Status: {status_display}")
        if status_details:
            print(f"Details: {status_details}")
        
        # Show current snapshots
        print("\n📋 Current Snapshots:")
        snapshots = self.get_snapshots(vmid)
        if snapshots:
            non_current_snapshots = [s for s in snapshots if s.get('name') != 'current']
            if non_current_snapshots:
                # Sort by snaptime (newest first) - handle missing snaptime
                non_current_snapshots.sort(key=lambda x: x.get('snaptime', 0), reverse=True)
                print(f"Found {len(non_current_snapshots)} existing snapshots:")
                for i, snapshot in enumerate(non_current_snapshots, 1):
                    name = snapshot.get('name', 'Unknown')
                    desc = snapshot.get('description', 'No description')
                    # Truncate long descriptions
                    if len(desc) > 50:
                        desc = desc[:47] + "..."
                    print(f"  {i}. {name} - {desc}")
            else:
                print("  No snapshots found")
        else:
            print("  No snapshots found")
        
        print("\n" + "=" * 50)
        
        # Get snapshot name prefix
        print("Enter snapshot prefix (will be combined with VM name and timestamp):")
        print("Examples: 'pre-update', 'backup', 'test', 'stable'")
        
        prefix = input("Snapshot prefix: ").strip()
        if not prefix:
            print("⚠️ Snapshot prefix is required")
            input("Press Enter to continue...")
            return
        
        # Validate prefix
        if len(prefix) > 20:
            print("⚠️ Prefix too long (max 20 characters)")
            input("Press Enter to continue...")
            return
        
        # Ask about vmstate (RAM)
        print(f"\nSnapshot options:")
        print(f"1. Include RAM state (vmstate) - Slower but complete state")
        print(f"2. Disk only - Faster but no RAM state")
        print(f"Note: RAM state is only saved if VM is currently running")
        
        vmstate_choice = input("Select option (1-2, default: 1): ").strip()
        save_vmstate = vmstate_choice != '2'
        
        # Show preview
        timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
        vm_name = self.get_vm_name(vmid) or f"VM-{vmid}"
        preview_name = f"{prefix}-{vm_name}-{timestamp}"
        
        print(f"\n📋 Snapshot Preview:")
        print(f"  VM: {vmid} ({vm_info.get('name', 'Unknown')})")
        print(f"  Snapshot name: {preview_name}")
        print(f"  Include RAM: {'Yes' if save_vmstate and is_running else 'No' if not save_vmstate else 'N/A (VM stopped)'}")
        print(f"  VM Status: {status_display}")
        
        # Final confirmation
        confirm = input(f"\nCreate snapshot '{preview_name}'? (y/N): ").strip().lower()
        if confirm in ['y', 'yes']:
            # Set vmstate option temporarily
            original_vmstate = getattr(self, 'save_vmstate', True)
            self.save_vmstate = save_vmstate
            
            try:
                success = self.create_snapshot(vmid, prefix)
                if success:
                    print("\n✅ Snapshot created successfully!")
                    
                    # Show updated snapshot list
                    print("\n📋 Updated Snapshot List:")
                    updated_snapshots = self.get_snapshots(vmid)
                    if updated_snapshots:
                        non_current = [s for s in updated_snapshots if s.get('name') != 'current']
                        # Sort by snaptime (newest first) - handle missing snaptime
                        non_current.sort(key=lambda x: x.get('snaptime', 0), reverse=True)
                        for i, snapshot in enumerate(non_current, 1):
                            name = snapshot.get('name', 'Unknown')
                            desc = snapshot.get('description', 'No description')
                            if len(desc) > 50:
                                desc = desc[:47] + "..."
                            print(f"  {i}. {name} - {desc}")
                else:
                    print("\n⚠️ Failed to create snapshot!")
            finally:
                # Restore original vmstate setting
                self.save_vmstate = original_vmstate
            
            input("\nPress Enter to continue...")
        else:
            print("Snapshot creation cancelled")
            input("Press Enter to continue...")
    
    def find_vm_by_name_or_id(self, identifier: str) -> Optional[str]:
        """Find VM ID by either VM ID or VM name."""
        # First, try to get VM info directly (assuming it's a VM ID)
        vm_info = self.get_vm_info(identifier)
        if vm_info:
            return identifier
        
        # If that fails, search by name
        all_vms = self.get_all_vms()
        for vm in all_vms:
            vm_name = vm.get('name', '')
            if vm_name.lower() == identifier.lower():
                return str(vm['vmid'])
        
        # Also try partial name matching (case-insensitive)
        for vm in all_vms:
            vm_name = vm.get('name', '')
            if identifier.lower() in vm_name.lower():
                return str(vm['vmid'])
        
        return None
    
    def main_menu(self):
        """Main menu for snapshot management."""
        print("Proxmox Snapshot Management Tool")
        print("=" * 30)
        
        while True:
            try:
                print()
                print("Options:")
                print("  1. View Available VMs")
                print("  2. Manage Single VM (enter VM ID or Name)")
                print("  3. Bulk Operations (Snapshots & VM Management)")
                print("  q. Quit")
                print()
                
                choice = input("Select option (1-3, VM ID/Name, or 'q'): ").strip()
                
                if choice.lower() in ['q', 'quit']:
                    print("Goodbye!")
                    break
                elif choice == '1':
                    # View Available VMs
                    self.display_vm_list_interactive()
                    input("\nPress Enter to continue...")
                elif choice == '2':
                    # Manage Single VM
                    vm_identifier = input("Enter VM ID or VM Name: ").strip()
                    if vm_identifier:
                        vm_id = self.find_vm_by_name_or_id(vm_identifier)
                        if vm_id:
                            vm_info = self.get_vm_info(vm_id)
                            print(f"✅ Found VM {vm_id}: {vm_info['name']}")
                            self.manage_vm_snapshots(vm_id)
                        else:
                            print(f"⚠️ VM '{vm_identifier}' not found")
                            print("You can enter:")
                            print("  - VM ID (e.g., 7303)")
                            print("  - Full VM name (e.g., xsf-dev-workstation03)")
                            print("  - Partial VM name (e.g., workstation03)")
                            input("Press Enter to continue...")
                elif choice == '3':
                    print("Bulk operations menu not implemented in this modular version yet.")
                    input("Press Enter to continue...")
                else:
                    # Try to interpret as VM ID or Name
                    vm_id = self.find_vm_by_name_or_id(choice)
                    if vm_id:
                        vm_info = self.get_vm_info(vm_id)
                        print(f"✅ Found VM {vm_id}: {vm_info['name']}")
                        self.manage_vm_snapshots(vm_id)
                    else:
                        print(f"⚠️ Invalid option or VM '{choice}' not found")
                        print("You can enter:")
                        print("  - Menu option (1-3)")
                        print("  - VM ID (e.g., 7303)")
                        print("  - Full VM name (e.g., xsf-dev-workstation03)")
                        print("  - Partial VM name (e.g., workstation03)")
                        input("Press Enter to continue...")
                
            except KeyboardInterrupt:
                print("\nGoodbye!")
                break
    
    def manage_vm_snapshots(self, vmid: str):
        """Sub-menu for VM snapshot management operations."""
        while True:
            try:
                # Show current VM status
                self.show_vm_details(vmid)
                
                # Get current status
                vm_info = self.get_vm_info(vmid)
                if not vm_info:
                    print("⚠️ VM not found or inaccessible")
                    return
                
                # Show operation menu
                print("VM Management Operations:")
                print("=" * 30)
                print("1. Create Snapshot")
                print("2. Rollback Snapshot")
                print("3. List Snapshots")
                print("4. Delete Snapshot")
                print("5. Start VM")
                print("6. Back to main menu")
                print("q. Quit")
                print()
                
                choice = input("Select operation (1-6, q): ").strip()
                
                if choice == '1':  # Create Snapshot
                    self.handle_create_snapshot(vmid)
                
                elif choice == '2':  # Rollback Snapshot
                    print("Rollback snapshot handler not implemented in this modular version yet.")
                    input("Press Enter to continue...")
                
                elif choice == '3':  # List Snapshots
                    self.list_snapshots(vmid)
                    input("\nPress Enter to continue...")
                
                elif choice == '4':  # Delete Snapshot
                    print("Delete snapshot handler not implemented in this modular version yet.")
                    input("Press Enter to continue...")
                
                elif choice == '5':  # Start VM
                    print("Start VM handler not implemented in this modular version yet.")
                    input("Press Enter to continue...")
                
                elif choice == '6':  # Back to main menu
                    break
                
                elif choice.lower() in ['q', 'quit']:
                    print("Goodbye!")
                    sys.exit(0)
                
                else:
                    print("Invalid choice. Please select 1-6 or 'q'.")
                    input("Press Enter to continue...")
                
            except KeyboardInterrupt:
                print("\nReturning to main menu...")
                break