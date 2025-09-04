#!/usr/bin/env python3

"""
Proxmox VM Management Script (Modular Version)
Main entry point for the modular VM management system

Usage: 
  python3 main.py                           # Interactive mode
  python3 main.py start --vmid 7303
  python3 main.py stop --vmid 7303
  python3 main.py backup --vmid 7303 --storage local-zfs --mode snapshot
  python3 main.py list-backups --vmid 7303

Provides comprehensive VM management capabilities:
- Start/Stop VMs with safety checks
- Create VM backups with storage selection
- Restore VMs from backups
- List and manage backups
- Bulk operations with progress tracking
- Real-time task monitoring
- Multi-node cluster support
"""

import sys
import argparse
from vm_manager import ProxmoxVMManager
from snapshot_integration import SnapshotIntegration


def parse_arguments():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description='Proxmox VM Management Tool (Modular)',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Interactive mode
  python3 main.py
  
  # Start/Stop VMs
  python3 main.py start --vmid 7303
  python3 main.py start --vmname smtp01 workstation03
  python3 main.py stop --vmid 7303 7304 --batch -y
  python3 main.py shutdown --vmname smtp01 --batch
  
  # Create backups
  python3 main.py backup --vmid 7303 --storage local-zfs --mode snapshot
  python3 main.py backup --vmname smtp01 --storage backup-storage --mode suspend --compress gzip
  
  # Bulk backups
  python3 main.py backup --vmid 7303 7304 --storage local-zfs --batch -y
  python3 main.py backup --vmname smtp01 workstation03 --storage backup-storage --batch
  
  # List and restore backups
  python3 main.py list-backups --vmid 7303
  python3 main.py list-backups --vmname smtp01
  python3 main.py restore --vmid 7303 --backup-file vzdump-qemu-7303-2025_01_01-12_00_00.vma.zst --batch -y
  
  # Delete backups (use full volid format - get from 'list-backups' command)
  python3 main.py delete-backups --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_01_01.vma.zst" --batch -y
  python3 main.py delete-backups --vmid 7303 --backup-file "backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z" --batch -y
  python3 main.py delete-backups --vmname smtp01 --pattern "*2024*" --batch
  python3 main.py delete-backups --vmid 7303 --cleanup --keep-count 3 --max-age-days 60 --batch
  
  # Storage management
  python3 main.py list-storages --type backup
  python3 main.py list-storages --type vm
        """
    )
    
    # Add global options
    parser.add_argument('--batch', action='store_true', 
                       help='Batch mode - never prompt for input, fail if interaction required')
    parser.add_argument('--yes', '-y', action='store_true', 
                       help='Auto-confirm operations (use with --batch for full automation)')
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Start command
    start_parser = subparsers.add_parser('start', help='Start VM(s)')
    start_parser.add_argument('--vmid', nargs='+', help='VM IDs to start')
    start_parser.add_argument('--vmname', nargs='+', help='VM names to start')
    start_parser.add_argument('--batch', action='store_true', help='Batch mode')
    start_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # Stop command
    stop_parser = subparsers.add_parser('stop', help='Stop VM(s) forcefully')
    stop_parser.add_argument('--vmid', nargs='+', help='VM IDs to stop')
    stop_parser.add_argument('--vmname', nargs='+', help='VM names to stop')
    stop_parser.add_argument('--batch', action='store_true', help='Batch mode')
    stop_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # Shutdown command
    shutdown_parser = subparsers.add_parser('shutdown', help='Shutdown VM(s) gracefully')
    shutdown_parser.add_argument('--vmid', nargs='+', help='VM IDs to shutdown')
    shutdown_parser.add_argument('--vmname', nargs='+', help='VM names to shutdown')
    shutdown_parser.add_argument('--batch', action='store_true', help='Batch mode')
    shutdown_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # Backup command
    backup_parser = subparsers.add_parser('backup', help='Create VM backup(s)')
    backup_parser.add_argument('--vmid', nargs='+', help='VM IDs to backup')
    backup_parser.add_argument('--vmname', nargs='+', help='VM names to backup')
    backup_parser.add_argument('--storage', required=True, help='Storage to save backup to')
    backup_parser.add_argument('--mode', choices=['snapshot', 'suspend', 'stop'], default='snapshot',
                              help='Backup mode (default: snapshot)')
    backup_parser.add_argument('--compress', choices=['zstd', 'gzip', 'lzo'], default='zstd',
                              help='Compression algorithm (default: zstd)')
    backup_parser.add_argument('--batch', action='store_true', help='Batch mode')
    backup_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # List backups command
    list_backups_parser = subparsers.add_parser('list-backups', help='List VM backups')
    list_backups_parser.add_argument('--vmid', nargs='+', help='VM IDs to list backups for')
    list_backups_parser.add_argument('--vmname', nargs='+', help='VM names to list backups for')
    list_backups_parser.add_argument('--storage', help='Specific storage to search in')
    list_backups_parser.add_argument('--all', action='store_true', help='List all backups across all storages')
    list_backups_parser.add_argument('--batch', action='store_true', help='Batch mode')
    
    # Restore command
    restore_parser = subparsers.add_parser('restore', help='Restore VM from backup')
    restore_parser.add_argument('--vmid', required=True, help='VM ID to restore to')
    restore_parser.add_argument('--backup-file', required=True, help='Backup file (volid) to restore from')
    restore_parser.add_argument('--node', help='Node where backup is located (auto-detected if not specified)')
    restore_parser.add_argument('--storage', help='Storage for restored VM disks (optional)')
    restore_parser.add_argument('--batch', action='store_true', help='Batch mode')
    restore_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # List storages command
    list_storages_parser = subparsers.add_parser('list-storages', help='List available storages')
    list_storages_parser.add_argument('--type', choices=['backup', 'vm'], default='backup',
                                     help='Type of storage to list (default: backup)')
    list_storages_parser.add_argument('--batch', action='store_true', help='Batch mode')
    
    # Delete backups command
    delete_backups_parser = subparsers.add_parser('delete-backups', help='Delete VM backup(s)')
    delete_backups_parser.add_argument('--vmid', help='VM ID to delete backups for')
    delete_backups_parser.add_argument('--vmname', help='VM name to delete backups for')
    delete_backups_parser.add_argument('--backup-file', help='Specific backup file (full volid) to delete - use list-backups to see volids')
    delete_backups_parser.add_argument('--pattern', help='Pattern to match backup files (e.g., "*2024*" or "all")')
    delete_backups_parser.add_argument('--storage', help='Specific storage to search in')
    delete_backups_parser.add_argument('--keep-count', type=int, default=5, help='Keep most recent N backups (for cleanup mode)')
    delete_backups_parser.add_argument('--max-age-days', type=int, default=30, help='Delete backups older than N days (for cleanup mode)')
    delete_backups_parser.add_argument('--cleanup', action='store_true', help='Cleanup mode: delete old backups based on keep-count and max-age')
    delete_backups_parser.add_argument('--batch', action='store_true', help='Batch mode')
    delete_backups_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # List VMs command
    list_vms_parser = subparsers.add_parser('list-vms', help='List all VMs')
    list_vms_parser.add_argument('--batch', action='store_true', help='Batch mode')
    
    # VM details command
    details_parser = subparsers.add_parser('details', help='Show VM details')
    details_parser.add_argument('--vmid', help='VM ID to show details for')
    details_parser.add_argument('--vmname', help='VM name to show details for')
    details_parser.add_argument('--batch', action='store_true', help='Batch mode')
    
    return parser.parse_args()


def resolve_vm_identifiers(manager, vmids=None, vmnames=None):
    """Resolve VM identifiers to VM IDs."""
    all_vms = manager.get_all_vms_info()
    resolved_ids = []
    
    if vmids:
        for vmid in vmids:
            # Handle comma-separated values
            for id_str in str(vmid).split(','):
                id_str = id_str.strip()
                if id_str:
                    vm_id = manager.vm_selector.find_vm_by_name_or_id(id_str, all_vms)
                    if vm_id and vm_id not in resolved_ids:
                        resolved_ids.append(vm_id)
                    elif not vm_id:
                        print(f"⚠️ VM '{id_str}' not found")
                        return None
    
    if vmnames:
        for vmname in vmnames:
            # Handle comma-separated values
            for name_str in str(vmname).split(','):
                name_str = name_str.strip()
                if name_str:
                    vm_id = manager.vm_selector.find_vm_by_name_or_id(name_str, all_vms)
                    if vm_id and vm_id not in resolved_ids:
                        resolved_ids.append(vm_id)
                    elif not vm_id:
                        print(f"⚠️ VM '{name_str}' not found")
                        return None
    
    return resolved_ids


def cmd_start_vms(manager, args):
    """Handle start command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    print(f"🚀 Starting {len(vm_ids)} VM(s)")
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            return False
        confirm = input(f"Start {len(vm_ids)} VMs? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    if len(vm_ids) == 1:
        success = manager.start_vm(vm_ids[0])
        return success
    else:
        # Bulk operation
        result = manager.bulk_start_vms(vm_ids)
        return len([r for r in result.results if r.success]) == len(vm_ids)


def cmd_stop_vms(manager, args):
    """Handle stop command.""" 
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    print(f"🛑 Stopping {len(vm_ids)} VM(s) forcefully")
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            return False
        confirm = input(f"⚠️  Forcefully stop {len(vm_ids)} VMs? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    if len(vm_ids) == 1:
        success = manager.stop_vm(vm_ids[0])
        return success
    else:
        # Bulk operation
        result = manager.bulk_stop_vms(vm_ids)
        return len([r for r in result.results if r.success]) == len(vm_ids)


def cmd_shutdown_vms(manager, args):
    """Handle shutdown command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    print(f"🔄 Gracefully shutting down {len(vm_ids)} VM(s)")
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            return False
        confirm = input(f"Gracefully shutdown {len(vm_ids)} VMs? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    if len(vm_ids) == 1:
        success = manager.shutdown_vm(vm_ids[0])
        return success
    else:
        # Bulk operation
        result = manager.bulk_shutdown_vms(vm_ids)
        return len([r for r in result.results if r.success]) == len(vm_ids)


def cmd_create_backups(manager, args):
    """Handle backup command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    print(f"💾 Creating backups for {len(vm_ids)} VM(s)")
    print(f"Storage: {args.storage}")
    print(f"Mode: {args.mode}")
    print(f"Compression: {args.compress}")
    
    # Validate storage space if possible
    if not manager.validate_storage_space(args.storage, 10.0):
        if args.batch:
            print("❌ BATCH MODE: Storage validation failed")
            return False
        proceed = input("⚠️  Storage space warning. Continue anyway? (y/N): ").strip().lower()
        if proceed not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            return False
        confirm = input(f"Create backups for {len(vm_ids)} VMs? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    if len(vm_ids) == 1:
        success = manager.create_backup(vm_ids[0], args.storage, args.mode, args.compress)
        return success
    else:
        # Bulk operation
        result = manager.bulk_create_backups(vm_ids, args.storage, args.mode, args.compress)
        return len([r for r in result.results if r.success]) == len(vm_ids)


def cmd_list_backups(manager, args):
    """Handle list-backups command."""
    if args.all:
        # List all backups across all storages
        print("🔍 Listing all backups across all storages...")
        storages = manager.storage_ops.get_backup_storages()
        
        all_backups = []
        for storage in storages:
            storage_backups = manager.backup_ops.list_all_backups_in_storage(storage['storage'])
            all_backups.extend(storage_backups)
        
        # Sort by creation time (newest first)
        all_backups.sort(key=lambda x: x.get('ctime', 0), reverse=True)
        manager.display_backup_list(all_backups)
        return True
    
    # List backups for specific VMs
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        print("❌ No VMs specified. Use --vmid, --vmname, or --all")
        return False
    
    for vm_id in vm_ids:
        print(f"\n📋 Backups for VM {vm_id}")
        print("=" * 60)
        backups = manager.list_backups_for_vm(vm_id, args.storage)
        manager.display_backup_list(backups)
        
        if len(vm_ids) > 1:
            print("\n" + "="*60 + "\n")
    
    return True


def cmd_restore_backup(manager, args):
    """Handle restore command."""
    vmid = args.vmid
    backup_file = args.backup_file
    
    print(f"🔄 Restoring VM {vmid} from backup")
    print(f"Backup file: {backup_file}")
    
    # Check and handle VM protection
    if not manager.check_and_handle_protection(vmid):
        return False
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            return False
        print(f"\n⚠️  RESTORE WARNING")
        print("=" * 40)
        print("This will:")
        print(f"  • Replace current VM {vmid} with backup data")
        print("  • Permanently lose all changes made after backup")
        print("  • This action cannot be undone!")
        print("=" * 40)
        
        confirm = input("Type 'RESTORE' to confirm this operation: ").strip()
        if confirm != 'RESTORE':
            print("Restore operation cancelled")
            return False
    
    # Find node if not specified
    node = args.node
    if not node:
        # Try to find the backup file in storages
        storages = manager.storage_ops.get_backup_storages()
        for storage in storages:
            backups = manager.backup_ops.list_all_backups_in_storage(storage['storage'])
            for backup in backups:
                if backup.get('volid') == backup_file:
                    node = backup.get('node')
                    break
            if node:
                break
        
        if not node:
            print(f"❌ Could not find backup file '{backup_file}' or determine node")
            return False
    
    # Perform restore
    success = manager.restore_backup(vmid, backup_file, node, args.storage)
    return success


def cmd_list_storages(manager, args):
    """Handle list-storages command."""
    if args.type == 'backup':
        storages = manager.display_backup_storage_list()
    else:
        storages = manager.display_vm_storage_list()
    
    return len(storages) > 0


def cmd_list_vms(manager, args):
    """Handle list-vms command."""
    manager.display_vm_list_interactive()
    return True


def cmd_delete_backups(manager, args):
    """Handle delete-backups command."""
    # Resolve VM identifier if provided
    vm_id = None
    if args.vmid or args.vmname:
        vm_identifier = args.vmid or args.vmname
        all_vms = manager.get_all_vms_info()
        vm_id = manager.vm_selector.find_vm_by_name_or_id(vm_identifier, all_vms)
        if not vm_id:
            print(f"❌ VM '{vm_identifier}' not found")
            return False
    
    # Mode 1: Delete specific backup file
    if args.backup_file:
        if not vm_id:
            print("❌ --vmid or --vmname must be specified when deleting a specific backup file")
            return False
        
        print(f"🗑️  Deleting specific backup for VM {vm_id}")
        print(f"Backup file: {args.backup_file}")
        
        # Find the backup file to get node and storage info
        backups = manager.list_backups_for_vm(vm_id, args.storage)
        target_backup = None
        for backup in backups:
            if backup.get('volid') == args.backup_file:
                target_backup = backup
                break
        
        if not target_backup:
            print(f"❌ Backup file '{args.backup_file}' not found for VM {vm_id}")
            return False
        
        # Handle batch mode confirmation
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                return False
            confirm = input(f"Delete backup '{args.backup_file}'? (y/N): ").strip().lower()
            if confirm not in ['y', 'yes']:
                print("Deletion cancelled")
                return False
        
        success = manager.backup_ops.delete_backup(
            target_backup['volid'],
            target_backup['node'], 
            target_backup['storage']
        )
        return success
    
    # Mode 2: Cleanup old backups
    elif args.cleanup:
        if not vm_id:
            print("❌ --vmid or --vmname must be specified for cleanup mode")
            return False
        
        print(f"🧙 Cleanup mode for VM {vm_id}")
        print(f"Keep count: {args.keep_count}")
        print(f"Max age: {args.max_age_days} days")
        
        # Handle batch mode confirmation
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                return False
        
        success = manager.backup_ops.delete_old_backups(vm_id, args.keep_count, args.max_age_days)
        return success
    
    # Mode 3: Delete by pattern
    elif args.pattern:
        if not vm_id:
            print("❌ --vmid or --vmname must be specified when using pattern deletion")
            return False
        
        print(f"🗑️  Pattern deletion for VM {vm_id}")
        print(f"Pattern: {args.pattern}")
        
        # Handle batch mode confirmation
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                return False
        
        success = manager.backup_ops.delete_backups_by_pattern(vm_id, args.pattern)
        return success
    
    # Mode 4: No specific operation - show help
    else:
        print("❌ No deletion operation specified. Use one of:")
        print("  --backup-file VOLID    Delete specific backup file (use full volid format)")
        print("  --pattern PATTERN      Delete backups matching pattern")
        print("  --cleanup              Delete old backups (with --keep-count and --max-age-days)")
        print("\nExamples:")
        print("  # File-based backup (get volid from 'list-backups'):")
        print("  python3 main.py delete-backups --vmid 7303 --backup-file 'local:backup/vzdump-qemu-7303-2025_01_01.vma.zst'")
        print("  # PBS backup:")
        print("  python3 main.py delete-backups --vmid 7303 --backup-file 'backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z'")
        print("  # Pattern deletion:")
        print("  python3 main.py delete-backups --vmid 7303 --pattern '*2024*'")
        print("  # Cleanup old backups:")
        print("  python3 main.py delete-backups --vmid 7303 --cleanup --keep-count 3 --max-age-days 60")
        print("\nNote: Use 'list-backups --vmid <ID>' to see available backup files and their volids.")
        return False


def cmd_vm_details(manager, args):
    """Handle details command."""
    vm_identifier = args.vmid or args.vmname
    if not vm_identifier:
        print("❌ Either --vmid or --vmname must be specified")
        return False
    
    all_vms = manager.get_all_vms_info()
    vm_id = manager.vm_selector.find_vm_by_name_or_id(vm_identifier, all_vms)
    
    if vm_id:
        manager.show_vm_details(vm_id)
        return True
    else:
        print(f"❌ VM '{vm_identifier}' not found")
        return False


def main():
    """Main entry point for the Proxmox VM Manager."""
    args = parse_arguments()
    manager = ProxmoxVMManager()
    
    # Determine if batch mode is requested
    batch_mode = getattr(args, 'batch', False)
    
    # In batch mode, check for environment variables first and fail fast
    if batch_mode:
        import os
        required_vars = ['PVE_HOST', 'PVE_USER', 'PVE_TOKEN_NAME', 'PVE_TOKEN_VALUE']
        missing_vars = [var for var in required_vars if not os.getenv(var)]
        if missing_vars:
            print("❌ BATCH MODE: Missing required environment variables:")
            for var in missing_vars:
                print(f"❌   {var}")
            print("❌ BATCH MODE: Cannot proceed without credentials")
            sys.exit(1)
    
    try:
        # Connect to Proxmox API
        if not manager.connect_to_proxmox(batch_mode):
            if batch_mode:
                print("❌ BATCH MODE: Failed to connect to Proxmox API")
                sys.exit(1)
            else:
                print("Failed to connect to Proxmox API")
                sys.exit(1)
        
        # Initialize snapshot integration
        snapshot_integration = SnapshotIntegration(manager.api, manager)
        manager.snapshot_integration = snapshot_integration
        
        # Handle commands
        success = False
        if args.command == 'start':
            success = cmd_start_vms(manager, args)
        elif args.command == 'stop':
            success = cmd_stop_vms(manager, args)
        elif args.command == 'shutdown':
            success = cmd_shutdown_vms(manager, args)
        elif args.command == 'backup':
            success = cmd_create_backups(manager, args)
        elif args.command == 'list-backups':
            success = cmd_list_backups(manager, args)
        elif args.command == 'restore':
            success = cmd_restore_backup(manager, args)
        elif args.command == 'list-storages':
            success = cmd_list_storages(manager, args)
        elif args.command == 'list-vms':
            success = cmd_list_vms(manager, args)
        elif args.command == 'delete-backups':
            success = cmd_delete_backups(manager, args)
        elif args.command == 'details':
            success = cmd_vm_details(manager, args)
        else:
            # No command specified
            if batch_mode:
                print("❌ BATCH MODE: No command specified. Interactive mode not available in batch mode.")
                print("Available commands: start, stop, shutdown, backup, list-backups, restore, delete-backups, list-storages, list-vms, details")
                sys.exit(1)
            else:
                # Add snapshot menu to main menu if available
                if snapshot_integration.is_available():
                    # Extend main menu with snapshot option
                    original_menu = manager.main_menu
                    def extended_menu():
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
                                print("9. 📸 Snapshot Management")  # New option
                                print("10. 🔧 Bulk Operations")
                                print("11. 🗄️  Storage Management")
                                print("12. ❓ Help")
                                print("13. 🚪 Exit")
                                print()
                                
                                choice = input("Select option (1-13): ").strip()
                                
                                if choice == '1':
                                    manager.display_vm_list_interactive()
                                elif choice == '2':
                                    manager.handle_vm_details()
                                elif choice == '3':
                                    manager.handle_start_vm()
                                elif choice == '4':
                                    manager.handle_stop_vm()
                                elif choice == '5':
                                    manager.handle_shutdown_vm()
                                elif choice == '6':
                                    manager.handle_create_backup()
                                elif choice == '7':
                                    manager.handle_restore_backup()
                                elif choice == '8':
                                    manager.handle_list_backups()
                                elif choice == '9':
                                    snapshot_integration.display_snapshot_menu()
                                elif choice == '10':
                                    manager.bulk_operations_menu()
                                elif choice == '11':
                                    manager.storage_management_menu()
                                elif choice == '12':
                                    manager.display_usage()
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
                    
                    extended_menu()
                else:
                    # Start original interactive mode
                    manager.main_menu()
                return
        
        # Exit with appropriate code for command line mode
        sys.exit(0 if success else 1)
        
    except KeyboardInterrupt:
        print("\nGoodbye!")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()