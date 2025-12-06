# Proxmox Management Tools

Comprehensive Proxmox VE management tools with AWS-style CLI interface and automated CI/CD.

[![Release](https://img.shields.io/github/v/release/yg-codes/proxmox)](https://github.com/yg-codes/proxmox/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yg-codes/proxmox/release.yml)](https://github.com/yg-codes/proxmox/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yg-codes/proxmox?filename=proxmox-admin-cli%2Fgo.mod)](https://github.com/yg-codes/proxmox)

> **⚠️ DEPRECATION NOTICE**: The Python implementation (`python/modular/`) is deprecated as of December 2025. The Go implementation (`pve`) has achieved 100% feature parity with superior performance (5-10x faster). See [PYTHON_DEPRECATION_ANALYSIS.md](PYTHON_DEPRECATION_ANALYSIS.md) for details. All users should migrate to the Go CLI.

## 📋 Requirements

### For Go CLI (Recommended)
- **No runtime dependencies** - Single static binary
- **Operating Systems**:
  - Linux (amd64, arm64)
  - macOS (Intel, Apple Silicon)
  - Windows (amd64)

### For Python Tools (Deprecated)
- **Python**: Version 3.8 or higher
- **Package Managers**: `uv` or `pipx`
- **Dependencies**: `requests`, `urllib3`

### Proxmox Environment
- **Proxmox VE**: Version 6.x or higher (tested on 7.x)
- **API Access**: Token-based authentication or username/password
- **Permissions**: `PVEVMAdmin` role or equivalent
- **Network**: HTTPS access to Proxmox API (default port 8006)

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

## ⚙️ Configuration

### Environment Variables (Required)

Both Go and Python implementations use the same authentication method:

```bash
# Set environment variables
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value

# Test connection
pve cluster task list
```

### API Token Setup

Create an API token in Proxmox Web UI, then grant permissions:

```bash
# Grant required permissions to your API token (run on Proxmox server)
pveum aclmod / -token 'username@pam!token-name' -role PVEVMAdmin
```

### Alternative: Password Authentication

```bash
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_PASSWORD=your-password
```

> **Note**: Token authentication is recommended for production use.

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

## 🚀 Latest Release: v1.1.1

**Released December 2025** - Multi-platform support with Python deprecation

### New Features
- ✅ **Multi-platform builds**: Linux (amd64/arm64), macOS (Intel/ARM), Windows (amd64)
- ✅ **Python deprecation**: Official deprecation of Python implementation
- ✅ **Enhanced documentation**: Complete feature parity analysis
- ✅ **Improved downloads**: Fixed download commands with `curl`

### Quick Install v1.1.1

**Linux:**
```bash
# Linux amd64
curl -LO https://github.com/yg-codes/proxmox/releases/download/v1.1.1/pve-linux-amd64
sudo install -m 755 pve-linux-amd64 /usr/local/bin/pve

# Linux arm64
curl -LO https://github.com/yg-codes/proxmox/releases/download/v1.1.1/pve-linux-arm64
sudo install -m 755 pve-linux-arm64 /usr/local/bin/pve
```

**macOS:**
```bash
# Intel Macs
curl -LO https://github.com/yg-codes/proxmox/releases/download/v1.1.1/pve-darwin-amd64
sudo install -m 755 pve-darwin-amd64 /usr/local/bin/pve

# Apple Silicon
curl -LO https://github.com/yg-codes/proxmox/releases/download/v1.1.1/pve-darwin-arm64
sudo install -m 755 pve-darwin-arm64 /usr/local/bin/pve
```

### Migration from Python

For complete migration instructions and command translation, see:
- [PYTHON_DEPRECATION_ANALYSIS.md](PYTHON_DEPRECATION_ANALYSIS.md) - Detailed analysis
- [python/DEPRECATED.md](python/DEPRECATED.md) - Quick migration guide

**Quick command reference:**
| Python Tool | Go Command |
|-------------|------------|
| `pve-snapshot-manager create` | `pve vm snapshot create` |
| `pve-vm-manager-modular backup` | `pve vm backup create` |
| `pve-vm-manager-modular start` | `pve vm start` |
| See deprecation analysis for full list |

## 📄 License

This project is licensed under the MIT License.

### MIT License

```
MIT License

Copyright (c) 2025 Proxmox Management Tools

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📚 Additional Documentation

- [PYTHON_DEPRECATION_ANALYSIS.md](PYTHON_DEPRECATION_ANALYSIS.md) - Detailed Python vs Go comparison
- [CLAUDE.md](CLAUDE.md) - Development guidelines for contributors
- [python/DEPRECATED.md](python/DEPRECATED.md) - Python migration guide
- [proxmox-admin-cli/README.md](proxmox-admin-cli/README.md) - Go implementation details