#!/usr/bin/env python3

"""
Proxmox VM Snapshot Management Script (Modular Version)
Main entry point for the modular snapshot management system

Usage: 
  python3 main.py                           # Interactive mode
  python3 main.py create --vmid 7303 --prefix backup
  python3 main.py list --vmid 7303
  python3 main.py rollback --vmid 7303 --snapshot_name backup-vm-20250101-1200
  python3 main.py delete --vmid 7303 --snapshot_name backup-vm-20250101-1200

Provides comprehensive VM snapshot management capabilities:
- Create snapshots with intelligent naming
- Rollback to previous snapshots
- List and manage existing snapshots
- Delete snapshots with safety checks
- Bulk snapshot operations
- Real-time task monitoring
"""

import sys
import argparse
from snapshot_manager import ProxmoxSnapshotManager


def parse_arguments():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description='Proxmox VM Snapshot Management Tool',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Interactive mode
  python3 main.py
  
  # Create snapshot for single VM
  python3 main.py create --vmid 7303 --prefix backup
  python3 main.py create --vmname smtp01 --prefix backup --vmstate 1
  
  # Create snapshots for multiple VMs (batch mode)
  python3 main.py create --vmid 7303 7304 --prefix pre-update --batch -y
  python3 main.py create --vmname smtp01 workstation03 --prefix backup --batch
  
  # Create with exact snapshot name
  python3 main.py create --vmid 7303 --snapshot_name backup-20250101-1200
  
  # List snapshots
  python3 main.py list --vmid 7303
  python3 main.py list --vmname smtp01 workstation03
  
  # Rollback to snapshot (batch mode)
  python3 main.py rollback --vmid 7303 --snapshot_name backup-smtp01-20250101-1200 --batch -y
  python3 main.py rollback --vmname smtp01 workstation03 --snapshot_name backup-20250101-1200 --batch
  
  # Delete snapshot (batch mode)
  python3 main.py delete --vmid 7303 --snapshot_name backup-smtp01-20250101-1200 --batch -y
  python3 main.py delete --vmname smtp01 --snapshot_name backup-20250101-1200 backup-20250102-1200 --batch
  python3 main.py delete --vmid 7303 --all --batch -y
        """
    )
    
    # Add global options
    parser.add_argument('--batch', action='store_true', 
                       help='Batch mode - never prompt for input, fail if interaction required')
    parser.add_argument('--yes', '-y', action='store_true', 
                       help='Auto-confirm operations (use with --batch for full automation)')
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Create command
    create_parser = subparsers.add_parser('create', help='Create VM snapshots')
    create_parser.add_argument('--vmid', nargs='+', help='VM IDs to create snapshots for')
    create_parser.add_argument('--vmname', nargs='+', help='VM names to create snapshots for')
    create_parser.add_argument('--prefix', help='Snapshot prefix (combines with VM name and timestamp)')
    create_parser.add_argument('--snapshot_name', help='Exact snapshot name (alternative to --prefix)')
    create_parser.add_argument('--vmstate', choices=['0', '1'], default='0', 
                              help='Include VM state (RAM) - 1=yes, 0=no (default)')
    create_parser.add_argument('--batch', action='store_true', 
                              help='Batch mode - never prompt for input, fail if interaction required')
    create_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List VM snapshots')
    list_parser.add_argument('--vmid', nargs='+', help='VM IDs to list snapshots for')
    list_parser.add_argument('--vmname', nargs='+', help='VM names to list snapshots for')
    list_parser.add_argument('--batch', action='store_true', 
                            help='Batch mode - never prompt for input, fail if interaction required')
    
    # Rollback command
    rollback_parser = subparsers.add_parser('rollback', help='Rollback VM(s) to snapshot')
    rollback_parser.add_argument('--vmid', nargs='+', help='VM IDs to rollback')
    rollback_parser.add_argument('--vmname', nargs='+', help='VM names to rollback')
    rollback_parser.add_argument('--snapshot_name', required=True, help='Snapshot name to rollback to')
    rollback_parser.add_argument('--batch', action='store_true', 
                                help='Batch mode - never prompt for input, fail if interaction required')
    rollback_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
    # Delete command
    delete_parser = subparsers.add_parser('delete', help='Delete VM snapshot(s)')
    delete_parser.add_argument('--vmid', nargs='+', help='VM IDs to delete snapshots from')
    delete_parser.add_argument('--vmname', nargs='+', help='VM names to delete snapshots from')
    delete_parser.add_argument('--snapshot_name', nargs='+', help='Snapshot names to delete')
    delete_parser.add_argument('--all', action='store_true', help='Delete all snapshots')
    delete_parser.add_argument('--batch', action='store_true', 
                              help='Batch mode - never prompt for input, fail if interaction required')
    delete_parser.add_argument('--yes', '-y', action='store_true', help='Skip confirmation')
    
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


def cmd_create_snapshots(manager, args):
    """Handle create command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    # Validate arguments
    if not args.prefix and not args.snapshot_name:
        print("⚠️ Either --prefix or --snapshot_name must be specified")
        return False
    
    if args.prefix and args.snapshot_name:
        print("⚠️ Cannot specify both --prefix and --snapshot_name")
        return False
    
    # Determine if using exact name or prefix
    use_exact_name = bool(args.snapshot_name)
    name_or_prefix = args.snapshot_name if use_exact_name else args.prefix
    
    # Set vmstate option
    include_vmstate = args.vmstate == '1'
    original_vmstate = manager.save_vmstate
    manager.save_vmstate = include_vmstate
    
    try:
        if use_exact_name:
            print(f"📸 Creating snapshots for {len(vm_ids)} VMs with exact name '{name_or_prefix}'")
        else:
            print(f"📸 Creating snapshots for {len(vm_ids)} VMs with prefix '{name_or_prefix}'")
        print(f"VM State: {'WITH vmstate (RAM)' if include_vmstate else 'WITHOUT vmstate (disk only)'}")
        
        # Handle batch mode confirmation
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                print("❌ BATCH MODE: Use --yes to auto-confirm operations")
                return False
            confirm = input(f"Proceed to create snapshots? (y/N): ").strip().lower()
            if confirm not in ['y', 'yes']:
                print("Operation cancelled")
                return False
        
        success_count = 0
        for vm_id in vm_ids:
            success = manager.create_snapshot(vm_id, name_or_prefix, use_exact_name)
            if success:
                success_count += 1
        
        print(f"\nSUMMARY: {success_count}/{len(vm_ids)} snapshots created successfully")
        return success_count == len(vm_ids)
        
    finally:
        manager.save_vmstate = original_vmstate


def cmd_list_snapshots(manager, args):
    """Handle list command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    for vm_id in vm_ids:
        manager.list_snapshots(vm_id)
        if len(vm_ids) > 1:
            print("\n" + "="*60 + "\n")
    
    return True


def cmd_rollback_snapshot(manager, args):
    """Handle rollback command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    print(f"⏪ Rolling back {len(vm_ids)} VM(s) to snapshot '{args.snapshot_name}'")
    
    # Handle batch mode confirmation
    if not args.yes:
        if args.batch:
            print("❌ BATCH MODE: Confirmation required but --yes not provided")
            print("❌ BATCH MODE: Use --yes to auto-confirm operations")
            return False
        confirm = input(f"This will revert all changes after the snapshot. Continue? (y/N): ").strip().lower()
        if confirm not in ['y', 'yes']:
            print("Operation cancelled")
            return False
    
    success_count = 0
    for vm_id in vm_ids:
        success = manager.rollback_snapshot(vm_id, args.snapshot_name)
        if success:
            success_count += 1
    
    print(f"\nSUMMARY: {success_count}/{len(vm_ids)} rollbacks completed successfully")
    return success_count == len(vm_ids)


def cmd_delete_snapshot(manager, args):
    """Handle delete command."""
    vm_ids = resolve_vm_identifiers(manager, args.vmid, args.vmname)
    if not vm_ids:
        return False
    
    if args.all:
        print(f"🗑️ Deleting ALL snapshots from {len(vm_ids)} VM(s)")
        
        # Handle batch mode confirmation for delete all
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                print("❌ BATCH MODE: Use --yes to auto-confirm dangerous operations")
                return False
            confirm = input("This will permanently delete ALL snapshots. Type 'DELETE ALL' to confirm: ").strip()
            if confirm != 'DELETE ALL':
                print("Operation cancelled")
                return False
        
        success_count = 0
        for vm_id in vm_ids:
            snapshots = manager.get_snapshots(vm_id)
            available_snapshots = [s for s in snapshots if s.get('name') != 'current']
            if available_snapshots:
                manager.delete_all_snapshots(vm_id, available_snapshots)
                success_count += 1
        
        print(f"\nSUMMARY: Processed {success_count}/{len(vm_ids)} VMs")
        return True
    
    elif args.snapshot_name:
        print(f"🗑️ Deleting snapshot(s) from {len(vm_ids)} VM(s)")
        
        # Handle batch mode confirmation for delete specific snapshots
        if not args.yes:
            if args.batch:
                print("❌ BATCH MODE: Confirmation required but --yes not provided")
                print("❌ BATCH MODE: Use --yes to auto-confirm operations")
                return False
            confirm = input(f"Delete {len(args.snapshot_name)} snapshot(s)? (y/N): ").strip().lower()
            if confirm not in ['y', 'yes']:
                print("Operation cancelled")
                return False
        
        success_count = 0
        for vm_id in vm_ids:
            for snapshot_name in args.snapshot_name:
                success = manager.delete_snapshot(vm_id, snapshot_name)
                if success:
                    success_count += 1
        
        total_operations = len(vm_ids) * len(args.snapshot_name)
        print(f"\nSUMMARY: {success_count}/{total_operations} deletions completed successfully")
        return success_count == total_operations
    
    else:
        print("⚠️ Either --snapshot_name or --all must be specified")
        return False


def main():
    """Main entry point for the Proxmox Snapshot Manager."""
    args = parse_arguments()
    manager = ProxmoxSnapshotManager()
    
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
        
        # Handle commands
        if args.command == 'create':
            success = cmd_create_snapshots(manager, args)
        elif args.command == 'list':
            success = cmd_list_snapshots(manager, args)
        elif args.command == 'rollback':
            success = cmd_rollback_snapshot(manager, args)
        elif args.command == 'delete':
            success = cmd_delete_snapshot(manager, args)
        else:
            # No command specified
            if batch_mode:
                print("❌ BATCH MODE: No command specified. Interactive mode not available in batch mode.")
                print("Available commands: create, list, rollback, delete")
                sys.exit(1)
            else:
                # Start interactive mode
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