# Proxmox Management Tools

This directory contains comprehensive Proxmox VE management tools organized into two main categories:

## 📁 Directory Structure

```
proxmox/
├── proxmox-admin-cli/        # Go implementation (recommended)
│   ├── cmd/                  # CLI command structure
│   ├── pkg/                  # Core packages
│   └── build/                # Compiled binaries
├── python/                   # Python implementations
│   └── modular/              # Modular Python tools
│       ├── snapshot-manager/ # Snapshot management
│       ├── vm-manager/       # VM management
│       └── pve-snapshots-cli.py  # CLI wrapper
├── CLAUDE.md                 # Development guidelines
└── README.md                 # This file
```

## 🎯 Recommended Usage

### For Production
Use the **Go implementation** in `proxmox-admin-cli/`:

- **Binary name**: `pve` (AWS CLI-style)
- **Features**: Full cluster management with AWS-style command hierarchy
- **Performance**: 5-10x faster than Python with goroutine-based concurrency

### For Python Development
Use the **Python modular implementations** in `python/modular/`:

- **`python/modular/snapshot-manager/`** - Comprehensive snapshot management system
- **`python/modular/vm-manager/`** - Complete VM lifecycle management

## ✨ Features

### VM Manager (`python/modular/vm-manager/`)
- **VM Lifecycle Management**: Start, stop, shutdown VMs with safety checks
- **Complete Backup Management**: Create, list, restore, and delete VM backups with multiple modes
  - Individual backup deletion with progress tracking
  - Pattern-based deletion with wildcard support (e.g., '*2024*')
  - Automated cleanup with retention policies
  - Bulk operations with concurrent processing
- **Storage Management**: List and validate storage options
- **Snapshot Integration**: Full snapshot management capabilities
- **Bulk Operations**: Concurrent operations on multiple VMs
- **CLI & Interactive Modes**: Both command-line and interactive interfaces

### Snapshot Manager (`python/modular/snapshot-manager/`)
- **Complete Snapshot Lifecycle**: Create, list, rollback, and delete snapshots
- **Flexible Naming**: Prefix-based or exact snapshot naming
- **VM State Support**: Optional inclusion of VM RAM state
- **Bulk Operations**: Manage snapshots across multiple VMs
- **CLI & Interactive Modes**: Both command-line and interactive interfaces
- **Production Ready**: Battle-tested modular architecture

## 🚀 Installation

Following the **pipx (global tools) + uv (projects)** principle for clean and safe Python tool management.

### ⚠️ Prerequisites

**Token Permissions Must Be Set FIRST**

```bash
# Grant required permissions to your API token
pveum aclmod / -token 'your-username@pam!your-token-name' -role PVEVMAdmin
```

### 🎯 RECOMMENDED: Global Installation with pipx

Install tools globally for system-wide access:

```bash
# Install pipx if not already installed
python3 -m pip install --user pipx
python3 -m pipx ensurepath

# Install Proxmox tools globally
cd python/modular/
pipx install ./snapshot-manager/
pipx install ./vm-manager/

# Use globally from anywhere
pve-snapshot-manager --help
pve-vm-manager-modular --help
```

### 🔧 ALTERNATIVE: Project Development with uv

For development or project-specific usage:

```bash
# Install uv if not already installed
curl -LsSf https://astral.sh/uv/install.sh | sh

# Navigate to specific project
cd python/modular/snapshot-manager/
uv run python main.py --help

cd ../vm-manager/
uv run python main.py --help
```

### Authentication Setup
```bash
# Set environment variables (required for all methods)
export PVE_HOST=your-proxmox-host.com
export PVE_USER=your-username@pam
export PVE_TOKEN_NAME=your-token-name
export PVE_TOKEN_VALUE=your-token-value
```

## 🎯 Key Features

### Complete Backup Lifecycle Management
The VM Manager now provides comprehensive backup operations with full CRUD capabilities:

- ✅ **Create**: Multiple backup modes (snapshot, suspend, stop) with storage selection
- ✅ **Read**: List and display backups with detailed information and volid format
- ✅ **Update**: Restore VMs from backups with protection handling
- ✅ **Delete**: Comprehensive deletion capabilities:
  - **Specific deletion**: Target individual backups using volid
  - **Pattern matching**: Delete multiple backups using wildcards (e.g., `*2024*`)
  - **Automated cleanup**: Retention policies with keep-count and age limits
  - **Bulk operations**: Concurrent deletion with progress tracking

### Safety Features
- Multi-level confirmations for destructive operations
- Batch mode support with `--yes` flag for automation
- Real-time progress tracking for all operations
- Proper volid format validation and error handling

## 📖 Usage

### VM Manager

#### Global Usage (Recommended)
```bash
# After pipx installation
pve-vm-manager-modular --help

# Interactive mode
pve-vm-manager-modular

# Command line examples
pve-vm-manager-modular start --vmid 7303
pve-vm-manager-modular backup --vmid 7303 --storage local-zfs --mode snapshot
pve-vm-manager-modular list-backups --vmid 7303
pve-vm-manager-modular delete-backups --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --batch --yes
pve-vm-manager-modular restore --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --batch --yes
```

#### Project Development Usage
```bash
cd python/modular/vm-manager/
uv run python main.py --help

# Command line examples with uv
uv run python main.py start --vmid 7303
uv run python main.py backup --vmid 7303 --storage local-zfs --mode snapshot
uv run python main.py list-backups --vmid 7303
```

> **💡 What is volid?**: A `volid` (Volume Identifier) is Proxmox's unique identifier for storage objects. Format: `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`. Examples:
> - File-based: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
> - PBS backup: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`
> 
> **Important**: Always use the full volid format for backup operations. Use `list-backups` to see the correct volid format for your backups.

### Snapshot Manager

#### Global Usage (Recommended)
```bash
# After pipx installation
pve-snapshot-manager --help

# Interactive mode
pve-snapshot-manager

# Command line usage
pve-snapshot-manager create --vmid 7303 --prefix backup
pve-snapshot-manager list --vmid 7303
pve-snapshot-manager rollback --vmid 7303 --snapshot_name backup-20250101
pve-snapshot-manager delete --vmid 7303 --snapshot_name backup-20250101 --yes
```

#### Project Development Usage
```bash
cd python/modular/snapshot-manager/
uv run python main.py --help

# Command line usage with uv
uv run python main.py create --vmid 7303 --prefix backup
uv run python main.py list --vmid 7303
uv run python main.py rollback --vmid 7303 --snapshot_name backup-20250101
uv run python main.py delete --vmid 7303 --snapshot_name backup-20250101 --yes
```

#### Helper Wrapper
```bash
# The helper wrapper provides guidance and fallback
cd python/modular/
./pve-snapshots-cli.py
```

## 🔄 Migration Guide

### From Legacy to Modular
1. **Snapshot Operations**: Replace `legacy/pve_snapshots/` usage with `python/modular/snapshot-manager/`
2. **VM Management**: Replace `legacy/proxmox-vm-manager/` usage with `python/modular/vm-manager/`
3. **CLI Access**: Use `./pve-snapshots-cli.py` for convenient snapshot management

### Benefits of Modular Approach
- **Clean architecture** with separated concerns
- **Better maintainability** and extensibility
- **Production ready** with proven stability
- **Consistent interfaces** across all tools
- **Enhanced error handling** and validation