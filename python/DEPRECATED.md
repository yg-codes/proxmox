# Python Implementation - DEPRECATED

**Deprecation Date:** December 2025
**Status:** ⚠️ No longer maintained

## Notice

The Python implementation of Proxmox management tools has been **deprecated** and is no longer actively maintained.

### Why Deprecated?

The Go implementation (`pve`) has achieved **100% feature parity** with the Python tools, plus:

- ✅ **5-10x faster performance**
- ✅ **Significantly more features** (cluster, node, container management)
- ✅ **Better user experience** (AWS-style CLI hierarchy)
- ✅ **Zero runtime dependencies** (single binary)
- ✅ **Automated CI/CD releases**
- ✅ **Production-ready** with comprehensive testing

### Migration Guide

See the comprehensive analysis: [PYTHON_DEPRECATION_ANALYSIS.md](../PYTHON_DEPRECATION_ANALYSIS.md)

#### Quick Migration

**Before (Python):**
```bash
# Two separate tools
pve-snapshot-manager create --vmid 7303 --prefix backup
pve-vm-manager-modular backup --vmid 7303 --storage local
```

**After (Go):**
```bash
# Single unified tool
pve vm snapshot create --vmid 7303 --prefix backup
pve vm backup create --vmid 7303 --storage local
```

### Command Translation

| Python (Old) | Go (New) |
|-------------|----------|
| `pve-snapshot-manager create` | `pve vm snapshot create` |
| `pve-snapshot-manager list` | `pve vm snapshot list` |
| `pve-snapshot-manager rollback` | `pve vm snapshot rollback` |
| `pve-snapshot-manager delete` | `pve vm snapshot delete` |
| `pve-vm-manager-modular start` | `pve vm start` |
| `pve-vm-manager-modular stop` | `pve vm stop` |
| `pve-vm-manager-modular backup` | `pve vm backup create` |
| `pve-vm-manager-modular list-backups` | `pve vm backup list` |
| `pve-vm-manager-modular restore` | `pve vm backup restore` |
| `pve-vm-manager-modular delete-backups` | `pve vm backup delete` |

### Installation of Go CLI

**Linux:**
```bash
curl -LO https://github.com/yg-codes/proxmox/releases/latest/download/pve-linux-amd64
sudo install -m 755 pve-linux-amd64 /usr/local/bin/pve
pve --version
```

**Windows:**
```powershell
# Download from: https://github.com/yg-codes/proxmox/releases/latest
# Rename to pve.exe and add to PATH
```

**macOS:**
```bash
# Intel Macs
curl -LO https://github.com/yg-codes/proxmox/releases/latest/download/pve-darwin-amd64
sudo install -m 755 pve-darwin-amd64 /usr/local/bin/pve

# Apple Silicon
curl -LO https://github.com/yg-codes/proxmox/releases/latest/download/pve-darwin-arm64
sudo install -m 755 pve-darwin-arm64 /usr/local/bin/pve
```

### Same Authentication

Both implementations use the same environment variables:

```bash
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value
```

### Support

For issues or questions:
- See the main [README.md](../README.md)
- Check the [Go implementation documentation](../proxmox-admin-cli/README.md)
- Review the [deprecation analysis](../PYTHON_DEPRECATION_ANALYSIS.md)

---

**This Python implementation will not receive:**
- Bug fixes
- Security updates
- New features
- Support

**Please migrate to the Go implementation (`pve`) immediately.**
