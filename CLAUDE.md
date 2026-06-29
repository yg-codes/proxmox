# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This repository provides a comprehensive Proxmox VE management CLI written in Go. It offers a high-performance, single-binary admin tool with an AWS-style command hierarchy.

### Requirements

- **Go**: Version 1.21 or higher
- **OS**: Linux, macOS, or Windows (WSL supported)

## Directory Structure

```
proxmox/
├── pve/                        # CLI entry point and commands
├── pkg/                        # Core packages
├── scripts/                    # Setup and utility scripts
│   ├── pve-token.sh            # API user/token manager (create/add/revoke/remove/list)
│   └── test-1password-integration.sh  # Manual op:// credential test
├── Makefile                    # Build automation
├── go.mod / go.sum             # Go module
├── .mise.toml                  # mise build tasks
└── docs/                      # COMMAND_REFERENCE.md, FUNCTIONAL_SPECIFICATION.md, DEMO_RUNBOOK.md
```

## Quick Reference

```bash
make fmt && make vet && make test && make build
```

Complete command and parameter documentation: [docs/COMMAND_REFERENCE.md](docs/COMMAND_REFERENCE.md)

## Development Commands

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
./build/pve --help

# Build + install for testing
make build && make install
```

Usage examples: [README.md](README.md) | Command reference: [docs/COMMAND_REFERENCE.md](docs/COMMAND_REFERENCE.md)

## Configuration

### Authentication (Required)

Environment variables only, no config files needed.

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

# Or use the setup script:
./scripts/pve-token.sh pve1
```

## Architecture

### Module Structure
```
pve/              # CLI entry point
pkg/
├── api/         # HTTP client and authentication
├── vm/          # VM operations and selection
├── snapshot/    # Snapshot lifecycle management
├── backup/      # Backup operations
├── storage/     # Storage management
├── bulk/        # Concurrent bulk operations
├── protection/  # VM protection handling
├── node/        # Node management
├── task/        # Task monitoring
├── resource/    # Resource statistics
├── container/   # LXC container operations
└── network/     # Network configuration
Makefile         # Build automation
go.mod           # Go module definition
```

**Key Technologies:**
- **cobra**: CLI framework
- **logrus**: Structured logging
- **net/http**: HTTP client (stdlib)
- **Environment variables**: Configuration

## Key Features

### VM Selection Patterns
- **Range**: `7201-7205` (all VMs in range)
- **List**: `7201,7203,7205` (specific VMs)
- **Wildcard**: `72*` (pattern matching)
- **Keywords**: `running`, `stopped`, `all`
- **Interactive**: Checkbox-style UI (`--vmid i`)
- **Names**: VM name resolution alongside IDs

### Snapshot Operations
- Create with prefix or exact name
- Optional VM state (RAM) inclusion
- List with configuration details
- Rollback with safety checks
- Delete single, multiple, or all snapshots
- Bulk operations with concurrency

### Backup Management
- **Create**: Multiple modes (snapshot, suspend, stop)
- **List**: Per-VM or storage-wide listing (`--all --storage`)
- **Restore**: VM restoration with protection handling
- **Delete**: By volid, pattern, keep-count, or max-age-days

**Important**: Backup volid format = `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`
- File-based: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
- PBS backup: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`

### Snapshot Naming Constraints
- Maximum snapshot **name** length: 40 characters (applied to the full assembled name `<prefix>-<vmname>-<timestamp>`, not to the prefix alone — there is no separate prefix-length limit)
- Automatic invalid character cleanup
- Intelligent timestamp appending (the `-YYYYMMDD-HHMM` suffix may be truncated for long VM names, since the 40-char cap applies to the whole name)
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

### Git Conventions (GitHub Repository)

**Commit Messages**:
- Clear, descriptive commit messages
- **No ticket ID required** (this is a GitHub repository, not GitLab)
- **Do NOT include** Claude Code attribution:
  ```
  # NEVER INCLUDE:
  Generated with [Claude Code](https://claude.ai/code)
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
- Go artifacts (`*.test`, `coverage.out`)

## Testing & Safety

### Testing Environment
- Use standard `go test`, coverage with `make test-coverage`
- Requires live Proxmox environment
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

## Common Issues

### Go Build Issues
- **"Go version too old"**: Ensure Go 1.21+ is installed (`go version`)
- **"Package not found"**: Run `make deps` to download dependencies
- **Build fails**: Try `make clean && make deps && make build`
- **Cross-compilation errors**: Ensure correct GOOS/GOARCH for target platform

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

## Build and Release Process

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
- Windows: amd64, arm64

All 6 platforms (linux/darwin/windows × amd64/arm64) are built by the GoReleaser CI/CD pipeline on tag push.

### CI/CD with GitHub Actions

**Automated Release Process**:
1. Push a tag matching `v*` pattern:
   ```bash
   git tag -a v1.4.0 -m "Release v1.4.0"
   git push origin v1.4.0
   ```

2. GitHub Actions automatically:
   - Builds all 6 platform binaries via GoReleaser
   - Generates SHA256 checksums
   - Creates GitHub release with binaries and documentation

**Manual Emergency Release** (if GitHub Actions unavailable):
```bash
make build-all
make release
# Upload manually via GitHub web UI
```
