#!/usr/bin/env python3

"""
Bulk Operations Module
Manages bulk operations with progress tracking and concurrent execution
"""

import threading
import time
from datetime import datetime
from typing import List, Dict, Tuple
from concurrent.futures import ThreadPoolExecutor, as_completed


class BulkOperationResult:
    """Result of a bulk operation on a single VM."""
    def __init__(self, vmid: str, operation: str, success: bool, message: str = "", duration: float = 0):
        self.vmid = vmid
        self.operation = operation
        self.success = success
        self.message = message
        self.duration = duration
        self.timestamp = datetime.now()


class BulkOperationManager:
    """Manages bulk operations with progress tracking and concurrent execution."""
    
    def __init__(self, max_workers: int = 3):
        self.max_workers = max_workers
        self.results: List[BulkOperationResult] = []
        self.lock = threading.Lock()
        self.cancelled = False
    
    def add_result(self, result: BulkOperationResult):
        """Thread-safe method to add operation result."""
        with self.lock:
            self.results.append(result)
    
    def get_progress(self) -> Tuple[int, int, int]:
        """Return (completed, successful, failed) counts."""
        with self.lock:
            completed = len(self.results)
            successful = sum(1 for r in self.results if r.success)
            failed = completed - successful
            return completed, successful, failed
    
    def cancel(self):
        """Cancel ongoing operations."""
        self.cancelled = True
    
    def print_progress(self, total: int, operation: str):
        """Print current progress."""
        completed, successful, failed = self.get_progress()
        queued = total - completed
        
        print(f"\r{operation} Progress: {completed}/{total} completed, {successful} successful, {failed} failed, {queued} queued", end="", flush=True)
    
    def print_summary(self, operation: str):
        """Print final operation summary."""
        print(f"\n\n{operation} Summary:")
        print("=" * 60)
        
        completed, successful, failed = self.get_progress()
        print(f"Total VMs: {len(self.results)}")
        print(f"Successful: {successful}")
        print(f"Failed: {failed}")
        print(f"Success Rate: {(successful/len(self.results)*100) if self.results else 0:.1f}%")
        
        if failed > 0:
            print(f"\nFailed Operations:")
            print("-" * 40)
            for result in self.results:
                if not result.success:
                    print(f"  VM {result.vmid}: {result.message}")


class BulkSnapshotOperations:
    """Handles bulk snapshot operations."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
    
    def bulk_create_snapshots(self, vm_ids: List[str], prefix: str, max_workers: int = 2) -> BulkOperationManager:
        """Create snapshots for multiple VMs concurrently."""
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n📸 Creating snapshots for {len(vm_ids)} VMs")
        print(f"Prefix: {prefix}")
        print(f"Max concurrent operations: {max_workers}")
        print("=" * 60)
        
        def snapshot_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM exists
                vm_info = self.vm_manager.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "snapshot", False, "VM not found", time.time() - start_time)
                
                # Create snapshot (silent version for bulk operations)
                success = self.vm_manager._create_snapshot_silent(vmid, prefix)
                message = "Snapshot created successfully" if success else "Failed to create snapshot"
                return BulkOperationResult(vmid, "snapshot", success, message, time.time() - start_time)
                
            except Exception as e:
                return BulkOperationResult(vmid, "snapshot", False, str(e), time.time() - start_time)
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vmid = {executor.submit(snapshot_single_vm, vmid): vmid for vmid in vm_ids}
            
            for future in as_completed(future_to_vmid):
                if operation_manager.cancelled:
                    break
                    
                result = future.result()
                operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(len(vm_ids), "Create Snapshots")
        
        operation_manager.print_summary("Bulk Create Snapshots")
        return operation_manager
    
    def bulk_delete_snapshots(self, snapshots_by_vm: Dict[str, List[str]], max_workers: int = 2) -> BulkOperationManager:
        """Delete multiple snapshots concurrently."""
        total_snapshots = sum(len(snapshots) for snapshots in snapshots_by_vm.values())
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🗑️  Deleting {total_snapshots} snapshots across {len(snapshots_by_vm)} VMs")
        print(f"Max concurrent operations: {max_workers}")
        print("=" * 60)
        
        def delete_vm_snapshots(vmid_and_snapshots) -> List[BulkOperationResult]:
            vmid, snapshot_names = vmid_and_snapshots
            results = []
            
            for snapshot_name in snapshot_names:
                start_time = time.time()
                try:
                    # Delete snapshot (silent version for bulk operations)
                    success = self.vm_manager._delete_snapshot_silent(vmid, snapshot_name)
                    message = f"Deleted {snapshot_name}" if success else f"Failed to delete {snapshot_name}"
                    result = BulkOperationResult(f"{vmid}:{snapshot_name}", "delete_snapshot", success, message, time.time() - start_time)
                    results.append(result)
                    
                except Exception as e:
                    result = BulkOperationResult(f"{vmid}:{snapshot_name}", "delete_snapshot", False, str(e), time.time() - start_time)
                    results.append(result)
            
            return results
        
        # Execute operations concurrently
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            future_to_vm = {executor.submit(delete_vm_snapshots, item): item for item in snapshots_by_vm.items()}
            
            for future in as_completed(future_to_vm):
                if operation_manager.cancelled:
                    break
                    
                vm_results = future.result()
                for result in vm_results:
                    operation_manager.add_result(result)
                
                # Print progress
                operation_manager.print_progress(total_snapshots, "Delete Snapshots")
        
        operation_manager.print_summary("Bulk Delete Snapshots")
        return operation_manager


class BulkVMOperations:
    """Handles bulk VM operations."""
    
    def __init__(self, api_client, vm_manager):
        self.api = api_client
        self.vm_manager = vm_manager
    
    def bulk_start_vms(self, vm_ids: List[str], max_workers: int = 3) -> BulkOperationManager:
        """Start multiple VMs concurrently."""
        operation_manager = BulkOperationManager(max_workers)
        
        print(f"\n🚀 Starting {len(vm_ids)} VMs (max {max_workers} concurrent)")
        print("=" * 60)
        
        def start_single_vm(vmid: str) -> BulkOperationResult:
            start_time = time.time()
            try:
                # Check if VM is already running
                vm_info = self.vm_manager.get_vm_info(vmid)
                if not vm_info:
                    return BulkOperationResult(vmid, "start", False, "VM not found", time.time() - start_time)
                
                if vm_info.get('running', False):
                    return BulkOperationResult(vmid, "start", True, "Already running", time.time() - start_time)
                
                # Start the VM (silent version for bulk operations)
                success = self.vm_manager._start_vm_silent(vmid)
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