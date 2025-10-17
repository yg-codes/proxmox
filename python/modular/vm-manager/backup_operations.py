#!/usr/bin/env python3

"""
Backup Operations Module
Handles all VM backup operations including creation, restoration, listing, and verification
"""

from datetime import datetime
from typing import List, Dict, Optional
from proxmox_api import ProxmoxAPIError


class BackupOperations:
    """Handles VM backup operations for Proxmox."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
    
    def create_backup(self, vmid: str, storage: str, mode: str, compress: str = 'zstd') -> bool:
        """Create a VM backup."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            print("❌ Could not find node for VM")
            return False
        
        try:
            print(f"\n🔄 Creating backup...")
            print(f"  Storage: {storage}")
            print(f"  Mode: {mode}")
            print(f"  Compression: {compress}")
            
            # Prepare backup data
            backup_data = {
                'vmid': vmid,
                'storage': storage,
                'mode': mode,
                'compress': compress,
                'remove': '0'  # Don't remove old backups
            }
            
            # Create backup
            task_id = self.api._request('POST', f'/nodes/{node}/vzdump', data=backup_data)
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Backup for VM {vmid}")
            
            if success:
                print("✅ Backup completed successfully!")
                
                # Try to get backup file info
                try:
                    # List backups for this VM
                    backups = self.api._request('GET', f'/nodes/{node}/storage/{storage}/content', 
                                              params={'vmid': vmid})
                    
                    # Find the most recent backup
                    if backups:
                        latest_backup = max(backups, key=lambda x: x.get('ctime', 0))
                        if latest_backup:
                            size_gb = latest_backup.get('size', 0) / (1024**3)
                            print(f"  Backup file: {latest_backup.get('volid', 'unknown')}")
                            print(f"  Size: {size_gb:.2f} GB")
                except:
                    pass
            else:
                print("❌ Backup failed!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to create backup: {e.message}")
            return False
    
    def create_backup_silent(self, vmid: str, storage: str, mode: str, compress: str = 'zstd') -> bool:
        """Create a VM backup without output (for bulk operations)."""
        node = self.vm_manager.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            backup_data = {
                'vmid': vmid,
                'storage': storage,
                'mode': mode,
                'compress': compress,
                'remove': '0'
            }
            
            task_id = self.api._request('POST', f'/nodes/{node}/vzdump', data=backup_data)
            # Monitor task without output
            return self.vm_manager._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False
    
    def list_backups_for_vm(self, vmid: str, storage: str = None) -> List[Dict]:
        """List all backups for a specific VM from specified storage or all storages."""
        all_backups = []
        
        if storage:
            # Check specific storage
            storages_to_check = [{'storage': storage}]
        else:
            # Check all backup-capable storages
            from storage_operations import StorageOperations
            storage_ops = StorageOperations(self.api, self.vm_manager)
            storages_to_check = storage_ops.get_backup_storages()
        
        for storage_info in storages_to_check:
            storage_name = storage_info['storage']
            storage_accessed = False
            
            # Try each node until we find one that can access this storage
            nodes = self.vm_manager.get_nodes()
            for node in nodes:
                try:
                    # Try with backup content filter first
                    contents = []
                    try:
                        contents = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage_name}/content',
                                                   params={'content': 'backup'})
                    except ProxmoxAPIError:
                        pass  # Will use fallback below
                    
                    # Always also try without filter to catch files that might not be properly tagged
                    try:
                        all_contents = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage_name}/content')
                        manual_contents = [item for item in all_contents if item.get('content') == 'backup' or 'vzdump' in item.get('volid', '')]
                        
                        # Merge results, avoiding duplicates
                        seen_volids = {item.get('volid') for item in contents}
                        for item in manual_contents:
                            if item.get('volid') not in seen_volids:
                                contents.append(item)
                    except ProxmoxAPIError:
                        # If both methods fail, we'll have an empty contents list
                        pass
                    
                    # Filter for backups of this VM
                    for item in contents:
                        # Enhanced backup identification logic
                        is_backup = False
                        item_vmid = item.get('vmid')
                        volid = item.get('volid', '')
                        
                        # Method 1: Direct VMID match (handle both string and int)
                        if item_vmid is not None and str(item_vmid) == str(vmid):
                            is_backup = True
                        
                        # Method 2: Check volid pattern for backup files
                        # More comprehensive pattern matching
                        backup_patterns = [
                            f'vzdump-qemu-{vmid}-',
                            f'vzdump-lxc-{vmid}-',
                            f'backup-{vmid}-',
                            f'vm-{vmid}-'
                        ]
                        
                        for pattern in backup_patterns:
                            if pattern in volid:
                                is_backup = True
                                # Extract VMID from volid if not set in item
                                if item_vmid is None:
                                    item['vmid'] = vmid
                                break
                        
                        # Method 3: Parse volid for VMID (backup files often contain VMID)
                        if not is_backup and 'vzdump' in volid:
                            try:
                                # Extract VMID from patterns like vzdump-qemu-123-2024_01_01-12_00_00.vma.zst
                                parts = volid.split('-')
                                if len(parts) >= 3:
                                    extracted_vmid = parts[2]
                                    if extracted_vmid == str(vmid):
                                        is_backup = True
                                        item['vmid'] = vmid
                            except (IndexError, ValueError):
                                pass
                        
                        if is_backup:
                            item['storage'] = storage_name
                            item['node'] = node['node']
                            # Ensure content type is set
                            if 'content' not in item:
                                item['content'] = 'backup'
                            # Add to list if not already present (avoid duplicates)
                            if not any(existing['volid'] == item['volid'] for existing in all_backups):
                                all_backups.append(item)
                    
                    storage_accessed = True
                    # Continue checking other nodes for local storage files
                    
                except ProxmoxAPIError as e:
                    # Only show error for debugging if specifically requested storage
                    if storage and len(storages_to_check) == 1:
                        print(f"  Debug: Could not access storage '{storage_name}' from node '{node['node']}': {e.message}")
                    continue  # Try next node
            
            # If no node could access the storage, show warning for specific storage requests
            if not storage_accessed and storage and len(storages_to_check) == 1:
                print(f"  ⚠️  Warning: Storage '{storage_name}' is not accessible from any node")
        
        # Sort by creation time (newest first)
        all_backups.sort(key=lambda x: x.get('ctime', 0), reverse=True)
        
        return all_backups
    
    def list_all_backups_in_storage(self, storage: str) -> List[Dict]:
        """List ALL backups in a storage, not filtered by VM."""
        all_backups = []
        
        # Try each node until we find one that can access this storage
        nodes = self.vm_manager.get_nodes()
        for node in nodes:
            try:
                # Try with backup content filter first
                contents = []
                try:
                    contents = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage}/content',
                                               params={'content': 'backup'})
                except ProxmoxAPIError:
                    pass  # Will use fallback below
                
                # Always also try without filter to catch files that might not be properly tagged
                try:
                    all_contents = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage}/content')
                    manual_contents = [item for item in all_contents if item.get('content') == 'backup' or 'vzdump' in item.get('volid', '')]
                    
                    # Merge results, avoiding duplicates
                    seen_volids = {item.get('volid') for item in contents}
                    for item in manual_contents:
                        if item.get('volid') not in seen_volids:
                            contents.append(item)
                except ProxmoxAPIError:
                    # If both methods fail, we'll have an empty contents list
                    pass
                
                # Process all backup items
                for item in contents:
                    item['storage'] = storage
                    item['node'] = node['node']
                    # Ensure content type is set
                    if 'content' not in item:
                        item['content'] = 'backup'
                    
                    # Try to extract VMID from volid if not present
                    if 'vmid' not in item or item.get('vmid') is None:
                        volid = item.get('volid', '')
                        if 'vzdump' in volid:
                            try:
                                # Extract VMID from patterns like vzdump-qemu-123-2024_01_01-12_00_00.vma.zst
                                parts = volid.split('-')
                                if len(parts) >= 3:
                                    item['vmid'] = parts[2]
                            except (IndexError, ValueError):
                                pass
                    
                    all_backups.append(item)
                
                # Continue checking other nodes for local storage files
            except ProxmoxAPIError as e:
                continue  # Try next node
        
        return all_backups
    
    def display_backup_list(self, backups: List[Dict]) -> List[Dict]:
        """Display formatted list of backups."""
        if not backups:
            print("❌ No backups found")
            return []
        
        print("\nAvailable Backups (Newest First):")
        print("=" * 110)
        print(f"{'#':<3} {'Backup File':<50} {'Size (GB)':<10} {'Created':<20} {'Storage':<15} {'Node'}")
        print("-" * 110)
        
        for i, backup in enumerate(backups, 1):
            volid = backup.get('volid', 'unknown')
            # Extract just the filename from volid
            filename = volid.split('/')[-1] if '/' in volid else volid
            
            size_gb = backup.get('size', 0) / (1024**3)
            
            # Format creation time
            ctime = backup.get('ctime', 0)
            if ctime:
                created = datetime.fromtimestamp(ctime).strftime('%Y-%m-%d %H:%M:%S')
            else:
                created = 'Unknown'
            
            storage = backup.get('storage', 'unknown')
            node = backup.get('node', 'unknown')
            
            # Truncate filename if too long
            if len(filename) > 49:
                filename = filename[:46] + "..."
            
            print(f"{i:<3} {filename:<50} {size_gb:<10.2f} {created:<20} {storage:<15} {node}")
        
        print("-" * 110)
        print(f"Total backups: {len(backups)}")
        
        return backups
    
    def get_backup_config(self, volid: str, node: str) -> Dict:
        """Extract configuration from a backup file."""
        try:
            # Get backup configuration
            config = self.api._request('GET', f'/nodes/{node}/vzdump/extractconfig', 
                                     params={'volume': volid})
            return config
        except ProxmoxAPIError as e:
            print(f"❌ Failed to extract backup configuration: {e.message}")
            return {}
    
    def restore_backup(self, vmid: str, volid: str, node: str, storage: str = None) -> bool:
        """Restore a VM from backup."""
        try:
            print(f"\n🔄 Restoring VM from backup...")
            print(f"  Backup: {volid}")
            print(f"  Target VMID: {vmid}")
            
            # Prepare restore data
            restore_data = {
                'vmid': vmid,
                'archive': volid,
                'force': '1'  # Overwrite existing VM
            }
            
            # If storage is specified, use it
            if storage:
                restore_data['storage'] = storage
            
            # Execute restore
            task_id = self.api._request('POST', f'/nodes/{node}/qemu', data=restore_data)
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Restore for VM {vmid}")
            
            if success:
                print("✅ Restore completed successfully!")
            else:
                print("❌ Restore failed!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to restore backup: {e.message}")
            return False
    
    def check_and_handle_protection(self, vmid: str) -> bool:
        """Check if VM has protection enabled and handle it."""
        vm_info = self.vm_manager.get_vm_info(vmid)
        if not vm_info:
            print("❌ Could not get VM info to check protection")
            return False
        
        config = vm_info.get('config', {})
        protection = config.get('protection', '0')
        
        # Check if protection is enabled (protection = 1)
        if protection == '1' or protection == 1:
            print("\n⚠️  VM PROTECTION DETECTED")
            print("=" * 50)
            print("This VM has protection mode enabled, which prevents:")
            print("  • VM deletion")
            print("  • Configuration changes") 
            print("  • Backup restore operations")
            print("=" * 50)
            print()
            print("Options:")
            print("1. Disable protection and continue with restore")
            print("2. Cancel restore operation")
            print()
            
            choice = input("Select option (1-2): ").strip()
            
            if choice == '1':
                print("\n🔓 Disabling VM protection...")
                node = self.vm_manager.find_vm_node(vmid)
                if not node:
                    print("❌ Could not find VM node to disable protection")
                    return False
                
                try:
                    # Disable protection
                    self.api._request('PUT', f'/nodes/{node}/qemu/{vmid}/config', 
                                    data={'protection': '0'})
                    print("✅ VM protection disabled successfully")
                    return True
                except ProxmoxAPIError as e:
                    print(f"❌ Failed to disable protection: {e.message}")
                    return False
            else:
                print("Restore operation cancelled")
                return False
        
        # Protection not enabled or already disabled
        return True
    
    def delete_backup(self, volid: str, node: str, storage: str) -> bool:
        """Delete a specific backup file."""
        try:
            print(f"\n🗑️  Deleting backup...")
            print(f"  Backup: {volid}")
            print(f"  Storage: {storage}")
            print(f"  Node: {node}")
            
            # Delete backup via storage content API
            task_id = self.api._request('DELETE', f'/nodes/{node}/storage/{storage}/content/{volid}')
            
            # Monitor task progress
            success = self.vm_manager.monitor_task(node, task_id, f"Backup deletion")
            
            if success:
                print("✅ Backup deleted successfully!")
            else:
                print("❌ Backup deletion failed!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to delete backup: {e.message}")
            return False
    
    def delete_backup_silent(self, volid: str, node: str, storage: str) -> bool:
        """Delete a backup without output (for bulk operations)."""
        try:
            # Delete backup via storage content API
            task_id = self.api._request('DELETE', f'/nodes/{node}/storage/{storage}/content/{volid}')
            
            # Monitor task without output
            return self.vm_manager._monitor_task_silent(node, task_id)
            
        except ProxmoxAPIError:
            return False
    
    def bulk_delete_backups(self, backups_to_delete: List[Dict], max_workers: int = 2):
        """Delete multiple backups concurrently."""
        from bulk_operations import BulkOperationManager, BulkOperationResult
        import time
        from concurrent.futures import ThreadPoolExecutor, as_completed
        
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🗑️  Deleting {len(backups_to_delete)} backup(s)")
        print(f"Max concurrent operations: {max_workers}")
        print("=" * 60)
        
        def delete_single_backup(backup: Dict) -> BulkOperationResult:
            start_time = time.time()
            volid = backup.get('volid', 'unknown')
            try:
                # Delete backup (silent version for bulk operations)
                success = self.delete_backup_silent(
                    backup.get('volid'),
                    backup.get('node'),
                    backup.get('storage')
                )
                message = f"Deleted {volid}" if success else f"Failed to delete {volid}"
                return BulkOperationResult(volid, "delete_backup", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(volid, "delete_backup", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_backup = {executor.submit(delete_single_backup, backup): backup for backup in backups_to_delete}
            
            for future in as_completed(future_to_backup):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(len(backups_to_delete), "Delete Backups")
        
        operation_manager.print_summary("Bulk Delete Backups")
        return operation_manager
    
    def delete_backups_by_pattern(self, vm_id: str, pattern: str, max_age_days: int = None) -> bool:
        """Delete backups matching a pattern or age criteria."""
        # Get all backups for the VM
        backups = self.list_backups_for_vm(vm_id)
        if not backups:
            print(f"❌ No backups found for VM {vm_id}")
            return False
        
        import fnmatch
        import time
        from datetime import datetime, timedelta
        
        backups_to_delete = []
        
        # Filter backups by pattern
        for backup in backups:
            volid = backup.get('volid', '')
            filename = volid.split('/')[-1] if '/' in volid else volid
            
            # Pattern matching
            if pattern and pattern.lower() != 'all':
                if not fnmatch.fnmatch(filename.lower(), pattern.lower()):
                    continue
            
            # Age filtering
            if max_age_days:
                ctime = backup.get('ctime', 0)
                if ctime:
                    backup_date = datetime.fromtimestamp(ctime)
                    cutoff_date = datetime.now() - timedelta(days=max_age_days)
                    if backup_date > cutoff_date:
                        continue  # Skip backups newer than max_age_days
                else:
                    continue  # Skip backups without creation time
            
            backups_to_delete.append(backup)
        
        if not backups_to_delete:
            print(f"❌ No backups match the criteria (pattern: '{pattern}', max_age: {max_age_days} days)")
            return False
        
        # Display backups to be deleted
        print(f"\n🗑️  Backups to delete for VM {vm_id}:")
        print("-" * 80)
        for i, backup in enumerate(backups_to_delete, 1):
            volid = backup.get('volid', 'unknown')
            filename = volid.split('/')[-1] if '/' in volid else volid
            size_gb = backup.get('size', 0) / (1024**3)
            
            ctime = backup.get('ctime', 0)
            if ctime:
                created = datetime.fromtimestamp(ctime).strftime('%Y-%m-%d %H:%M')
            else:
                created = 'Unknown'
            
            print(f"  {i:2d}. {filename} ({size_gb:.2f} GB, {created})")
        
        print("-" * 80)
        print(f"Total: {len(backups_to_delete)} backups")
        
        # Confirmation
        if pattern and pattern.lower() == 'all':
            confirm_text = "DELETE ALL"
            print(f"\n⚠️  WARNING: This will delete ALL {len(backups_to_delete)} backups!")
            confirm = input(f"Type '{confirm_text}' to confirm: ").strip()
            if confirm != confirm_text:
                print("Deletion cancelled")
                return False
        else:
            confirm = input(f"\nDelete {len(backups_to_delete)} backup(s)? (y/N): ").strip().lower()
            if confirm not in ['y', 'yes']:
                print("Deletion cancelled")
                return False
        
        # Execute bulk deletion
        result = self.bulk_delete_backups(backups_to_delete)
        success_count = len([r for r in result.results if r.success])
        
        return success_count == len(backups_to_delete)
    
    def delete_old_backups(self, vm_id: str, keep_count: int = 5, max_age_days: int = 30) -> bool:
        """Delete old backups keeping only the most recent ones."""
        backups = self.list_backups_for_vm(vm_id)
        if not backups:
            print(f"❌ No backups found for VM {vm_id}")
            return False
        
        # Sort by creation time (newest first)
        sorted_backups = sorted(backups, key=lambda x: x.get('ctime', 0), reverse=True)
        
        from datetime import datetime, timedelta
        cutoff_date = datetime.now() - timedelta(days=max_age_days)
        
        backups_to_delete = []
        
        # Keep the most recent 'keep_count' backups, delete the rest
        for i, backup in enumerate(sorted_backups):
            # Always keep the newest 'keep_count' backups
            if i < keep_count:
                continue
            
            # Delete backups older than max_age_days
            ctime = backup.get('ctime', 0)
            if ctime:
                backup_date = datetime.fromtimestamp(ctime)
                if backup_date < cutoff_date:
                    backups_to_delete.append(backup)
        
        if not backups_to_delete:
            print(f"✅ No old backups to delete for VM {vm_id}")
            print(f"   Keeping {len(sorted_backups)} backups (within keep_count={keep_count} and max_age={max_age_days}d)")
            return True
        
        # Display cleanup plan
        print(f"\n📋 Backup cleanup plan for VM {vm_id}:")
        print(f"   Total backups: {len(sorted_backups)}")
        print(f"   Keep most recent: {keep_count}")
        print(f"   Keep newer than: {max_age_days} days")
        print(f"   Will delete: {len(backups_to_delete)} old backups")
        
        confirm = input(f"\nProceed with cleanup? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Cleanup cancelled")
            return False
        
        # Execute bulk deletion
        result = self.bulk_delete_backups(backups_to_delete)
        success_count = len([r for r in result.results if r.success])
        
        return success_count == len(backups_to_delete)