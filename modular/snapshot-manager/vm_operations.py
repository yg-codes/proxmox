#!/usr/bin/env python3

"""
VM Operations Module
Handles VM management operations including status checking, starting, and information retrieval
"""

import re
import time
from typing import List, Dict, Optional, Tuple
from proxmox_api import ProxmoxAPIError


class VMOperations:
    """Handles VM operations for Proxmox."""
    
    def __init__(self, api_client):
        self.api = api_client
        self.nodes_cache = {}
    
    def get_nodes(self) -> List[Dict]:
        """Get all nodes in the cluster."""
        if not self.nodes_cache:
            try:
                nodes = self.api._request('GET', '/nodes')
                self.nodes_cache = {node['node']: node for node in nodes}
            except ProxmoxAPIError as e:
                print(f"Error getting nodes: {e.message}")
                return []
        return list(self.nodes_cache.values())
    
    def find_vm_node(self, vmid: str) -> Optional[str]:
        """Find which node a VM is on."""
        nodes = self.get_nodes()
        for node in nodes:
            try:
                self.api._request('GET', f'/nodes/{node["node"]}/qemu/{vmid}/status/current')
                return node['node']
            except ProxmoxAPIError:
                continue
        return None
    
    def get_all_vms(self) -> List[Dict]:
        """Get all VMs from all nodes."""
        all_vms = []
        nodes = self.get_nodes()
        
        for node in nodes:
            try:
                vms = self.api._request('GET', f'/nodes/{node["node"]}/qemu')
                for vm in vms:
                    vm['node'] = node['node']
                    all_vms.append(vm)
            except ProxmoxAPIError as e:
                print(f"⚠️  Warning: Could not get VMs from node {node['node']}: {e.message}")
        
        return all_vms
    
    def get_vm_info(self, vmid: str) -> Optional[Dict]:
        """Get comprehensive VM information."""
        node = self.find_vm_node(vmid)
        if not node:
            return None
        
        try:
            # Get current status
            status = self.api._request('GET', f'/nodes/{node}/qemu/{vmid}/status/current')
            
            # Get configuration
            config = self.api._request('GET', f'/nodes/{node}/qemu/{vmid}/config')
            
            # Combine information
            vm_info = {
                'vmid': vmid,
                'node': node,
                'name': status.get('name', f'vm-{vmid}'),
                'status': status.get('status', 'unknown'),
                'running': status.get('status') == 'running',
                'cpu_usage': status.get('cpu', 0) * 100,
                'memory_usage': status.get('mem', 0),
                'memory_max': status.get('maxmem', 0),
                'uptime': status.get('uptime', 0),
                'pid': status.get('pid'),
                'config': config
            }
            
            return vm_info
            
        except ProxmoxAPIError:
            return None
    
    def get_all_vms_info(self) -> List[Dict]:
        """Get comprehensive info for all VMs."""
        all_vms = []
        nodes = self.get_nodes()
        
        for node in nodes:
            try:
                vms = self.api._request('GET', f'/nodes/{node["node"]}/qemu')
                for vm in vms:
                    vm_info = {
                        'vmid': str(vm['vmid']),
                        'name': vm.get('name', f'VM-{vm["vmid"]}'),
                        'node': node['node'],
                        'running': vm.get('status') == 'running',
                        'status': vm.get('status', 'unknown'),
                        'cpu_usage': vm.get('cpu', 0),
                        'memory_usage': vm.get('mem', 0),
                        'memory_max': vm.get('maxmem', 0),
                        'uptime': vm.get('uptime', 0)
                    }
                    all_vms.append(vm_info)
            except ProxmoxAPIError:
                continue
        
        # Sort by VM ID
        all_vms.sort(key=lambda x: int(x['vmid']))
        return all_vms
    
    def get_vm_status_detailed(self, vmid: str) -> Tuple[bool, str, str]:
        """Get detailed VM status with colored indicator."""
        vm_info = self.get_vm_info(vmid)
        
        if not vm_info:
            return False, "⚠️ error", "VM not found or inaccessible"
        
        is_running = vm_info['running']
        
        if is_running:
            cpu_usage = vm_info['cpu_usage']
            memory_usage = vm_info['memory_usage'] // (1024**2)  # MB
            status_display = f"🟢 running (CPU: {cpu_usage:.1f}%, RAM: {memory_usage}MB)"
            status_details = f"Node: {vm_info['node']}, PID: {vm_info.get('pid', 'N/A')}, Uptime: {vm_info['uptime']}s"
        else:
            status_display = "🔴 stopped"
            status_details = f"Node: {vm_info['node']}"
        
        return is_running, status_display, status_details
    
    def get_vm_name(self, vmid: str) -> Optional[str]:
        """Get VM name and extract the clean name according to rules."""
        vm_info = self.get_vm_info(vmid)
        if not vm_info:
            return None
        
        full_name = vm_info['name']
        
        # Extract the 3rd section separated by hyphens
        name_parts = full_name.split('-')
        if len(name_parts) >= 3:
            clean_name = '-'.join(name_parts[2:])
        else:
            # Fall back to removing common prefixes
            clean_name = full_name
            if clean_name.startswith('xsf-dev-'):
                clean_name = clean_name[8:]
            elif clean_name.startswith('xaj-prod-'):
                clean_name = clean_name[9:]
        
        return clean_name if clean_name else full_name
    
    def get_full_vm_name(self, vmid: str) -> Optional[str]:
        """Get the full VM name."""
        vm_info = self.get_vm_info(vmid)
        return vm_info['name'] if vm_info else None
    
    def truncate_vm_name_intelligently(self, vm_name: str, max_length: int) -> str:
        """Intelligently truncate VM name while preserving meaningful parts."""
        if len(vm_name) <= max_length:
            return vm_name
            
        # Strategy 1: Try to keep the last number/identifier
        number_match = re.search(r'^(.+)([0-9]+)$', vm_name)
        if number_match:
            base_part = number_match.group(1)
            number_part = number_match.group(2)
            base_length = max_length - len(number_part)
            
            if base_length > 0:
                return base_part[:base_length] + number_part
                
        # Strategy 2: Break at word boundaries
        temp_name = vm_name
        while len(temp_name) > max_length and '-' in temp_name:
            temp_name = temp_name.rsplit('-', 1)[0]
            
        if len(temp_name) <= max_length and len(temp_name) > max_length // 2:
            return temp_name
            
        # Strategy 3: Simple truncation
        return vm_name[:max_length]
    
    def monitor_task(self, node: str, task_id: str, description: str = "Task") -> bool:
        """Monitor a Proxmox task until completion."""
        print(f"  🔄 {description} started (Task: {task_id})")
        
        start_time = time.time()
        last_status = ""
        
        while True:
            try:
                task_status = self.api._request('GET', f'/nodes/{node}/tasks/{task_id}/status')
                
                status = task_status.get('status', 'unknown')
                
                if status != last_status:
                    if status == 'running':
                        elapsed = int(time.time() - start_time)
                        print(f"  ⏳ {description} in progress... ({elapsed}s)")
                    last_status = status
                
                if status == 'stopped':
                    exit_status = task_status.get('exitstatus')
                    if exit_status == 'OK':
                        elapsed = int(time.time() - start_time)
                        print(f"  ✅ {description} completed successfully ({elapsed}s)")
                        return True
                    else:
                        print(f"  ⚠️ {description} failed: {exit_status}")
                        return False
                        
                time.sleep(2)
                
            except ProxmoxAPIError as e:
                print(f"  ⚠️  Error monitoring task: {e.message}")
                return False
            except KeyboardInterrupt:
                print(f"\n  ⏸️  Task monitoring interrupted. Task {task_id} may still be running.")
                return False
    
    def _monitor_task_silent(self, node: str, task_id: str) -> bool:
        """Monitor a Proxmox task silently (for bulk operations)."""
        while True:
            try:
                task_status = self.api._request('GET', f'/nodes/{node}/tasks/{task_id}/status')
                status = task_status.get('status', 'unknown')
                
                if status == 'stopped':
                    exit_status = task_status.get('exitstatus', '')
                    return exit_status == 'OK'
                elif status in ['error', 'cancelled']:
                    return False
                
                time.sleep(2)
                
            except ProxmoxAPIError:
                return False
    
    def start_vm(self, vmid: str) -> bool:
        """Start a VM."""
        node = self.find_vm_node(vmid)
        if not node:
            print("❌ Could not find node for VM")
            return False
        
        try:
            print("🚀 Starting VM...")
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/start')
            
            # Monitor task progress
            success = self.monitor_task(node, task_id, f"VM {vmid} startup")
            
            if success:
                print("✅ VM started successfully!")
            else:
                print("❌ Failed to start VM!")
            
            return success
            
        except ProxmoxAPIError as e:
            print(f"❌ Failed to start VM: {e.message}")
            return False
    
    def _start_vm_silent(self, vmid: str) -> bool:
        """Start a VM without output (for bulk operations)."""
        node = self.find_vm_node(vmid)
        if not node:
            return False
        
        try:
            task_id = self.api._request('POST', f'/nodes/{node}/qemu/{vmid}/status/start')
            # Monitor task without output
            return self._monitor_task_silent(node, task_id)
        except ProxmoxAPIError:
            return False
    
    def display_vm_list_interactive(self, snapshot_operations=None):
        """Display enhanced VM list for interactive mode with snapshot counts."""
        print("\nAvailable VMs:")
        print("=" * 100)
        print(f"{'VMID':<8} {'Name':<25} {'Status':<20} {'Node':<12} {'Snapshots'}")
        print("-" * 100)
        
        all_vms = self.get_all_vms()
        if not all_vms:
            print("No VMs found")
            return
        
        for vm in sorted(all_vms, key=lambda x: int(x['vmid'])):
            vmid = vm['vmid']
            name = vm.get('name', f'vm-{vmid}')[:24]
            
            # Get detailed status
            vm_info = self.get_vm_info(str(vmid))
            if vm_info:
                if vm_info['running']:
                    status = "🟢 running"
                else:
                    status = "🔴 stopped"
                node = vm_info['node']
            else:
                status = "⚠️ error"
                node = vm.get('node', 'unknown')
            
            # Count snapshots (excluding 'current' state) if snapshot_operations is provided
            snapshot_info = "No snapshots"
            if snapshot_operations:
                snapshots = snapshot_operations.get_snapshots(str(vmid))
                snapshot_count = len([s for s in snapshots if s.get('name') != 'current'])
                snapshot_info = f"{snapshot_count} snapshots" if snapshot_count > 0 else "No snapshots"
            
            print(f"{vmid:<8} {name:<25} {status:<20} {node:<12} {snapshot_info}")
        
        print("-" * 100)
        print(f"Total VMs: {len(all_vms)}")
    
    def show_vm_details(self, vmid: str):
        """Show comprehensive VM details."""
        print(f"\n{'='*60}")
        print(f"VM {vmid} - Detailed Information")
        print(f"{'='*60}")
        
        vm_info = self.get_vm_info(vmid)
        if not vm_info:
            print("⚠️ VM not found or inaccessible")
            return
        
        # Basic info
        print(f"Name: {vm_info['name']}")
        print(f"Node: {vm_info['node']}")
        
        # Status
        is_running, status_display, status_details = self.get_vm_status_detailed(vmid)
        print(f"Status: {status_display}")
        if status_details:
            print(f"Details: {status_details}")
        
        # Protection status
        config = vm_info.get('config', {})
        protection = config.get('protection', '0')
        if protection == '1' or protection == 1:
            print(f"Protection: 🔒 ENABLED (prevents deletion/changes)")
        else:
            print(f"Protection: 🔓 disabled")
        
        # Resource usage
        if is_running:
            cpu_usage = vm_info.get('cpu_usage', 0)
            memory_usage = vm_info.get('memory_usage', 0) // (1024**2)  # MB
            memory_max = vm_info.get('memory_max', 0) // (1024**2)  # MB
            uptime_seconds = vm_info.get('uptime', 0)
            
            # Format uptime
            days = uptime_seconds // 86400
            hours = (uptime_seconds % 86400) // 3600
            minutes = (uptime_seconds % 3600) // 60
            
            if days > 0:
                uptime_str = f"{days}d {hours}h {minutes}m"
            elif hours > 0:
                uptime_str = f"{hours}h {minutes}m"
            else:
                uptime_str = f"{minutes}m"
            
            print(f"CPU Usage: {cpu_usage:.1f}%")
            print(f"Memory: {memory_usage} MB / {memory_max} MB ({(memory_usage/memory_max*100) if memory_max > 0 else 0:.1f}%)")
            print(f"Uptime: {uptime_str}")
        
        print(f"{'='*60}\n")