#!/usr/bin/env python3
"""
Simple CLI wrapper for Proxmox snapshot management.

⚠️  DEPRECATED: This Python implementation is deprecated.
   Please migrate to the Go implementation (proxmox-admin-cli).

   Migration: Replace 'pve-snapshot-manager' with 'pve snapshot'
   Example: 'pve snapshot create --vmid 7303 --prefix backup'

   See: https://github.com/yg-codes/proxmox for the Go CLI.

This wrapper helps users choose between:
1. Global installation via pipx (recommended)
2. Local project usage via uv

Usage:
  pipx install ./snapshot-manager/  # Install globally
  pve-snapshot-manager --help       # Use globally

  OR

  ./pve-snapshots-cli.py --help     # Use locally with uv
"""

import sys
import os
import subprocess
import shutil

def print_deprecation_warning():
    """Print deprecation warning."""
    print("\n" + "=" * 70)
    print("⚠️  DEPRECATION WARNING")
    print("=" * 70)
    print("This Python CLI is DEPRECATED and will not receive new features.")
    print("Please migrate to the Go CLI (proxmox-admin-cli) which is 5-10x faster.")
    print()
    print("Migration examples:")
    print("  Old: pve-snapshot-manager create --vmid 7303 --prefix backup")
    print("  New: pve snapshot create --vmid 7303 --prefix backup")
    print()
    print("See: https://github.com/yg-codes/proxmox")
    print("=" * 70 + "\n")

def main():
    """Provide helpful guidance and fallback execution."""
    # Print deprecation warning
    print_deprecation_warning()

    # Check if pve-snapshot-manager is globally available
    if shutil.which('pve-snapshot-manager'):
        print("✅ pve-snapshot-manager is globally installed!")
        print("Run: pve-snapshot-manager --help")
        print("For global usage, use 'pve-snapshot-manager' instead of this wrapper.")
        return 0

    # If not globally available, show installation instructions
    print("📦 Proxmox Snapshot Management Setup")
    print("")
    print("🎯 RECOMMENDED: Install globally with pipx")
    print("  pipx install ./snapshot-manager/")
    print("  pve-snapshot-manager --help")
    print("")
    print("🔧 ALTERNATIVE: Use locally with uv")
    script_dir = os.path.dirname(os.path.abspath(__file__))
    snapshot_manager_dir = os.path.join(script_dir, 'snapshot-manager')
    
    if not os.path.exists(snapshot_manager_dir):
        print(f"❌ Error: {snapshot_manager_dir} not found")
        return 1
    
    # Try uv first, then python
    if shutil.which('uv'):
        print(f"  cd {snapshot_manager_dir}")
        print("  uv run python main.py --help")
        print("")
        print("🚀 Running with uv...")
        
        original_cwd = os.getcwd()
        try:
            os.chdir(snapshot_manager_dir)
            cmd = ['uv', 'run', 'python', 'main.py'] + sys.argv[1:]
            result = subprocess.run(cmd, check=False)
            return result.returncode
        except Exception as e:
            print(f"❌ Error running with uv: {e}", file=sys.stderr)
            return 1
        finally:
            os.chdir(original_cwd)
    else:
        print(f"  cd {snapshot_manager_dir}")
        print("  python main.py --help")
        print("")
        print("⚠️  uv not found, install uv for better dependency management")
        print("   curl -LsSf https://astral.sh/uv/install.sh | sh")
        return 1

if __name__ == '__main__':
    sys.exit(main())