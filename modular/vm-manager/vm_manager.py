#!/usr/bin/env python3

"""
VM Manager Module
Main orchestrator class that coordinates all VM management operations including
backup, storage, and snapshot operations through a unified interface
"""

import os
import sys
import time
from typing import List, Dict, Optional, Tuple
from concurrent.futures import ThreadPoolExecutor, as_completed

from proxmox_api import ProxmoxAPI, ProxmoxAPIError
from vm_operations import VMOperations
from vm_selector import VMSelector
from backup_operations import BackupOperations
from storage_operations import StorageOperations
from bulk_operations import BulkOperationManager, BulkOperationResult


class ProxmoxVMManager:
    """Main manager class that orchestrates all Proxmox VM management operations."""
    
    def __init__(self):
        self.api = None
        self.vm_ops = None
        self.backup_ops = None
        self.storage_ops = None
        self.vm_selector = None
        
        # Concurrency limits
        self.MAX_CONCURRENT_START_STOP = 3
        self.MAX_CONCURRENT_BACKUPS = 2
        self.MAX_CONCURRENT_SNAPSHOTS = 2
    
    def display_usage(self):
        """Display usage information."""
        usage_text = """
Proxmox VM Management Tool (Modular Version)
=============================================

This tool provides comprehensive VM management capabilities:
- View all VMs with real-time status
- Start/Stop VMs with safety checks
- Create VM backups with storage selection
- Manage VM protection settings
- Real-time task monitoring
- Multi-node cluster support
- Bulk operations with progress tracking

API Authentication Options:
  1. Username/Password (prompted)
  2. API Token (set environment variables):
     export PVE_HOST=your-proxmox-host
     export PVE_USER=username@realm
     export PVE_TOKEN_NAME=token-name
     export PVE_TOKEN_VALUE=token-value

Usage: python3 main.py
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
            print(f"   PVE_HOST: {'✅ Set' if host else '❌ Not set'}")
            print(f"   PVE_USER: {'✅ Set' if user else '❌ Not set'}")
            print(f"   PVE_TOKEN_NAME: {'✅ Set' if token_name else '❌ Not set'}")
            print(f"   PVE_TOKEN_VALUE: {'✅ Set' if token_value else '❌ Not set'}")
        
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
                    return False
                else:
                    print(f"❌ Token authentication failed: {e.message}")
                    print("🔄 Falling back to password authentication...")
        else:
            if batch_mode:
                print("❌ BATCH MODE: Missing required environment variables")
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
                print("❌ Host and username are required")
                return False
            
            print(f"🔗 Connecting to {host}...")
            self.api = ProxmoxAPI(host, user)
            print("✅ Connected successfully")
            self._initialize_components()
            return True
            
        except ProxmoxAPIError as e:
            print(f"❌ Connection failed: {e.message}")
            return False
        except KeyboardInterrupt:
            print("\n❌ Connection cancelled")
            return False
    
    def _initialize_components(self):
        """Initialize all component modules after API connection is established."""
        self.vm_ops = VMOperations(self.api)
        self.backup_ops = BackupOperations(self.api, self.vm_ops)
        self.storage_ops = StorageOperations(self.api, self.vm_ops)
        self.vm_selector = VMSelector(self.vm_ops)
    
    # Delegate core VM operations
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
    
    def monitor_task(self, node: str, task_id: str, description: str = "Task") -> bool:
        return self.vm_ops.monitor_task(node, task_id, description)
    
    def _monitor_task_silent(self, node: str, task_id: str) -> bool:
        return self.vm_ops._monitor_task_silent(node, task_id)
    
    def start_vm(self, vmid: str) -> bool:
        return self.vm_ops.start_vm(vmid)
    
    def _start_vm_silent(self, vmid: str) -> bool:
        return self.vm_ops._start_vm_silent(vmid)
    
    def show_vm_details(self, vmid: str):
        return self.vm_ops.show_vm_details(vmid)
    
    def display_vm_list_interactive(self):
        """Display enhanced VM list for interactive mode."""
        return self.vm_ops.display_vm_list_interactive()
    
    # VM state management operations
    def stop_vm(self, vmid: str) -> bool:
        """Stop a VM with safety checks."""
        node = self.find_vm_node(vmid)
        if not node:
            print("❌ Could not find node for VM")
            return False
        
        try:
            print("🛑 Stopping VM...")
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/stop')
            
            # Monitor task progress
            success = self.monitor_task(node, task_id, f"VM {vmid} shutdown")
            
            if success:
                print("✅ VM stopped successfully!")
            else:
                print("❌ Failed to stop VM!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to stop VM: {e.message}")
            return False

    def shutdown_vm(self, vmid: str) -> bool:
        """Gracefully shutdown a VM using ACPI signal."""
        node = self.find_vm_node(vmid)
        if not node:
            print("❌ Could not find node for VM")
            return False
        
        try:
            print("🔄 Gracefully shutting down VM...")
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/shutdown')
            
            # Monitor task progress
            success = self.monitor_task(node, task_id, f"VM {vmid} graceful shutdown")
            
            if success:
                print("✅ VM shutdown successfully!")
            else:
                print("❌ Failed to shutdown VM!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to shutdown VM: {e.message}")
            return False
    
    def _stop_vm_silent(self, vmid: str) -> bool:
        """Stop a VM without output (for bulk operations)."""
        node = self.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/stop')
            return self._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False

    def _shutdown_vm_silent(self, vmid: str) -> bool:
        """Gracefully shutdown a VM without output (for bulk operations)."""
        node = self.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/shutdown')
            return self._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False
    
    # Backup operations delegation
    def create_backup(self, vmid: str, storage: str, mode: str, compress: str = 'zstd') -> bool:
        return self.backup_ops.create_backup(vmid, storage, mode, compress)
    
    def list_backups_for_vm(self, vmid: str, storage: str = None) -> List[Dict]:
        return self.backup_ops.list_backups_for_vm(vmid, storage)
    
    def display_backup_list(self, backups: List[Dict]) -> List[Dict]:
        return self.backup_ops.display_backup_list(backups)
    
    def restore_backup(self, vmid: str, volid: str, node: str, storage: str = None) -> bool:
        return self.backup_ops.restore_backup(vmid, volid, node, storage)
    
    def check_and_handle_protection(self, vmid: str) -> bool:
        return self.backup_ops.check_and_handle_protection(vmid)
    
    def delete_backup(self, volid: str, node: str, storage: str) -> bool:
        return self.backup_ops.delete_backup(volid, node, storage)
    
    def bulk_delete_backups(self, backups_to_delete: List[Dict], max_workers: int = 2):
        return self.backup_ops.bulk_delete_backups(backups_to_delete, max_workers)
    
    def delete_backups_by_pattern(self, vm_id: str, pattern: str, max_age_days: int = None) -> bool:
        return self.backup_ops.delete_backups_by_pattern(vm_id, pattern, max_age_days)
    
    def delete_old_backups(self, vm_id: str, keep_count: int = 5, max_age_days: int = 30) -> bool:
        return self.backup_ops.delete_old_backups(vm_id, keep_count, max_age_days)
    
    # Storage operations delegation
    def display_vm_storage_list(self) -> List[Dict]:
        return self.storage_ops.display_vm_storage_list()
    
    def display_backup_storage_list(self) -> List[Dict]:
        return self.storage_ops.display_backup_storage_list()
    
    def select_storage_interactive(self, storage_type: str = "backup") -> str:
        return self.storage_ops.select_storage_interactive(storage_type)
    
    def validate_storage_space(self, storage_name: str, required_space_gb: float = 10.0) -> bool:
        return self.storage_ops.validate_storage_space(storage_name, required_space_gb)
    
    # Bulk operations
    def bulk_start_vms(self, vm_ids: List[str], max_workers: int = None) -> BulkOperationManager:
        """Start multiple VMs concurrently."""
        if max_workers is None:
            max_workers = self.MAX_CONCURRENT_START_STOP
            
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🚀 Starting {len(vm_ids)} VMs (max {max_workers} concurrent)")
        print("=" * 60)
        
        def start_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM is already running
                vm_info = self.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "start", False, "VM not found", time.time() - start_time)
                
                if vm_info.get('running', False):
                    return BulkOperationResult(vmid, "start", True, "Already running", time.time() - start_time)
                
                # Start the VM (silent version for bulk operations)
                success = self._start_vm_silent(vmid)
                message = "Started successfully" if success else "Failed to start"
                return BulkOperationResult(vmid, "start", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(vmid, "start", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vmid = {executor.submit(start_single_vm, vmid): vmid for vmid in vm_ids}
            
            for future in as_completed(future_to_vmid):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(len(vm_ids), "Start VMs")
        
        operation_manager.print_summary("Bulk Start VMs")
        return operation_manager
    
    def bulk_shutdown_vms(self, vm_ids: List[str], max_workers: int = None) -> BulkOperationManager:
        """Gracefully shutdown multiple VMs concurrently."""
        if max_workers is None:
            max_workers = self.MAX_CONCURRENT_START_STOP
            
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🔄 Gracefully shutting down {len(vm_ids)} VMs (max {max_workers} concurrent)")
        print("=" * 60)
        
        def shutdown_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM is already stopped
                vm_info = self.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "shutdown", False, "VM not found", time.time() - start_time)
                
                if not vm_info.get('running', False):
                    return BulkOperationResult(vmid, "shutdown", True, "Already stopped", time.time() - start_time)
                
                # Shutdown the VM (silent version for bulk operations)
                success = self._shutdown_vm_silent(vmid)
                message = "Shutdown successfully" if success else "Failed to shutdown"
                return BulkOperationResult(vmid, "shutdown", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(vmid, "shutdown", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vmid = {executor.submit(shutdown_single_vm, vmid): vmid for vmid in vm_ids}
            
            for future in as_completed(future_to_vmid):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                operation_manager.print_progress(len(vm_ids), "Shutdown VMs")
        
        operation_manager.print_summary("Bulk Shutdown VMs")
        return operation_manager
    
    def bulk_stop_vms(self, vm_ids: List[str], max_workers: int = None) -> BulkOperationManager:
        """Stop multiple VMs concurrently."""
        if max_workers is None:
            max_workers = self.MAX_CONCURRENT_START_STOP
            
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🛑 Stopping {len(vm_ids)} VMs (max {max_workers} concurrent)")
        print("=" * 60)
        
        def stop_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM is already stopped
                vm_info = self.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "stop", False, "VM not found", time.time() - start_time)
                
                if not vm_info.get('running', False):
                    return BulkOperationResult(vmid, "stop", True, "Already stopped", time.time() - start_time)
                
                # Stop the VM (silent version for bulk operations)
                success = self._stop_vm_silent(vmid)
                message = "Stopped successfully" if success else "Failed to stop"
                return BulkOperationResult(vmid, "stop", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(vmid, "stop", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vmid = {executor.submit(stop_single_vm, vmid): vmid for vmid in vm_ids}
            
            for future in as_completed(future_to_vmid):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(len(vm_ids), "Stop VMs")
        
        operation_manager.print_summary("Bulk Stop VMs")
        return operation_manager
    
    def bulk_create_backups(self, vm_ids: List[str], storage: str, mode: str = 'snapshot', 
                           compress: str = 'zstd', max_workers: int = None) -> BulkOperationManager:
        """Create backups for multiple VMs concurrently."""
        if max_workers is None:
            max_workers = self.MAX_CONCURRENT_BACKUPS
            
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n💾 Creating backups for {len(vm_ids)} VMs")
        print(f"Storage: {storage}, Mode: {mode}, Compression: {compress}")
        print(f"Max concurrent operations: {max_workers}")
        print("=" * 60)
        
        def backup_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM exists
                vm_info = self.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "backup", False, "VM not found", time.time() - start_time)
                
                # Create backup (silent version for bulk operations)
                success = self.backup_ops.create_backup_silent(vmid, storage, mode, compress)
                message = "Backup created successfully" if success else "Failed to create backup"
                return BulkOperationResult(vmid, "backup", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(vmid, "backup", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vmid = {executor.submit(backup_single_vm, vmid): vmid for vmid in vm_ids}
            
            for future in as_completed(future_to_vmid):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(len(vm_ids), "Create Backups")
        
        operation_manager.print_summary("Bulk Create Backups")
        return operation_manager
    
    # Interactive menu system
    def main_menu(self):
        """Main interactive menu."""
        while True:
            try:
                print("\n" + "=" * 60)
                print("🖥️  PROXMOX VM MANAGER (Modular)")
                print("=" * 60)
                print("1. 📋 View All VMs")
                print("2. 🔍 VM Details")
                print("3. 🚀 Start VM")
                print("4. 🛑 Stop VM")
                print("5. 🔄 Graceful Shutdown VM")
                print("6. 💾 Create Backup")
                print("7. 🔄 Restore Backup")
                print("8. 📋 List Backups")
                print("9. 🗑️  Delete Backups")
                print("10. 🔧 Bulk Operations")
                print("11. 🗄️  Storage Management")
                print("12. ❓ Help")
                print("13. 🚪 Exit")
                print()
                
                choice = input("Select option (1-13): ").strip()
                
                if choice == '1':
                    self.display_vm_list_interactive()
                elif choice == '2':
                    self.handle_vm_details()
                elif choice == '3':
                    self.handle_start_vm()
                elif choice == '4':
                    self.handle_stop_vm()
                elif choice == '5':
                    self.handle_shutdown_vm()
                elif choice == '6':
                    self.handle_create_backup()
                elif choice == '7':
                    self.handle_restore_backup()
                elif choice == '8':
                    self.handle_list_backups()
                elif choice == '9':
                    self.handle_delete_backups()
                elif choice == '10':
                    self.bulk_operations_menu()
                elif choice == '11':
                    self.storage_management_menu()
                elif choice == '12':
                    self.display_usage()
                elif choice == '13':
                    print("Goodbye!")
                    break
                else:
                    print("❌ Invalid choice. Please select 1-13")
                    
            except KeyboardInterrupt:
                print("\nGoodbye!")
                break
            except Exception as e:
                print(f"❌ Unexpected error: {e}")
    
    def handle_vm_details(self):
        """Handle VM details display."""
        print("\n🔍 VM Details")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if vm_id:
            self.show_vm_details(vm_id)
        else:
            print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_start_vm(self):
        """Handle VM start operation."""
        print("\n🚀 Start VM")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if vm_id:
            self.start_vm(vm_id)
        else:
            print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_stop_vm(self):
        """Handle VM stop operation."""
        print("\n🛑 Stop VM")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if vm_id:
            self.stop_vm(vm_id)
        else:
            print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_shutdown_vm(self):
        """Handle VM graceful shutdown operation."""
        print("\n🔄 Graceful Shutdown VM")
        print("=" * 40)
        
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if vm_id:
            self.shutdown_vm(vm_id)
        else:
            print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_create_backup(self):
        """Handle backup creation."""
        print("\n💾 Create Backup")
        print("=" * 40)
        
        # Select VM
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # Select storage
        storage = self.select_storage_interactive("backup")
        if not storage:
            return
        
        # Select backup mode
        print("\nBackup modes:")
        print("1. snapshot - Fast backup using VM snapshots")
        print("2. suspend - Suspend VM during backup (ensures consistency)")
        print("3. stop - Stop VM during backup (maximum consistency)")
        
        mode_choice = input("Select backup mode (1-3): ").strip()
        mode_map = {'1': 'snapshot', '2': 'suspend', '3': 'stop'}
        mode = mode_map.get(mode_choice, 'snapshot')
        
        # Select compression
        print("\nCompression options:")
        print("1. zstd - Fast compression (recommended)")
        print("2. gzip - Standard compression")
        print("3. lzo - Fastest compression")
        
        compress_choice = input("Select compression (1-3): ").strip()
        compress_map = {'1': 'zstd', '2': 'gzip', '3': 'lzo'}
        compress = compress_map.get(compress_choice, 'zstd')
        
        # Create backup
        self.create_backup(vm_id, storage, mode, compress)
        
        input("Press Enter to continue...")
    
    def handle_restore_backup(self):
        """Handle backup restoration."""
        print("\n🔄 Restore Backup")
        print("=" * 40)
        
        # Select VM
        vm_selection = input("Enter VM ID or name to restore: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # List backups for this VM
        backups = self.list_backups_for_vm(vm_id)
        if not backups:
            print(f"❌ No backups found for VM {vm_id}")
            input("Press Enter to continue...")
            return
        
        # Display backups and let user select
        displayed_backups = self.display_backup_list(backups)
        
        try:
            backup_choice = input(f"\nSelect backup to restore (1-{len(displayed_backups)}, or 'q' to quit): ").strip().lower()
            
            if backup_choice == 'q':
                return
            
            backup_num = int(backup_choice)
            if 1 <= backup_num <= len(displayed_backups):
                selected_backup = displayed_backups[backup_num - 1]
                
                # Check and handle VM protection
                if not self.check_and_handle_protection(vm_id):
                    return
                
                # Confirm restore
                print(f"\n⚠️  RESTORE WARNING")
                print("=" * 40)
                print("This will:")
                print(f"  • Replace current VM {vm_id} with backup data")
                print("  • Permanently lose all changes made after backup")
                print("  • This action cannot be undone!")
                print("=" * 40)
                
                confirm = input("Type 'RESTORE' to confirm this operation: ").strip()
                if confirm != 'RESTORE':
                    print("Restore operation cancelled")
                    input("Press Enter to continue...")
                    return
                
                # Perform restore
                volid = selected_backup['volid']
                node = selected_backup['node']
                self.restore_backup(vm_id, volid, node)
            else:
                print(f"❌ Invalid choice. Please select 1-{len(displayed_backups)}")
        except ValueError:
            print("❌ Invalid input. Please enter a number")
        
        input("Press Enter to continue...")
    
    def handle_list_backups(self):
        """Handle backup listing."""
        print("\n📋 List Backups")
        print("=" * 40)
        
        # Select VM
        vm_selection = input("Enter VM ID or name (or 'all' for all backups): ").strip()
        if not vm_selection:
            return
        
        if vm_selection.lower() == 'all':
            # Show all backups across all storages
            print("\n🔍 Checking all backup storages...")
            storages = self.storage_ops.get_backup_storages()
            
            all_backups = []
            for storage in storages:
                storage_backups = self.backup_ops.list_all_backups_in_storage(storage['storage'])
                all_backups.extend(storage_backups)
            
            # Sort by creation time (newest first)
            all_backups.sort(key=lambda x: x.get('ctime', 0), reverse=True)
            
            self.display_backup_list(all_backups)
        else:
            all_vms = self.get_all_vms_info()
            vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
            
            if vm_id:
                backups = self.list_backups_for_vm(vm_id)
                self.display_backup_list(backups)
            else:
                print(f"❌ VM '{vm_selection}' not found")
        
        input("Press Enter to continue...")
    
    def handle_delete_backups(self):
        """Handle backup deletion."""
        print("\n🗑️  Delete Backups")
        print("=" * 40)
        
        # Select VM
        vm_selection = input("Enter VM ID or name: ").strip()
        if not vm_selection:
            return
        
        all_vms = self.get_all_vms_info()
        vm_id = self.vm_selector.find_vm_by_name_or_id(vm_selection, all_vms)
        
        if not vm_id:
            print(f"❌ VM '{vm_selection}' not found")
            input("Press Enter to continue...")
            return
        
        # Get backups for this VM
        backups = self.list_backups_for_vm(vm_id)
        if not backups:
            print(f"❌ No backups found for VM {vm_id}")
            input("Press Enter to continue...")
            return
        
        # Display options
        print(f"\n🗑️  Backup Deletion Options for VM {vm_id}")
        print("=" * 50)
        print("1. Delete specific backup(s)")
        print("2. Delete by pattern (e.g., 'backup-*')")
        print("3. Delete all backups")
        print("4. Delete old backups (cleanup)")
        print("5. Cancel")
        print()
        
        choice = input("Select option (1-5): ").strip()
        
        if choice == '1':
            self._handle_delete_specific_backups(vm_id, backups)
        elif choice == '2':
            self._handle_delete_by_pattern(vm_id)
        elif choice == '3':
            self._handle_delete_all_backups(vm_id)
        elif choice == '4':
            self._handle_delete_old_backups(vm_id)
        elif choice == '5':
            print("Operation cancelled")
        else:
            print("❌ Invalid choice")
        
        input("Press Enter to continue...")
    
    def _handle_delete_specific_backups(self, vm_id: str, backups: List[Dict]):
        """Handle deletion of specific backup selection."""
        displayed_backups = self.display_backup_list(backups)
        
        if not displayed_backups:
            return
        
        # Get user selection
        selection = input(f"\nSelect backup(s) to delete (1-{len(displayed_backups)}, ranges like '1-3', or 'q' to quit): ").strip().lower()
        
        if selection == 'q':
            return
        
        # Parse selection
        selected_indices = []
        try:
            for part in selection.split(','):
                part = part.strip()
                if '-' in part:
                    # Range selection (e.g., "1-3")
                    start, end = map(int, part.split('-'))
                    selected_indices.extend(range(start-1, end))
                else:
                    # Single selection
                    selected_indices.append(int(part) - 1)
        except ValueError:
            print("❌ Invalid selection format")
            return
        
        # Validate indices and get selected backups
        selected_backups = []
        for idx in selected_indices:
            if 0 <= idx < len(displayed_backups):
                selected_backups.append(displayed_backups[idx])
            else:
                print(f"❌ Invalid selection: {idx + 1}")
                return
        
        if not selected_backups:
            print("❌ No valid backups selected")
            return
        
        # Remove duplicates
        unique_backups = []
        seen_volids = set()
        for backup in selected_backups:
            volid = backup.get('volid')
            if volid not in seen_volids:
                unique_backups.append(backup)
                seen_volids.add(volid)
        
        # Confirm deletion
        print(f"\n⚠️  Confirm Deletion")
        print("=" * 30)
        print(f"You selected {len(unique_backups)} backup(s) to delete:")
        for backup in unique_backups:
            volid = backup.get('volid', 'unknown')
            filename = volid.split('/')[-1] if '/' in volid else volid
            size_gb = backup.get('size', 0) / (1024**3)
            print(f"  • {filename} ({size_gb:.2f} GB)")
        
        confirm = input(f"\nDelete {len(unique_backups)} backup(s)? (y/N): ").strip().lower()
        if confirm in ['y', 'yes']:
            self.bulk_delete_backups(unique_backups)
        else:
            print("Deletion cancelled")
    
    def _handle_delete_by_pattern(self, vm_id: str):
        """Handle deletion by pattern."""
        print("\n🔍 Delete Backups by Pattern")
        print("=" * 40)
        print("Examples:")
        print("  backup-*     - Delete backups starting with 'backup-'")
        print("  *-daily      - Delete backups ending with '-daily'")
        print("  *2024*       - Delete backups containing '2024'")
        print("  all          - Delete ALL backups (dangerous!)")
        print()
        
        pattern = input("Enter pattern: ").strip()
        if not pattern:
            print("❌ No pattern specified")
            return
        
        # Optional age filter
        age_input = input("Delete only backups older than X days (enter number or press Enter to skip): ").strip()
        max_age_days = None
        if age_input:
            try:
                max_age_days = int(age_input)
                if max_age_days <= 0:
                    print("❌ Age must be positive")
                    return
            except ValueError:
                print("❌ Invalid age format")
                return
        
        # Execute pattern-based deletion
        self.delete_backups_by_pattern(vm_id, pattern, max_age_days)
    
    def _handle_delete_all_backups(self, vm_id: str):
        """Handle deletion of all backups."""
        print(f"\n⚠️  DELETE ALL BACKUPS WARNING")
        print("=" * 50)
        print("This will permanently delete ALL backups for this VM!")
        print("This action cannot be undone!")
        print("=" * 50)
        
        confirm1 = input("Type 'DELETE ALL' to confirm: ").strip()
        if confirm1 != 'DELETE ALL':
            print("Operation cancelled")
            return
        
        confirm2 = input("Are you absolutely sure? Type 'YES' to proceed: ").strip().upper()
        if confirm2 != 'YES':
            print("Operation cancelled")
            return
        
        # Execute deletion
        self.delete_backups_by_pattern(vm_id, 'all')
    
    def _handle_delete_old_backups(self, vm_id: str):
        """Handle cleanup of old backups."""
        print("\n📋 Backup Cleanup Configuration")
        print("=" * 40)
        
        # Get keep count
        keep_input = input("Keep how many most recent backups? (default: 5): ").strip()
        keep_count = 5
        if keep_input:
            try:
                keep_count = int(keep_input)
                if keep_count < 1:
                    print("❌ Keep count must be at least 1")
                    return
            except ValueError:
                print("❌ Invalid keep count")
                return
        
        # Get max age
        age_input = input("Delete backups older than how many days? (default: 30): ").strip()
        max_age_days = 30
        if age_input:
            try:
                max_age_days = int(age_input)
                if max_age_days <= 0:
                    print("❌ Age must be positive")
                    return
            except ValueError:
                print("❌ Invalid age")
                return
        
        # Execute cleanup
        self.delete_old_backups(vm_id, keep_count, max_age_days)
    
    def bulk_operations_menu(self):
        """Interactive menu for bulk operations."""
        while True:
            try:
                print("\n🔧 Bulk Operations Menu")
                print("=" * 40)
                print("1. Bulk Start VMs")
                print("2. Bulk Stop VMs (Forceful)")
                print("3. Bulk Shutdown VMs (Graceful)")
                print("4. Bulk Create Backups")
                print("5. VM Selection Help")
                print("6. Back to Main Menu")
                print()
                
                choice = input("Select operation (1-6): ").strip()
                
                if choice == '1':  # Bulk Start
                    self.handle_bulk_start()
                elif choice == '2':  # Bulk Stop
                    self.handle_bulk_stop()
                elif choice == '3':  # Bulk Shutdown
                    self.handle_bulk_shutdown()
                elif choice == '4':  # Bulk Backup
                    self.handle_bulk_backup()
                elif choice == '5':  # Selection Help
                    self.vm_selector.display_selection_help()
                    input("Press Enter to continue...")
                elif choice == '6':  # Back
                    break
                else:
                    print("❌ Invalid choice. Please select 1-6")
                    
            except KeyboardInterrupt:
                print("\nReturning to main menu...")
                break
    
    def handle_bulk_start(self):
        """Handle bulk VM start operations."""
        print("\n🚀 Bulk Start VMs")
        print("=" * 40)
        
        all_vms = self.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        print(f"\nSelected VMs: {', '.join(vm_ids)}")
        confirm = input(f"Start {len(vm_ids)} VMs? (y/N): ").strip().lower()
        
        if confirm in ['y', 'yes']:
            self.bulk_start_vms(vm_ids)
        else:
            print("Operation cancelled")
        
        input("Press Enter to continue...")
    
    def handle_bulk_stop(self):
        """Handle bulk VM stop operations."""
        print("\n🛑 Bulk Stop VMs (Forceful)")
        print("=" * 40)
        
        all_vms = self.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        print(f"\nSelected VMs: {', '.join(vm_ids)}")
        print("⚠️  This will forcefully stop the VMs immediately")
        confirm = input(f"Stop {len(vm_ids)} VMs? (y/N): ").strip().lower()
        
        if confirm in ['y', 'yes']:
            self.bulk_stop_vms(vm_ids)
        else:
            print("Operation cancelled")
        
        input("Press Enter to continue...")
    
    def handle_bulk_shutdown(self):
        """Handle bulk VM graceful shutdown operations."""
        print("\n🔄 Bulk Shutdown VMs (Graceful)")
        print("=" * 40)
        
        all_vms = self.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        print(f"\nSelected VMs: {', '.join(vm_ids)}")
        print("ℹ️  This will send ACPI shutdown signal to VMs")
        confirm = input(f"Gracefully shutdown {len(vm_ids)} VMs? (y/N): ").strip().lower()
        
        if confirm in ['y', 'yes']:
            self.bulk_shutdown_vms(vm_ids)
        else:
            print("Operation cancelled")
        
        input("Press Enter to continue...")
    
    def handle_bulk_backup(self):
        """Handle bulk backup operations."""
        print("\n💾 Bulk Create Backups")
        print("=" * 40)
        
        all_vms = self.get_all_vms_info()
        selection = input("Enter VM selection (use 'help' for selection formats): ").strip()
        
        if selection.lower() == 'help':
            self.vm_selector.display_selection_help()
            input("Press Enter to continue...")
            return
        
        vm_ids = self.vm_selector.parse_selection(selection, all_vms)
        
        if not vm_ids:
            print("❌ No VMs selected or found")
            input("Press Enter to continue...")
            return
        
        # Select storage
        storage = self.select_storage_interactive("backup")
        if not storage:
            return
        
        # Select backup mode
        print("\nBackup modes:")
        print("1. snapshot - Fast backup using VM snapshots")
        print("2. suspend - Suspend VM during backup")
        print("3. stop - Stop VM during backup")
        
        mode_choice = input("Select backup mode (1-3): ").strip()
        mode_map = {'1': 'snapshot', '2': 'suspend', '3': 'stop'}
        mode = mode_map.get(mode_choice, 'snapshot')
        
        print(f"\nSelected VMs: {', '.join(vm_ids)}")
        print(f"Storage: {storage}")
        print(f"Mode: {mode}")
        confirm = input(f"Create backups for {len(vm_ids)} VMs? (y/N): ").strip().lower()
        
        if confirm in ['y', 'yes']:
            self.bulk_create_backups(vm_ids, storage, mode)
        else:
            print("Operation cancelled")
        
        input("Press Enter to continue...")
    
    def storage_management_menu(self):
        """Storage management menu."""
        while True:
            try:
                print("\n🗄️  Storage Management")
                print("=" * 40)
                print("1. View Backup Storages")
                print("2. View VM Disk Storages")
                print("3. Check All Storage Status")
                print("4. Back to Main Menu")
                print()
                
                choice = input("Select option (1-4): ").strip()
                
                if choice == '1':
                    self.display_backup_storage_list()
                    input("Press Enter to continue...")
                elif choice == '2':
                    self.display_vm_storage_list()
                    input("Press Enter to continue...")
                elif choice == '3':
                    self.storage_ops.check_all_storages_status()
                    input("Press Enter to continue...")
                elif choice == '4':
                    break
                else:
                    print("❌ Invalid choice. Please select 1-4")
                    
            except KeyboardInterrupt:
                print("\nReturning to main menu...")
                break