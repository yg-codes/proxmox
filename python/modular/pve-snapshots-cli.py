#!/usr/bin/env python3
"""
Simple CLI wrapper for Proxmox snapshot management.

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

def main():
    """Provide helpful guidance and fallback execution."""
    
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