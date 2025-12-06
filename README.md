# Proxmox Management Tools

Comprehensive Proxmox VE management tools with AWS-style CLI interface and automated CI/CD.

[![Release](https://img.shields.io/github/v/release/yg-codes/proxmox)](https://github.com/yg-codes/proxmox/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yg-codes/proxmox/release.yml)](https://github.com/yg-codes/proxmox/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yg-codes/proxmox?filename=proxmox-admin-cli%2Fgo.mod)](https://github.com/yg-codes/proxmox)

> **⚠️ DEPRECATION NOTICE**: The Python implementation (`python/modular/`) is deprecated as of December 2025. The Go implementation (`pve`) has achieved 100% feature parity with superior performance (5-10x faster). See [PYTHON_DEPRECATION_ANALYSIS.md](PYTHON_DEPRECATION_ANALYSIS.md) for details. All users should migrate to the Go CLI.

## 📁 Directory Structure

```
proxmox/
├── proxmox-admin-cli/        # Go implementation (Production Ready)
│   ├── cmd/                  # AWS-style CLI command structure
│   ├── pkg/                  # Core packages (api, vm, snapshot, backup, etc.)
│   ├── Makefile              # Build automation
│   └── README.md             # Go implementation docs
├── python/                   # Python implementations
│   └── modular/              # Modular Python tools
│       ├── snapshot-manager/ # Snapshot management
│       ├── vm-manager/       # VM & backup management
│       └── pve-snapshots-cli.py  # CLI wrapper
├── .github/
│   └── workflows/
│       └── release.yml       # Automated build & release
├── CLAUDE.md                 # Development guidelines
└── README.md                 # This file
```

## 🎯 Quick Start

### Download Pre-built Binaries (Recommended)

**Linux (amd64)**
```bash
# Download latest release
curl -LO https://github.com/yg-codes/proxmox/releases/latest/download/pve-linux-amd64
sudo install -m 755 pve-linux-amd64 /usr/local/bin/pve
pve --version
```

**Windows (amd64)**
```powershell
# Download from: https://github.com/yg-codes/proxmox/releases/latest
# Rename to pve.exe and add to PATH
pve.exe --version
```

### Configuration
```bash
# Set environment variables
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value

# Test connection
pve cluster task list
```

## 🚀 Recommended Usage

### For Production: Go Implementation (`pve`) ✅ RECOMMENDED
**Binary name**: `pve` (AWS CLI-style naming)

**Key Features**:
- ⚡ **5-10x faster** than Python with goroutine-based concurrency
- 📦 **Single binary** with no runtime dependencies
- 🏗️ **AWS-style hierarchy**: `pve cluster|node|vm|container`
- 🔄 **Automated releases** via GitHub Actions
- 💪 **Production ready** with comprehensive error handling

> **Note**: This is the only actively maintained implementation. The Python version is deprecated.

**Command Structure**:
```bash
pve cluster task list                    # Cluster operations
pve node status --node pve1              # Node operations
pve vm snapshot create --vmid 100        # VM operations
pve vm bulk start                        # Bulk operations
pve container list                       # Container operations
```

### ~~For Development: Python Modular Tools~~ ⚠️ DEPRECATED
- ~~**`python/modular/snapshot-manager/`** - Comprehensive snapshot management~~
- ~~**`python/modular/vm-manager/`** - VM lifecycle & backup management~~

**⚠️ DEPRECATED**: Python tools are no longer maintained. Use the Go implementation (`pve`) instead.

## ✨ Features

### Go CLI (`pve`) - Production Ready
Complete Proxmox VE cluster management with AWS-style command interface:

#### Cluster Operations
- **Task Management**: List and monitor cluster tasks
- **Storage Operations**: List backup storage, manage volumes
- **Network Management**: Query network configuration across nodes

#### Node Operations
- **Resource Monitoring**: CPU, memory, disk statistics
- **Service Management**: Control system services
- **Power Management**: Shutdown, reboot operations

#### VM Operations
- **Lifecycle**: Start, stop, shutdown, restart VMs
- **Snapshots**: Create, list, rollback, delete snapshots
- **Backups**: Create, list backups with storage selection
- **Bulk Operations**:
  - `pve vm bulk start` - Start all stopped VMs concurrently
  - `pve vm bulk stop` - Stop all running VMs concurrently
  - `pve vm bulk backup` - Backup all VMs concurrently

#### Container Operations
- **List & Manage**: Container lifecycle operations

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

## 🛠️ Building from Source

### Go Implementation
```bash
cd proxmox-admin-cli/

# Build for current platform
make build

# Build for all platforms (Linux, Windows, macOS)
make build-all

# Run tests
make test

# Create release archives
make release
```

### CI/CD Pipeline
Automated builds and releases via GitHub Actions:

- **Trigger**: Push any tag matching `v*` pattern
- **Platforms**: Linux amd64, Windows amd64
- **Output**: Pre-built binaries with SHA256 checksums
- **Release**: Auto-created GitHub release with documentation

**Create a new release:**
```bash
git tag -a v1.0.1 -m "Release v1.0.1 - Bug fixes"
git push origin v1.0.1
# GitHub Actions automatically builds and publishes
```

## 🚀 Installation

### Go CLI Installation

**Option 1: Download Pre-built Binaries (Recommended)**

See [Quick Start](#-quick-start) section above.

**Option 2: Build from Source**

See [Building from Source](#️-building-from-source) section above.

### Python Tools Installation

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

### Go CLI (`pve`)

#### Cluster Operations
```bash
# List cluster tasks
pve cluster task list

# List backup storage
pve cluster storage list-backup

# List network configuration
pve cluster network list --node pve1
```

#### Node Operations
```bash
# List all nodes
pve node list

# Show node status
pve node status --node pve1

# Show resource statistics
pve node resource stats --node pve1
```

#### VM Operations
```bash
# List all VMs
pve vm list

# VM lifecycle
pve vm start --vmid 100
pve vm stop --vmid 100
pve vm shutdown --vmid 100

# Snapshots
pve vm snapshot create --vmid 100 --prefix backup
pve vm snapshot list --vmid 100
pve vm snapshot rollback --vmid 100 --snapshot backup-20250117
pve vm snapshot delete --vmid 100 --snapshot backup-20250117

# Backups
pve vm backup create --vmid 100 --storage local
pve vm backup list --vmid 100

# Bulk operations (concurrent processing)
pve vm bulk start              # Start all stopped VMs
pve vm bulk stop               # Stop all running VMs
pve vm bulk backup --storage local  # Backup all VMs
```

#### Container Operations
```bash
# List containers
pve container list
```

### Python VM Manager

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

### From Python to Go (Recommended for Production)

**Performance Benefits**:
- 5-10x faster execution
- Lower memory footprint
- Single binary deployment
- Better concurrent operations

**Command Migration**:
```bash
# Python (old)
python3 main.py create --vmid 100 --prefix backup

# Go (new - AWS-style)
pve vm snapshot create --vmid 100 --prefix backup
```

**Breaking Changes (v1.0.0+)**:
- Binary renamed: `proxmox-admin-cli` → `pve`
- Command structure: AWS-style hierarchy
  - `quick-start-all` → `pve vm bulk start`
  - `quick-stop-all` → `pve vm bulk stop`
  - `quick-backup-all` → `pve vm bulk backup`

### From Legacy to Modular (Python)
1. **Snapshot Operations**: `legacy/pve_snapshots/` → `python/modular/snapshot-manager/`
2. **VM Management**: `legacy/proxmox-vm-manager/` → `python/modular/vm-manager/`
3. **CLI Access**: Use `./pve-snapshots-cli.py` for snapshot management

### Benefits Summary

**Go Implementation**:
- ⚡ 5-10x performance improvement
- 📦 Zero runtime dependencies
- 🔄 Automated CI/CD releases
- 💪 Type safety and compile-time checks

**Python Modular**:
- 🧩 Clean architecture with separated concerns
- 🔧 Better maintainability and extensibility
- ✅ Production ready with proven stability
- 🎯 Consistent interfaces across all tools