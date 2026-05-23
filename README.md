# Proxmox Management Tools

Comprehensive Proxmox VE management CLI with AWS-style command interface and automated CI/CD.

[![Release](https://img.shields.io/github/v/release/yg-codes/proxmox)](https://github.com/yg-codes/proxmox/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yg-codes/proxmox/release.yml)](https://github.com/yg-codes/proxmox/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yg-codes/proxmox?filename=proxmox-admin-cli%2Fgo.mod)](https://github.com/yg-codes/proxmox)

## Requirements

- **No runtime dependencies** - Single static binary
- **Operating Systems**: Linux (amd64, arm64), macOS (Intel, Apple Silicon), Windows (amd64)
- **Proxmox VE**: Version 6.x or higher (tested on 7.x)
- **API Access**: Token-based authentication or username/password
- **Permissions**: `PVEVMAdmin` role or `Administrator`

## Directory Structure

```
proxmox/
├── proxmox-admin-cli/        # Go CLI implementation
│   ├── cmd/                  # AWS-style CLI command structure
│   ├── pkg/                  # Core packages (api, vm, snapshot, backup, etc.)
│   ├── Makefile              # Build automation
│   └── README.md             # Go implementation docs
├── scripts/                  # Setup and utility scripts
│   ├── create-api-token.sh   # Fast API token provisioning
│   ├── setup-pve-cli-user.sh # Full user + token setup
│   └── pve-ssh-exec.sh       # Multi-node SSH command runner
├── .github/
│   └── workflows/
│       └── release.yml       # Automated build & release
├── CLAUDE.md                 # Development guidelines
├── FUNCTIONAL_SPECIFICATION.md  # Complete feature reference
└── README.md                 # This file
```

## Quick Start

### Download Pre-built Binaries (Recommended)

**Linux (amd64)**
```bash
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

## Configuration

### Environment Variables (Required)

```bash
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
pveum aclmod / -token 'username@pam!token-name' -role PVEVMAdmin
```

Or use the setup script to create a user + token with Administrator role:

```bash
./scripts/create-api-token.sh pve1
```

### Alternative: Password Authentication

```bash
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_PASSWORD=your-password
```

> **Note**: Token authentication is recommended for production use.

## Features

Complete Proxmox VE cluster management with AWS-style command interface:

### Command Structure
```bash
pve cluster task list                    # Cluster operations
pve node status --node pve1              # Node operations
pve vm snapshot create --vmid 100        # VM operations
pve vm bulk start                        # Bulk operations
pve container list                       # Container operations
```

### Cluster Operations
- **Task Management**: List, monitor, stop cluster tasks with filtering
- **Storage Operations**: List backup and VM storages
- **Network Management**: List, create, delete interfaces; SDN zones/vnets; firewall rules

### Node Operations
- **Resource Monitoring**: CPU, memory, disk statistics and RRD history
- **Service Management**: List, start, stop, restart services
- **Power Management**: Shutdown, reboot nodes

### VM Operations
- **Lifecycle**: Start, stop, shutdown VMs
- **Snapshots**: Create, list, rollback, delete (single or multiple)
- **Backups**: Create, list, restore, delete with retention policies
- **Storage-wide backup listing**: `pve vm backup list --all --storage <name>`
- **Protection handling**: Auto-detect and offer to disable VM protection on restore
- **Bulk Operations**:
  - `pve vm bulk start` - Start all stopped VMs concurrently
  - `pve vm bulk stop` - Stop all running VMs concurrently
  - `pve vm bulk backup` - Backup all VMs concurrently

### VM Selection
- **Flexible patterns**: By ID, name, range (`7201-7205`), wildcard (`72*`), or keywords (`running`, `stopped`)
- **Checkbox-style selection**: Toggle VMs interactively with `all`/`none`/`done` commands

### Container Operations
- **Full lifecycle**: Create, start, stop, shutdown, restart, delete, clone
- **Snapshots**: Create, list, rollback, delete container snapshots

## Usage

### Cluster Operations
```bash
pve cluster task list
pve cluster storage list-backup
pve cluster network list --node pve1
```

### Node Operations
```bash
pve node list
pve node status --node pve1
pve node resource stats --node pve1
```

### VM Operations
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
pve vm snapshot delete --vmid 100 --snapshot snap1,snap2,snap3 -y

# Backups
pve vm backup create --vmid 100 --storage local
pve vm backup list --vmid 100
pve vm backup list --all --storage local
pve vm backup restore --vmid 100 --backup-file "local:backup/..." --node pve1

# Bulk operations
pve vm bulk start
pve vm bulk stop
pve vm bulk backup --storage local
```

### Container Operations
```bash
pve container list
pve container start --node pve1 --vmid 200
pve container stop --node pve1 --vmid 200
```

> **What is volid?**: A `volid` (Volume Identifier) is Proxmox's unique identifier for storage objects. Format: `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`. Examples:
> - File-based: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
> - PBS backup: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`

## Building from Source

```bash
cd proxmox-admin-cli/

make build          # Build for current platform
make build-all      # Cross-compile all platforms
make test           # Run tests
make release        # Create release archives
```

### CI/CD Pipeline
Automated builds and releases via GitHub Actions:

- **Trigger**: Push any tag matching `v*` pattern
- **Platforms**: Linux amd64, Windows amd64
- **Output**: Pre-built binaries with SHA256 checksums
- **Retention**: Only the latest release is kept

**Create a new release:**
```bash
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0
```

## Safety Features
- Multi-level confirmations for destructive operations
- Batch mode support with `--yes` flag for automation
- Dry-run mode with `--dry-run` flag
- Real-time progress tracking for all operations
- VM protection detection before restore
- Proper volid format validation and error handling

## Additional Documentation

- [FUNCTIONAL_SPECIFICATION.md](FUNCTIONAL_SPECIFICATION.md) - Complete feature reference (single source of truth)
- [CLAUDE.md](CLAUDE.md) - Development guidelines for contributors
- [proxmox-admin-cli/README.md](proxmox-admin-cli/README.md) - Go implementation details

## License

This project is licensed under the MIT License.

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
