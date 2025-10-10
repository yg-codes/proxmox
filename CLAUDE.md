# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This repository provides comprehensive Proxmox VE management tools in three implementations:

1. **Go Implementation** (Recommended for production): High-performance, single-binary snapshot manager
2. **Python Modular** (Stable): Clean architecture with VM and snapshot management
3. **Python Legacy** (Deprecated): Original monolithic implementations

## Directory Structure

```
proxmox/
├── proxmox-admin-cli/              # Go implementation (5-10x faster)
├── modular/                        # Python modular implementations
│   ├── snapshot-manager/          # Modular snapshot management
│   ├── vm-manager/                # Modular VM & backup management
│   └── pve-snapshots-cli.py       # CLI wrapper
└── legacy/                         # Deprecated monolithic scripts
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

# Direct usage (after build)
./build/proxmox-admin-cli --help
./build/proxmox-admin-cli create --vmid 7303 --prefix backup
```

### Python Modular Implementation

```bash
# Installation (choose one approach)

# Option 1: Global installation with pipx (recommended)
cd modular/
pipx install ./snapshot-manager/
pipx install ./vm-manager/
pve-snapshot-manager --help
pve-vm-manager-modular --help

# Option 2: Project development with uv
cd modular/snapshot-manager/
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
- Use `gofmt` for formatting
- Run `go vet` before commits
- Use `golangci-lint` for comprehensive linting
- Explicit error handling (no exceptions)
- Use context for cancellation and timeouts

### Python
- Python 3.8+ with type hints
- Black formatter (88-character line limit)
- PEP 8 compliance
- Comprehensive docstrings for public methods
- Custom `ProxmoxAPIError` exception
- Explicit error handling patterns

## Testing Considerations

- **Go**: Use standard `go test`, coverage with `make test-coverage`
- **Python**: Use pytest (if tests present)
- Both require live Proxmox environment
- Use development VM IDs for testing (avoid production)
- Test API token permissions before bulk operations
- Verify storage availability before backup operations

## Performance Benchmarks (Go vs Python)

| Operation | Python (ThreadPool=3) | Go (Goroutines=3) | Improvement |
|-----------|----------------------|-------------------|-------------|
| Create 10 snapshots | 45.2s | 8.7s | 5.2x faster |
| Delete 20 snapshots | 52.1s | 9.3s | 5.6x faster |
| List 50 VMs | 12.4s | 2.1s | 5.9x faster |
| Rollback 5 VMs | 78.9s | 12.4s | 6.4x faster |

*Benchmarks on Proxmox 7.4 cluster, 3 nodes, 100+ VMs*

## Migration Guide

### Python Legacy → Python Modular
1. Replace `legacy/pve_snapshots/` with `modular/snapshot-manager/`
2. Replace `legacy/proxmox-vm-manager/` with `modular/vm-manager/`
3. Use `./pve-snapshots-cli.py` for convenient snapshot management

### Python → Go (Drop-in Replacement)
```bash
# Python version
python3 main.py create --vmid 7303 --prefix backup

# Go version (identical CLI)
proxmox-admin-cli snapshot create --vmid 7303 --prefix backup
```

**Migration Benefits:**
- Same CLI interface and functionality
- 5-10x performance improvement
- Single binary deployment (no Python environment)
- Compile-time error detection
- Superior concurrency handling

## Common Issues

- **"Permission check failed"**: Requires proper token ACL configuration
- **Network timeouts**: May require retry logic for bulk operations
- **Storage space**: Validation prevents backup failures
- **VM lock detection**: Prevents concurrent operation conflicts
- **Go build errors**: Ensure Go 1.21+ is installed
- **Python dependencies**: Use `uv sync` or `pip install requests urllib3`

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
- Linux: amd64, arm64
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Windows: amd64

Note: Per user preferences, only Linux (amd64) and Windows builds are typically needed to save storage.
