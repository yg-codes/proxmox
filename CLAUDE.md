# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This repository provides comprehensive Proxmox VE management tools in two implementations:

1. **Go Implementation** (Recommended for production): High-performance, single-binary admin CLI
2. **Python Modular** (Stable): Clean architecture with VM and snapshot management

### Requirements

- **Go**: Version 1.21 or higher (for Go implementation)
- **Python**: Version 3.8 or higher (for Python implementation)
- **Package Managers**: `uv` (Python), `pipx` (Python CLI tools)
- **OS**: Linux, macOS, or Windows (WSL supported)

## Directory Structure

```
proxmox/
├── proxmox-admin-cli/              # Go implementation (5-10x faster)
└── python/                         # Python implementations
    └── modular/                    # Modular implementations
        ├── snapshot-manager/       # Modular snapshot management
        ├── vm-manager/             # Modular VM & backup management
        └── pve-snapshots-cli.py    # CLI wrapper
```

## Quick Reference

### One-Liner Quality Checks

**Go (Complete Check)**:
```bash
cd proxmox-admin-cli/ && make fmt && make vet && make test && make build
```

**Python (Complete Check)**:
```bash
black . && flake8 . && mypy .
```

### Common Tasks
```bash
# Build Go binary
cd proxmox-admin-cli/ && make build

# Test Go CLI
./build/pve vm list

# Run Python tool (development)
cd python/modular/snapshot-manager/ && uv run python main.py --help

# Create new release
git tag -a v1.0.1 -m "Release v1.0.1" && git push origin v1.0.1
```

## Development Commands

### Go Implementation (proxmox-admin-cli/)

```bash
# Setup and build
make deps           # Download dependencies
make build          # Build for current platform
make build-all      # Cross-compile for all platforms
make install        # Install to GOPATH/bin

# Development
make run ARGS='--help'
make dev ARGS='create --vmid 100 --prefix backup'
make test           # Run tests
make test-coverage  # Generate coverage report

# Code quality
make fmt            # Format code
make vet            # Run go vet
make lint           # Run golangci-lint (requires golangci-lint)

# Release
make release        # Create release archives for all platforms
make clean          # Remove build artifacts

# Docker
make docker-build
make docker-run ARGS='--help'

# Direct usage (after build) - AWS-STYLE COMMAND STRUCTURE
./build/pve --help

# Cluster commands (task, storage, network)
pve cluster task list
pve cluster storage list-backup
pve cluster network list --node pve1

# Node commands (resource monitoring, services, power)
pve node list
pve node status --node pve1
pve node resource stats --node pve1

# VM commands (snapshot, backup, lifecycle)
pve vm list
pve vm snapshot create --vmid 7303 --prefix backup
pve vm snapshot list --vmid 7303
pve vm backup create --vmid 7303 --storage local
pve vm start --vmid 7303

# VM bulk operations (all VMs at once)
pve vm bulk start              # Start all stopped VMs
pve vm bulk stop               # Stop all running VMs
pve vm bulk backup --storage local  # Backup all VMs

# Container commands (top-level)
pve container list
pve container create --name test-ct
```

### Python Modular Implementation

```bash
# Installation (choose one approach)

# Option 1: Global installation with pipx (recommended)
cd python/modular/
pipx install ./snapshot-manager/
pipx install ./vm-manager/
pve-snapshot-manager --help
pve-vm-manager-modular --help

# Option 2: Project development with uv
cd python/modular/snapshot-manager/
uv run python main.py --help
cd ../vm-manager/
uv run python main.py --help

# Code quality (in project root)
black .             # Format code
flake8 .            # Lint code
mypy .              # Type checking
pytest              # Run tests (if present)
```

## Configuration

### Authentication (Required for all implementations)

Both Python and Go implementations use the **same authentication approach** - environment variables only, no config files needed.

```bash
# Set environment variables (required)
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value

# Alternatively, for password authentication
export PVE_PASSWORD=your-password
```

### API Token Setup (Critical)

```bash
# Create token in Proxmox Web UI first, then run:
pveum aclmod / -token 'username@pam!token-name' -role PVEVMAdmin
```

## Architecture

### Go Implementation (Recommended for Production)

**Benefits:**
- **5-10x faster** than Python with goroutine-based concurrency
- **Single binary** with no runtime dependencies
- **Memory efficient**: ~10-20MB vs ~50-100MB (Python)
- **Type safety**: Compile-time error detection
- **Cross-platform**: Native builds for Linux, macOS, Windows
- **Startup time**: ~0.1s vs ~2-3s (Python)

**Module Structure:**
```
proxmox-admin-cli/
├── cmd/              # CLI entry point
├── pkg/
│   ├── api/         # HTTP client and authentication
│   ├── vm/          # VM operations and selection
│   ├── snapshot/    # Snapshot lifecycle management
│   ├── backup/      # Backup operations
│   ├── storage/     # Storage management
│   ├── bulk/        # Concurrent bulk operations
│   └── config/      # Configuration management
├── Makefile         # Build automation
└── go.mod           # Go module definition
```

**Key Technologies:**
- **cobra**: CLI framework
- **logrus**: Structured logging
- **net/http**: HTTP client (stdlib)
- **Environment variables**: Configuration (matching Python implementation)

### Python Modular Architecture

**Snapshot Manager** (`modular/snapshot-manager/`):
- `main.py` - CLI entry point
- `snapshot_manager.py` - Main orchestrator
- `proxmox_api.py` - API communication
- `vm_operations.py` - VM management
- `vm_selector.py` - Flexible VM selection
- `snapshot_operations.py` - Snapshot CRUD
- `bulk_operations.py` - Concurrent operations

**VM Manager** (`modular/vm-manager/`):
- All snapshot-manager modules (reused)
- `vm_manager.py` - Main orchestrator
- `backup_operations.py` - Complete backup lifecycle (CRUD)
- `storage_operations.py` - Storage discovery
- `snapshot_integration.py` - Bridge to snapshot-manager

**Shared Code Reuse:**
- 60% code reuse through shared modules: `proxmox_api.py`, `vm_operations.py`, `vm_selector.py`, `bulk_operations.py`
- Clean separation of concerns
- Individual module testing capability

## Key Features

### VM Selection Patterns (All Implementations)
- **Range**: `7201-7205` (all VMs in range)
- **List**: `7201,7203,7205` (specific VMs)
- **Wildcard**: `72*` (pattern matching)
- **Keywords**: `running`, `stopped`, `all`
- **Interactive**: Checkbox-style UI
- **Names**: VM name resolution alongside IDs

### Snapshot Operations
- Create with prefix or exact name
- Optional VM state (RAM) inclusion
- List with configuration details
- Rollback with safety checks
- Bulk operations with concurrency limits

### Backup Management (Python VM Manager only)
- **Create**: Multiple modes (snapshot, suspend, stop)
- **List**: Detailed backup information with volid format
- **Restore**: VM restoration with protection handling
- **Delete**:
  - Specific deletion using volid
  - Pattern-based with wildcards (e.g., `*2024*`)
  - Automated cleanup with retention policies
  - Bulk concurrent operations

**Important**: Backup volid format = `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`
- File-based: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
- PBS backup: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`

### Concurrency Limits (Python)
- `MAX_CONCURRENT_START_STOP=3` - VM state changes
- `MAX_CONCURRENT_BACKUPS=2` - Backup operations
- `MAX_CONCURRENT_SNAPSHOTS=2` - Snapshot operations

### Snapshot Naming Constraints
- Maximum prefix length: 25 characters
- Automatic invalid character cleanup
- Intelligent timestamp appending
- `vmstate` keyword detection for RAM inclusion

## Code Style

### Go
- Follow standard Go conventions
- Use `gofmt` for formatting (via `make fmt`)
- Run `go vet` before commits (via `make vet`)
- Use `golangci-lint` for comprehensive linting (via `make lint`)
- Explicit error handling (no exceptions)
- Use context for cancellation and timeouts
- 2-space indentation (standard Go formatting)
- Descriptive variable names (no hardcoded values)

### Python
- Python 3.8+ with type hints (mandatory)
- Black formatter (88-character line limit)
- PEP 8 compliance (enforced via `flake8`)
- Comprehensive docstrings for public methods
- Custom `ProxmoxAPIError` exception
- Explicit error handling patterns
- 4-space indentation (Python standard)
- Descriptive variable names (no hardcoded values)

### Git Conventions (GitHub Repository)

**Commit Messages**:
- Clear, descriptive commit messages
- **No ticket ID required** (this is a GitHub repository, not GitLab)
- **Do NOT include** Claude Code attribution:
  ```
  # NEVER INCLUDE:
  🤖 Generated with [Claude Code](https://claude.ai/code)
  Co-Authored-By: Claude <noreply@anthropic.com>
  ```

**Branching**:
- Main branch: `main`
- Standard GitHub workflow (feature branches, pull requests)
- No special naming conventions required

**.gitignore Rules**:
- Always ignore `claude` and `.trae` directories
- Build artifacts (`build/`, `dist/`)
- Environment files (`.env`)
- Python artifacts (`__pycache__/`, `*.pyc`)
- Go artifacts (`*.test`, `coverage.out`)

## Testing & Safety

### Testing Environment
- **Go**: Use standard `go test`, coverage with `make test-coverage`
- **Python**: Use pytest (if tests present)
- Both require live Proxmox environment
- Test API token permissions before bulk operations
- Verify storage availability before backup operations

### Testing Constraints (IMPORTANT)

**Approved for Testing (No Approval Required)**:
- VMID **7303 only** for VM operations
- Read-only operations on any VM (list, status, info)
- Cluster-level commands (task list, storage list, network list)
- Node-level commands (node list, status, resource stats)

**Requires Explicit Approval**:
- Operations on VMs other than 7303
- Destructive operations (delete, stop, shutdown) on any VM
- Bulk operations affecting multiple VMs
- Backup operations (creates data on storage)
- Snapshot operations on non-7303 VMs

**Testing Best Practices**:
- Always verify API token permissions first
- Check storage availability before backup operations
- Validate VM lock status before operations
- Use `--dry-run` flags when available
- Test in development/test environments when possible

## Performance Benchmarks (Go vs Python)

| Operation | Python (ThreadPool=3) | Go (Goroutines=3) | Improvement |
|-----------|----------------------|-------------------|-------------|
| Create 10 snapshots | 45.2s | 8.7s | 5.2x faster |
| Delete 20 snapshots | 52.1s | 9.3s | 5.6x faster |
| List 50 VMs | 12.4s | 2.1s | 5.9x faster |
| Rollback 5 VMs | 78.9s | 12.4s | 6.4x faster |

*Benchmarks on Proxmox 7.4 cluster, 3 nodes, 100+ VMs*

## Migration Guide

### Python Modular
- Snapshot management: `python/modular/snapshot-manager/`
- VM management: `python/modular/vm-manager/`
- CLI wrapper: `python/modular/pve-snapshots-cli.py`

### Python → Go (AWS-Style Command Structure)
```bash
# Python version (flat structure)
python3 main.py create --vmid 7303 --prefix backup
python3 main.py backup --vmid 7303 --storage local

# Go version (AWS-style hierarchy - binary name: pve)
pve vm snapshot create --vmid 7303 --prefix backup
pve vm backup create --vmid 7303 --storage local

# Note: The Go version uses nested commands similar to AWS CLI:
# - cluster (task, storage, network)
# - node (resource, services, power)
# - vm (snapshot, backup, lifecycle)
# - container (top-level)
```

**Migration Benefits:**
- AWS CLI-style command organization
- 5-10x performance improvement
- Single binary deployment (no Python environment)
- Compile-time error detection
- Superior concurrency handling
- Better command discoverability with logical grouping

## Common Issues

### Go Build Issues
- **"Go version too old"**: Ensure Go 1.21+ is installed (`go version`)
- **"Package not found"**: Run `make deps` to download dependencies
- **Build fails**: Try `make clean && make deps && make build`
- **Cross-compilation errors**: Ensure correct GOOS/GOARCH for target platform

### Python Issues
- **"Module not found"**: Install with `pipx install` or use `uv run`
- **"Python version too old"**: Ensure Python 3.8+ (`python3 --version`)
- **Dependencies missing**: Run `uv sync` in module directory
- **Type check failures**: Run `mypy .` to see specific issues

### Proxmox API Issues
- **"Permission check failed"**: Requires proper token ACL configuration
  ```bash
  pveum aclmod / -token 'username@pam!token-name' -role PVEVMAdmin
  ```
- **Network timeouts**: May require retry logic for bulk operations
- **Storage space**: Validation prevents backup failures
- **VM lock detection**: Prevents concurrent operation conflicts
- **Connection refused**: Verify `PVE_HOST`, firewall, and SSL/TLS settings

### CI/CD Issues
- **Release not created**: Verify tag format starts with `v` (e.g., `v1.0.1`)
- **Build fails in Actions**: Check Go version compatibility (workflow uses 1.22)
- **Upload fails**: Verify `permissions: contents: write` in workflow
- **Checksums missing**: Ensure `sha256sum` available in build environment

## Build and Release Process (Go)

### Local Build
```bash
make build          # Current platform only
make build-all      # All platforms (Linux, macOS, Windows)
```

### Create Release
```bash
make release        # Creates tar.gz/zip archives in build/release/
```

### Cross-compilation Targets
- Linux: amd64 (primary), arm64
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Windows: amd64

**Build Priority**: Per user preferences, only **Linux (amd64)** and **Windows (amd64)** builds are needed for releases to save storage. The CI/CD pipeline builds only these two platforms.

### CI/CD with GitHub Actions

**Automated Release Process**:
1. Push a tag matching `v*` pattern:
   ```bash
   git tag -a v1.0.1 -m "Release v1.0.1 - Bug fixes and improvements"
   git push origin v1.0.1
   ```

2. GitHub Actions automatically:
   - Builds Linux (amd64) and Windows (amd64) binaries
   - Generates SHA256 checksums
   - Creates GitHub release with binaries and documentation

**Manual Emergency Release** (if GitHub Actions unavailable):
```bash
cd proxmox-admin-cli/
make build-all
make release
# Upload manually via GitHub web UI or glab
```

**Troubleshooting CI/CD**:
- Check workflow status: https://github.com/yg-codes/proxmox/actions
- Verify tag format: Must start with `v` (e.g., `v1.0.1`)
- Ensure Go version in workflow matches go.mod (1.22 in workflow, 1.21+ required)
- Check build logs for compilation errors
- Verify permissions for `GITHUB_TOKEN` (needs `contents: write`)