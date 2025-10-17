#!/usr/bin/env python3

"""
Snapshot Operations Module
Handles all snapshot-related operations including creation, deletion, rollback, and listing
"""

import re
import time
from datetime import datetime
from typing import List, Dict, Optional
from proxmox_api import ProxmoxAPIError


class SnapshotOperations:
    """Handles snapshot operations for Proxmox VMs."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
        self.max_snapshot_name_length = 40
        self.vmstate_keywords = ['vmstate', 'RAM', 'with vmstate', 'RAM included', 'with VM state', 'VM state included']
    
    def get_snapshots(self, vmid: str) -> List[Dict]:
        """Get list of snapshots for a VM."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            return []
        
        try:
            snapshots = self.api._request('GET', f'/nodes/{node}/qemu/{vmid}/snapshot')
            return snapshots
        except ProxmoxAPIError:
            return []
    
    def get_snapshot_config(self, vmid: str, snapshot_name: str) -> Dict:
        """Get configuration of a specific snapshot."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            return {}
        
        try:
            config = self.api._request('GET', f'/nodes/{node}/qemu/{vmid}/snapshot/{snapshot_name}/config')
            return config
        except ProxmoxAPIError:
            return {}
    
    def create_snapshot(self, vmid: str, name_or_prefix: str, use_exact_name: bool = False, save_vmstate: bool = False) -> bool:
        """Create a snapshot for a VM with intelligent naming and monitoring."""
        timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
        vm_name = self.vm_manager.get_vm_name(vmid)
        
        if not vm_name:
            print(f"  ✗ Could not retrieve VM name for VMID {vmid}")
            return False
        
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            print(f"  ✗ Could not find node for VM {vmid}")
            return False
        
        # Create snapshot name based on mode
        if use_exact_name:
            full_snapshot_name = name_or_prefix
            print(f"Creating snapshot for VM {vmid} with exact name...")
        else:
            # Original behavior: prefix + vm_name + timestamp
            full_snapshot_name = f"{name_or_prefix}-{vm_name}-{timestamp}"
            print(f"Creating snapshot for VM {vmid} with generated name...")
            
            # Handle name length limits for generated names
            if len(full_snapshot_name) > self.max_snapshot_name_length:
                print(f"  ⚠ Snapshot name too long ({len(full_snapshot_name)} chars), truncating VM name...")
                
                prefix_suffix_length = len(name_or_prefix) + 1 + 1 + 13
                max_vm_name_length = self.max_snapshot_name_length - prefix_suffix_length
                
                if max_vm_name_length <= 0:
                    print(f"  ✗ Prefix '{name_or_prefix}' is too long. Maximum prefix length is {self.max_snapshot_name_length - 14} characters")
                    return False
                
                truncated_vm_name = self.vm_manager.truncate_vm_name_intelligently(vm_name, max_vm_name_length)
                full_snapshot_name = f"{name_or_prefix}-{truncated_vm_name}-{timestamp}"
                print(f"  📝 Truncated VM name: '{vm_name}' -> '{truncated_vm_name}'")
        
        # Validate final snapshot name length
        if len(full_snapshot_name) > self.max_snapshot_name_length:
            print(f"  ✗ Final snapshot name '{full_snapshot_name}' is too long ({len(full_snapshot_name)} chars, max {self.max_snapshot_name_length})")
            return False
        
        full_vm_name = self.vm_manager.get_full_vm_name(vmid)
        if full_vm_name:
            print(f"  Full VM Name: {full_vm_name}")
        print(f"  Clean VM Name: {vm_name}")
        print(f"  Snapshot: {full_snapshot_name} ({len(full_snapshot_name)} chars)")
        print(f"  Node: {node}")
        
        # Check VM status
        print(f"  📊 Checking VM status...")
        is_running, status_display, status_details = self.vm_manager.get_vm_status_detailed(vmid)
        print(f"  Status: {status_display}")
        
        # Determine vmstate behavior
        if save_vmstate and not is_running:
            print(f"  ⚠ VM {vmid} is not running - vmstate will be ignored")
        
        print(f"  VM State: {'WITH vmstate (RAM)' if save_vmstate and is_running else 'WITHOUT vmstate'}")
        
        try:
            # Prepare snapshot data
            snapshot_data = {
                'snapname': full_snapshot_name,
                'description': f'Snapshot created {"with" if save_vmstate and is_running else "without"} vmstate - {timestamp}'
            }
            
            if save_vmstate and is_running:
                snapshot_data['vmstate'] = '1'
            
            # Create snapshot
            print(f"  🔄 Creating snapshot...")
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/snapshot', data=snapshot_data)
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Snapshot creation for VM {vmid}")
            
            if success:
                # Verify snapshot
                print(f"  🔍 Verifying snapshot...")
                snapshot_config = self.get_snapshot_config(vmid, full_snapshot_name)
                if snapshot_config:
                    print(f"  ✓ Snapshot verification successful")
                else:
                    print(f"  ⚠ Could not verify snapshot config")
                
                # Check VM status after snapshot
                print(f"  📊 Checking VM status after snapshot...")
                is_running_after, status_display_after, _ = self.vm_manager.get_vm_status_detailed(vmid)
                print(f"  Status: {status_display_after}")
                
                if is_running == is_running_after:
                    print(f"  ✓ VM status unchanged (as expected)")
                else:
                    print(f"  ⚠ VM status changed from {'running' if is_running else 'stopped'} to {'running' if is_running_after else 'stopped'}")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"  ✗ Failed to create snapshot: {e.message}")
            return False
    
    def create_snapshot_silent(self, vmid: str, prefix: str, save_vmstate: bool = True) -> bool:
        """Create a snapshot without output (for bulk operations)."""
        timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
        vm_name = self.vm_manager.get_vm_name(vmid) or f"VM-{vmid}"
        snapshot_name = f"{prefix}-{vm_name}-{timestamp}"
        
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            # Prepare snapshot data
            snapshot_data = {
                'snapname': snapshot_name,
                'description': f'Bulk snapshot created {"with" if save_vmstate else "without"} vmstate - {timestamp}'
            }
            
            # Check if VM is running and vmstate is enabled
            vm_info = self.vm_manager.get_vm_info(vmid)
            if vm_info and vm_info.get('running') and save_vmstate:
                snapshot_data['vmstate'] = '1'
            
            # Create snapshot
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/snapshot', data=snapshot_data)
            
            # Monitor task without output
            return self.vm_manager._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False
    
    def rollback_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        """Rollback VM to a specific snapshot."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            print("⚠️ Could not find node for VM")
            return False
        
        try:
            print(f"⏪ Executing rollback to snapshot '{snapshot_name}'...")
            
            # Execute rollback via API
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/snapshot/{snapshot_name}/rollback')
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Rollback for VM {vmid}")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"⚠️ Rollback failed: {e.message}")
            return False
    
    def delete_snapshot(self, vmid: str, snapshot_name: str) -> bool:
        """Delete a specific snapshot."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            print("⚠️ Could not find node for VM")
            return False
        
        try:
            print(f"🗑️  Executing snapshot deletion...")
            
            # Execute deletion via API
            task_id = self.api._request('DELETE', f'/nodes/{node}/qemu/{vmid}/snapshot/{snapshot_name}')
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Snapshot deletion for VM {vmid}")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"⚠️ Snapshot deletion failed: {e.message}")
            return False
    
    def delete_snapshot_silent(self, vmid: str, snapshot_name: str) -> bool:
        """Delete a snapshot without output (for bulk operations)."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            # Execute deletion via API
            task_id = self.api._request('DELETE', f'/nodes/{node}/qemu/{vmid}/snapshot/{snapshot_name}')
            
            # Monitor task without output
            return self.vm_manager._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False
    
    def list_snapshots(self, vmid: str):
        """List all snapshots for a VM in a formatted table."""
        print(f"\n📸 Snapshots for VM {vmid}")
        print("=" * 120)
        
        # Show VM details
        vm_info = self.vm_manager.get_vm_info(vmid)
        if not vm_info:
            print("⚠️ Could not get VM information")
            return
        
        # Show current VM status
        is_running, status_display, status_details = self.vm_manager.get_vm_status_detailed(vmid)
        print(f"VM: {vm_info['name']} ({status_display})")
        print(f"Node: {vm_info['node']}")
        print()
        
        # Get snapshots
        snapshots = self.get_snapshots(vmid)
        if not snapshots:
            print("⚠️ No snapshots found for this VM")
            return
        
        # Filter out 'current' snapshot
        available_snapshots = [s for s in snapshots if s.get('name') != 'current']
        if not available_snapshots:
            print("⚠️ No snapshots available (only current state found)")
            return
        
        # Sort by snaptime (newest first) - handle missing snaptime
        available_snapshots.sort(key=lambda x: x.get('snaptime', 0), reverse=True)
        
        # Display snapshots table with full names
        print(f"Found {len(available_snapshots)} snapshot(s):")
        print("-" * 160)
        print(f"{'#':<3} {'Snapshot Name':<50} {'Description':<40} {'Created':<20} {'Parent':<20} {'VMState':<8}")
        print("-" * 160)
        
        for i, snapshot in enumerate(available_snapshots, 1):
            name = snapshot.get('name', 'Unknown')
            desc = snapshot.get('description', 'No description')
            parent = snapshot.get('parent', 'N/A')
            
            # Format creation time
            snaptime = snapshot.get('snaptime', 0)
            if snaptime:
                created = datetime.fromtimestamp(snaptime).strftime('%Y-%m-%d %H:%M:%S')
            else:
                created = 'Unknown'
            
            # Check if snapshot has vmstate
            has_vmstate = '⛔'
            if 'vmstate' in snapshot:
                has_vmstate = '✅' if snapshot.get('vmstate') else '⛔'
            elif any(keyword in desc.lower() for keyword in self.vmstate_keywords):
                has_vmstate = '✅'
            
            # Truncate only description if too long, keep names full
            if len(desc) > 39:
                desc = desc[:36] + "..."
            if len(parent) > 19:
                parent = parent[:16] + "..."
            
            print(f"{i:<3} {name:<50} {desc:<40} {created:<20} {parent:<20} {has_vmstate:<8}")
        
        print("-" * 160)
        print(f"Total: {len(available_snapshots)} snapshots")
        print("Legend: ✅ = Has VM state (RAM), ⛔ = Disk only")
    
    def delete_all_snapshots(self, vmid: str, snapshots: List[Dict]):
        """Delete all snapshots for a VM."""
        print(f"\n🗑️  DELETE ALL SNAPSHOTS WARNING")
        print("=" * 60)
        print("This operation will:")
        print(f"  • Permanently delete ALL {len(snapshots)} snapshots")
        print("  • Free up significant disk space")
        print("  • This action cannot be undone!")
        print("=" * 60)
        
        print(f"\nSnapshots to be deleted:")
        for i, snapshot in enumerate(snapshots, 1):
            name = snapshot.get('name', 'Unknown')
            desc = snapshot.get('description', 'No description')
            snaptime = snapshot.get('snaptime', 0)
            if snaptime:
                created = datetime.fromtimestamp(snaptime).strftime('%Y-%m-%d %H:%M')
            else:
                created = 'Unknown'
            print(f"  {i}. {name} (Created: {created})")
        
        print("=" * 60)
        
        # Double confirmation for safety
        print("\n⚠️  FINAL WARNING: This will delete ALL snapshots!")
        confirm1 = input("Type 'DELETE ALL' to confirm this dangerous operation: ").strip()
        
        if confirm1 != 'DELETE ALL':
            print("Operation cancelled - confirmation text did not match")
            input("Press Enter to continue...")
            return
        
        confirm2 = input("Are you absolutely sure? Type 'YES' to proceed: ").strip().upper()
        
        if confirm2 != 'YES':
            print("Operation cancelled")
            input("Press Enter to continue...")
            return
        
        # Proceed with bulk deletion
        print(f"\n🗑️  Deleting {len(snapshots)} snapshots...")
        print("This may take several minutes depending on snapshot sizes...\n")
        
        success_count = 0
        failed_count = 0
        
        for i, snapshot in enumerate(snapshots, 1):
            snapshot_name = snapshot.get('name', 'Unknown')
            print(f"[{i}/{len(snapshots)}] Deleting '{snapshot_name}'...")
            
            success = self.delete_snapshot(vmid, snapshot_name)
            if success:
                success_count += 1
                print(f"  ✅ Deleted successfully")
            else:
                failed_count += 1
                print(f"  ⚠️ Failed to delete")
            
            # Small delay between deletions to avoid overwhelming the system
            if i < len(snapshots):
                time.sleep(1)
        
        print(f"\n{'='*60}")
        print("BULK DELETION COMPLETED")
        print(f"{'='*60}")
        print(f"✅ Successfully deleted: {success_count}")
        print(f"⚠️ Failed to delete: {failed_count}")
        print(f"📊 Total processed: {len(snapshots)}")
        
        if failed_count > 0:
            print(f"\n⚠️  Some snapshots could not be deleted. Check VM status and try again.")
        else:
            print(f"\n🎉 All snapshots deleted successfully!")
        
        print(f"{'='*60}")
        input("\nPress Enter to continue...")