#!/usr/bin/env python3

"""
Storage Operations Module
Handles storage discovery, management, and status reporting for both VM disks and backup storage
"""

from typing import List, Dict
from proxmox_api import ProxmoxAPIError


class StorageOperations:
    """Handles storage operations for Proxmox VMs and backups."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
    
    def get_vm_storages(self) -> List[Dict]:
        """Get all available storages suitable for VM disks."""
        storages = []
        nodes = self.vm_manager.get_nodes()
        
        for node in nodes:
            try:
                node_storages = self.api._request('GET', f'/nodes/{node["node"]}/storage')
                for storage in node_storages:
                    # Check if storage supports 'images' content (VM disks)
                    storage_info = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage["storage"]}/status')
                    content = storage_info.get('content', '')
                    if 'images' in content or 'rootdir' in content:
                        storage['node'] = node['node']
                        storage['content_types'] = content
                        storages.append(storage)
            except ProxmoxAPIError:
                continue
        
        # Remove duplicates (shared storages appear on multiple nodes)
        unique_storages = {}
        for storage in storages:
            key = storage['storage']
            if key not in unique_storages:
                unique_storages[key] = storage
        
        return list(unique_storages.values())
    
    def display_vm_storage_list(self) -> List[Dict]:
        """Display available VM disk storages and return the list."""
        storages = self.get_vm_storages()
        
        if not storages:
            print("❌ No VM disk storages found")
            return []
        
        print("\nAvailable VM Disk Storages:")
        print("=" * 80)
        print(f"{'#':<3} {'Storage':<15} {'Type':<10} {'Status':<10} {'Content Types':<20} {'Free Space':<15}")
        print("-" * 80)
        
        for i, storage in enumerate(storages, 1):
            name = storage['storage']
            storage_type = storage.get('type', 'unknown')
            content_types = storage.get('content_types', 'unknown')
            
            # Get detailed status for first node that has this storage
            try:
                status_info = self.api._request('GET', f'/nodes/{storage["node"]}/storage/{name}/status')
                
                if status_info.get('active'):
                    status = "🟢 active"
                else:
                    status = "🔴 inactive"
                
                # Convert bytes to human-readable format
                avail = status_info.get('avail', 0)
                if avail > 0:
                    free_gb = avail / (1024**3)
                    free_space = f"{free_gb:.1f} GB"
                else:
                    free_space = "N/A"
                
            except ProxmoxAPIError:
                status = "❌ error"
                free_space = "N/A"
            
            print(f"{i:<3} {name:<15} {storage_type:<10} {status:<10} {content_types:<20} {free_space:<15}")
        
        print("-" * 80)
        print(f"Total VM disk storages: {len(storages)}")
        
        return storages
    
    def get_backup_storages(self) -> List[Dict]:
        """Get all available backup storages from all nodes."""
        storages = []
        nodes = self.vm_manager.get_nodes()
        
        for node in nodes:
            try:
                node_storages = self.api._request('GET', f'/nodes/{node["node"]}/storage')
                for storage in node_storages:
                    # Check if storage supports backup content
                    storage_info = self.api._request('GET', f'/nodes/{node["node"]}/storage/{storage["storage"]}/status')
                    if storage_info.get('content', '').find('backup') != -1 or storage_info.get('content', '').find('vztmpl') != -1:
                        storage['node'] = node['node']
                        storages.append(storage)
            except ProxmoxAPIError:
                continue
        
        # Remove duplicates (shared storages appear on multiple nodes)
        unique_storages = {}
        for storage in storages:
            key = storage['storage']
            if key not in unique_storages:
                unique_storages[key] = storage
        
        return list(unique_storages.values())
    
    def display_backup_storage_list(self) -> List[Dict]:
        """Display available backup storages and return the list."""
        storages = self.get_backup_storages()
        
        if not storages:
            print("❌ No backup-capable storages found")
            return []
        
        print("\nAvailable Backup Storages:")
        print("=" * 70)
        print(f"{'#':<3} {'Storage':<15} {'Type':<10} {'Status':<10} {'Free Space':<15} {'Total Space'}")
        print("-" * 70)
        
        for i, storage in enumerate(storages, 1):
            name = storage['storage']
            storage_type = storage.get('type', 'unknown')
            
            # Get detailed status for first node that has this storage
            try:
                status_info = self.api._request('GET', f'/nodes/{storage["node"]}/storage/{name}/status')
                
                if status_info.get('active'):
                    status = "🟢 active"
                else:
                    status = "🔴 inactive"
                
                # Convert bytes to human-readable format
                avail = status_info.get('avail', 0)
                total = status_info.get('total', 0)
                
                if total > 0:
                    free_gb = avail / (1024**3)
                    total_gb = total / (1024**3)
                    free_space = f"{free_gb:.1f} GB"
                    total_space = f"{total_gb:.1f} GB"
                else:
                    free_space = "N/A"
                    total_space = "N/A"
                
            except ProxmoxAPIError:
                status = "❌ error"
                free_space = "N/A"
                total_space = "N/A"
            
            print(f"{i:<3} {name:<15} {storage_type:<10} {status:<10} {free_space:<15} {total_space}")
        
        print("-" * 70)
        print(f"Total storages: {len(storages)}")
        
        return storages
    
    def get_storage_status(self, storage_name: str, node: str = None) -> Dict:
        """Get detailed status information for a specific storage."""
        if not node:
            # Try to find a node that has access to this storage
            nodes = self.vm_manager.get_nodes()
            for n in nodes:
                try:
                    status = self.api._request('GET', f'/nodes/{n["node"]}/storage/{storage_name}/status')
                    status['node'] = n['node']
                    return status
                except ProxmoxAPIError:
                    continue
            return {}
        else:
            try:
                status = self.api._request('GET', f'/nodes/{node}/storage/{storage_name}/status')
                status['node'] = node
                return status
            except ProxmoxAPIError:
                return {}
    
    def validate_storage_space(self, storage_name: str, required_space_gb: float = 10.0) -> bool:
        """Validate that storage has sufficient free space."""
        status = self.get_storage_status(storage_name)
        if not status:
            print(f"⚠️  Could not get status for storage '{storage_name}'")
            return False
        
        avail_bytes = status.get('avail', 0)
        avail_gb = avail_bytes / (1024**3)
        
        if avail_gb < required_space_gb:
            print(f"⚠️  Storage '{storage_name}' has insufficient space: {avail_gb:.1f} GB available, {required_space_gb:.1f} GB required")
            return False
        
        return True
    
    def select_storage_interactive(self, storage_type: str = "backup") -> str:
        """Interactive storage selection."""
        if storage_type == "backup":
            storages = self.display_backup_storage_list()
        else:
            storages = self.display_vm_storage_list()
        
        if not storages:
            return ""
        
        while True:
            try:
                choice = input(f"\nSelect storage (1-{len(storages)}, or 'q' to quit): ").strip().lower()
                
                if choice == 'q':
                    return ""
                
                storage_num = int(choice)
                if 1 <= storage_num <= len(storages):
                    selected_storage = storages[storage_num - 1]['storage']
                    print(f"✅ Selected storage: {selected_storage}")
                    return selected_storage
                else:
                    print(f"❌ Invalid choice. Please enter 1-{len(storages)}")
                    
            except ValueError:
                print("❌ Invalid input. Please enter a number")
            except KeyboardInterrupt:
                print("\nSelection cancelled")
                return ""
    
    def check_all_storages_status(self):
        """Check status of all storages in the cluster."""
        print("\n🔍 Checking all storage status...")
        print("=" * 80)
        
        # Check backup storages
        backup_storages = self.get_backup_storages()
        if backup_storages:
            print("\n📦 Backup Storages:")
            print("-" * 40)
            for storage in backup_storages:
                name = storage['storage']
                status_info = self.get_storage_status(name)
                
                if status_info:
                    active = status_info.get('active', False)
                    status_str = "🟢 active" if active else "🔴 inactive"
                    
                    avail = status_info.get('avail', 0)
                    total = status_info.get('total', 0)
                    
                    if total > 0:
                        avail_gb = avail / (1024**3)
                        total_gb = total / (1024**3)
                        usage_pct = ((total - avail) / total) * 100
                        print(f"  {name}: {status_str} - {avail_gb:.1f} GB free / {total_gb:.1f} GB total ({usage_pct:.1f}% used)")
                    else:
                        print(f"  {name}: {status_str} - Size unknown")
                else:
                    print(f"  {name}: ❌ Status unavailable")
        
        # Check VM disk storages
        vm_storages = self.get_vm_storages()
        if vm_storages:
            print("\n💽 VM Disk Storages:")
            print("-" * 40)
            for storage in vm_storages:
                name = storage['storage']
                # Skip if already shown in backup storages
                if not any(b['storage'] == name for b in backup_storages):
                    status_info = self.get_storage_status(name)
                    
                    if status_info:
                        active = status_info.get('active', False)
                        status_str = "🟢 active" if active else "🔴 inactive"
                        
                        avail = status_info.get('avail', 0)
                        total = status_info.get('total', 0)
                        
                        if total > 0:
                            avail_gb = avail / (1024**3)
                            total_gb = total / (1024**3)
                            usage_pct = ((total - avail) / total) * 100
                            print(f"  {name}: {status_str} - {avail_gb:.1f} GB free / {total_gb:.1f} GB total ({usage_pct:.1f}% used)")
                        else:
                            print(f"  {name}: {status_str} - Size unknown")
                    else:
                        print(f"  {name}: ❌ Status unavailable")
        
        print("=" * 80)