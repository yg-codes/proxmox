# pve

Single-binary Proxmox VE management CLI with AWS-style command interface.

[![Release](https://img.shields.io/github/v/release/yg-codes/proxmox)](https://github.com/yg-codes/proxmox/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/yg-codes/proxmox/release.yml)](https://github.com/yg-codes/proxmox/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yg-codes/proxmox)](https://github.com/yg-codes/proxmox)

## How-to

### Install

**From pre-built binary (Linux amd64):**
```bash
curl -LO https://github.com/yg-codes/proxmox/releases/latest/download/pve-linux-amd64
sudo install -m 755 pve-linux-amd64 /usr/local/bin/pve
```

**From source with mise:**
```bash
cd ~/src/github/proxmox
mise run install    # builds + installs pve to $GOPATH/bin
```

**From source with make:**
```bash
make install
```

### Configure

Set environment variables for Proxmox API access:

```bash
export PVE_HOST=proxmox-host.com
export PVE_USER=username@pam
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value
```

For password auth instead of token: set `PVE_PASSWORD` instead of `PVE_TOKEN_NAME`/`PVE_TOKEN_VALUE`. Token auth is recommended for production.

To create an API token on a Proxmox node:
```bash
./scripts/create-api-token.sh pve1
```

### Verify

```bash
pve --version
# pve v1.2.0 (commit: 4d763aa, built: 2026-05-23)

pve node list
# NAME    STATUS   CPU   MEM       UPTIME
# pve1    online   12%   45.2%     42d 5h
```

### Release a New Version

```bash
git tag -a v1.3.0 -m "Release v1.3.0"
git push origin v1.3.0
# GitHub Actions builds Linux amd64 + Windows amd64 binaries automatically
```

## Reference

### Commands

| Command | Purpose |
|---------|---------|
| `pve cluster task list` | List cluster tasks |
| `pve cluster storage list-backup` | List backup storages |
| `pve cluster network list --node pve1` | List network interfaces |
| `pve node list` | List cluster nodes |
| `pve node status --node pve1` | Node status and resources |
| `pve node services --node pve1` | List node services |
| `pve node reboot --node pve1 --confirm` | Reboot a node (dry-run by default) |
| `pve node shutdown --node pve1 --confirm` | Shutdown a node |
| `pve vm list` | List all VMs |
| `pve vm start --vmid 100` | Start a VM |
| `pve vm stop --vmid 100` | Stop a VM |
| `pve vm shutdown --vmid 100` | Graceful shutdown |
| `pve vm snapshot create --vmid 100 --prefix backup` | Create snapshot |
| `pve vm snapshot list --vmid 100` | List snapshots |
| `pve vm snapshot rollback --vmid 100 --snapshot name` | Rollback to snapshot |
| `pve vm snapshot delete --vmid 100 --snapshot name` | Delete snapshot |
| `pve vm backup create --vmid 100 --storage local` | Create backup |
| `pve vm backup list --vmid 100` | List VM backups |
| `pve vm backup list --all --storage local` | List all backups on storage |
| `pve vm backup restore --vmid 100 --backup-file "..." --node pve1` | Restore backup |
| `pve vm bulk start` | Start all stopped VMs |
| `pve vm bulk stop` | Stop all running VMs |
| `pve vm bulk backup --storage local` | Backup all VMs |
| `pve container list` | List containers |
| `pve container start --node pve1 --vmid 200` | Start container |

### VM Selection Patterns

| Pattern | Example | Matches |
|---------|---------|---------|
| Single ID | `--vmid 7303` | One VM |
| List | `--vmid 7201,7203,7205` | Specific VMs |
| Range | `--vmid 7201-7205` | All VMs in range |
| Wildcard | `--vmid 72*` | Pattern match |
| Keyword | `--vmid running` | All running VMs |
| Interactive | `--vmid i` | Checkbox UI |

### Safety Flags

| Flag | Behavior |
|------|----------|
| `--dry-run` | Preview without executing |
| `--yes` / `-y` | Skip confirmation prompts |
| `--confirm` | Required for node power operations |

### Build Commands

| Command | Purpose |
|---------|---------|
| `mise run build` | Build `pve` binary (uses Go from mise) |
| `mise run install` | Build + install to `$GOPATH/bin` |
| `mise run clean` | Remove build artifacts |
| `make build-all` | Cross-compile all platforms |
| `make release` | Create release archives |

### Configuration

| Variable | Required | Description |
|----------|:--------:|-------------|
| `PVE_HOST` | Yes | Proxmox hostname or IP |
| `PVE_USER` | Yes | Username (e.g. `root@pam`) |
| `PVE_TOKEN_NAME` | Token auth | API token name |
| `PVE_TOKEN_VALUE` | Token auth | API token secret |
| `PVE_PASSWORD` | Password auth | User password |

### Directory Structure

```
proxmox/
├── cmd/                      # Cobra commands
├── pkg/                      # Core packages
├── scripts/                  # API token setup, SSH runner
├── .github/workflows/        # CI/CD release pipeline
├── .mise.toml                # mise build tasks
├── FUNCTIONAL_SPECIFICATION.md
└── CLAUDE.md                 # Dev guidelines
```

## Explanation

### Architecture

`pve` is a Go CLI built with Cobra. It talks to the Proxmox VE REST API over HTTPS — no agent on the Proxmox side, no local config files. All state comes from the cluster API at runtime.

Binary compiles statically with version/commit/build-time injected via ldflags. No runtime dependencies beyond the Proxmox API endpoint.

Full feature reference: [FUNCTIONAL_SPECIFICATION.md](FUNCTIONAL_SPECIFICATION.md)

---
Last Updated: 2026-05-23
