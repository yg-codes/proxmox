# Command Reference

Complete reference for all `pve` CLI commands, flags, and parameters.

## Global Flags

All commands inherit these flags from the root command.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | | string | | Config file path |
| `--batch` | | bool | `false` | Batch mode — no interactive prompts |
| `--yes` | `-y` | bool | `false` | Auto-confirm operations |
| `--verbose` | `-v` | bool | `false` | Verbose output |
| `--quiet` | `-q` | bool | `false` | Quiet output |
| `--dry-run` | | bool | `false` | Preview without making changes |

---

## Command Hierarchy

```
pve
├── vm [--node]
│   ├── list
│   ├── details
│   ├── start
│   ├── stop
│   ├── shutdown
│   ├── snapshot
│   │   ├── create
│   │   ├── list
│   │   ├── rollback
│   │   └── delete
│   ├── backup
│   │   ├── create
│   │   ├── list
│   │   ├── restore
│   │   └── delete
│   └── bulk
│       ├── start
│       ├── stop
│       ├── shutdown
│       └── backup
├── cluster
│   ├── task
│   │   ├── list
│   │   ├── running
│   │   ├── failed
│   │   ├── status
│   │   ├── log
│   │   └── stop
│   ├── storage
│   │   ├── list-backup
│   │   └── list-vm
│   └── network
│       ├── list
│       ├── summary
│       ├── show
│       ├── create-bridge
│       ├── delete
│       ├── apply
│       ├── revert
│       ├── sdn
│       │   ├── zones
│       │   └── vnets
│       └── firewall
│           └── rules
├── node
│   ├── list
│   ├── status
│   ├── services
│   ├── service
│   │   ├── start
│   │   ├── stop
│   │   ├── restart
│   │   └── status
│   ├── reboot
│   ├── shutdown
│   ├── version
│   └── resource
│       ├── stats
│       ├── list
│       ├── nodes
│       ├── vms
│       ├── storages
│       ├── node
│       ├── vm
│       └── history
└── container
    ├── list
    ├── summary
    ├── show
    ├── status
    ├── create
    ├── start
    ├── stop
    ├── shutdown
    ├── restart
    ├── delete
    ├── clone
    └── snapshot
        ├── create
        ├── list
        ├── rollback
        └── delete
```

---

## VM Management

### `pve vm`

Persistent flag inherited by all VM subcommands:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--node` | string | | Filter to VMs on a specific node |

### `pve vm list`

List all VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--status` | string | | Filter by status: `running`, `stopped` |

### `pve vm details`

Show VM details. Requires a single VM ID.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--vmid` | stringSlice | | **yes** | VM ID (single VM only) |

### `pve vm start`

Start one or more VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |

### `pve vm stop`

Force-stop one or more VMs. Not a graceful shutdown.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |

### `pve vm shutdown`

Gracefully shut down one or more VMs (ACPI signal).

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |

---

## VM Snapshots

### `pve vm snapshot create`

Create snapshots for one or more VMs. Use `--prefix` for timestamped names or `--name` for exact names.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |
| `--prefix` | string | | Snapshot name prefix (timestamp appended) |
| `--name` | string | | Exact snapshot name |
| `--vmstate` | bool | `false` | Include VM state (RAM) in snapshot |

### `pve vm snapshot list`

List snapshots for one or more VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |

### `pve vm snapshot rollback`

Rollback one or more VMs to a named snapshot. VM must be stopped.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--vmid` | stringSlice | | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | | VM names (comma-separated) |
| `--snapshot` | string | | **yes** | Snapshot name to rollback to |

### `pve vm snapshot delete`

Delete snapshots from one or more VMs. Use `--snapshot` for specific names or `--all` to remove all.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |
| `--snapshot` | stringSlice | | Snapshot name(s) to delete (comma-separated) |
| `--all` | bool | `false` | Delete all snapshots |

---

## VM Backups

### `pve vm backup create`

Create a backup of one or more VMs.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--vmid` | stringSlice | | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | | VM names (comma-separated) |
| `--storage` | string | | **yes** | Target storage ID |
| `--mode` | string | `snapshot` | | Backup mode: `snapshot`, `suspend`, `stop` |
| `--compress` | string | `zstd` | | Compression: `zstd`, `gzip`, `lzo` |

### `pve vm backup list`

List backups for VMs or across storage.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | stringSlice | | VM IDs (comma-separated, ranges supported) |
| `--vmname` | stringSlice | | VM names (comma-separated) |
| `--storage` | string | | Storage to check (checks all if omitted) |
| `--all` | bool | `false` | List all backups in storage (requires `--storage`) |

### `pve vm backup restore`

Restore a VM from a backup file.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--vmid` | stringSlice | | **yes** | VM ID (typically one) |
| `--backup-file` | string | | **yes** | Backup volid |
| `--node` | string | | **yes** | Target node |
| `--storage` | string | | | Target storage |

### `pve vm backup delete`

Delete VM backups. Supports multiple selection strategies.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--vmid` | stringSlice | | **yes** | VM IDs |
| `--backup-file` | string | | | Specific backup volid to delete |
| `--pattern` | string | | | Delete backups matching glob pattern |
| `--keep-count` | int | `0` | | Keep only N most recent backups |
| `--max-age-days` | int | `0` | | Delete backups older than N days |
| `--storage` | string | | | Storage to search |

---

## VM Bulk Operations

Bulk commands auto-discover matching VMs across the cluster. All bulk commands inherit `--node` from `pve vm` and support `--vmid` to target specific VMs instead of all matching VMs.

### `pve vm bulk start`

Start all stopped VMs. Auto-filters to stopped VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | string | | Comma-separated VM IDs to target (default: all stopped VMs) |

### `pve vm bulk stop`

Force-stop all running VMs. Auto-filters to running VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | string | | Comma-separated VM IDs to target (default: all running VMs) |

### `pve vm bulk shutdown`

Gracefully shut down all running VMs (ACPI signal). Auto-filters to running VMs.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--vmid` | string | | Comma-separated VM IDs to target (default: all running VMs) |

### `pve vm bulk backup`

Backup all VMs.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--storage` | string | | **yes** | Target storage ID |
| `--mode` | string | `snapshot` | | Backup mode: `snapshot`, `suspend`, `stop` |
| `--compress` | string | `zstd` | | Compression: `zstd`, `gzip`, `lzo` |
| `--vmid` | string | | | Comma-separated VM IDs to target (default: all VMs) |

---

## Cluster

### `pve cluster task list`

List cluster tasks.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--node` | string | | Filter by node |
| `--running` | bool | `false` | Show only running tasks |
| `--errors` | bool | `false` | Show only failed tasks |
| `--type` | string | | Filter by task type (e.g., `vzdump`) |
| `--user` | string | | Filter by user |
| `--limit` | int | `50` | Maximum number of tasks to return |

### `pve cluster task running`

Quick view of currently running tasks. No flags.

### `pve cluster task failed`

Quick view of recently failed tasks. No flags.

### `pve cluster task status`

Get detailed status of a specific task.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--upid` | string | | **yes** | Task UPID |

### `pve cluster task log`

View task log output.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--upid` | string | | **yes** | Task UPID |
| `--tail` | int | `100` | | Number of lines from end |
| `--follow` | bool | `false` | | Follow log output (streaming) |

### `pve cluster task stop`

Stop a running task.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--upid` | string | | **yes** | Task UPID |

---

## Cluster Storage

### `pve cluster storage list-backup`

List backup-capable storages. No flags.

### `pve cluster storage list-vm`

List VM disk storages. No flags.

---

## Cluster Network

### `pve cluster network list`

List network interfaces on a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--type` | string | | | Filter by type: `bridge`, `bond`, `eth`, `vlan` |
| `--active` | string | | | Filter by active status: `true`, `false` |

### `pve cluster network summary`

Show network interface summary for a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve cluster network show`

Show details of a specific network interface.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--iface` | string | | **yes** | Interface name (e.g., `vmbr0`) |

### `pve cluster network create-bridge`

Create a new network bridge interface.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--iface` | string | | **yes** | Interface name (e.g., `vmbr1`) |
| `--bridge-ports` | string | | **yes** | Bridge ports (e.g., `eth1`) |
| `--address` | string | | | IP address |
| `--netmask` | string | | | Network mask |
| `--gateway` | string | | | Gateway |
| `--comments` | string | | | Description/comments |
| `--autostart` | bool | `true` | | Start on boot |
| `--vlan-aware` | bool | `false` | | Enable VLAN awareness |

### `pve cluster network delete`

Delete a network interface.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--iface` | string | | **yes** | Interface name |

### `pve cluster network apply`

Apply pending network configuration changes.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve cluster network revert`

Revert pending network configuration changes.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve cluster network sdn zones`

List SDN zones.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | | Filter by zone type: `vlan`, `vxlan`, `qinq`, `simple` |

### `pve cluster network sdn vnets`

List SDN virtual networks.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--zone` | string | | Filter by zone name |

### `pve cluster network firewall rules`

List firewall rules for a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

---

## Node Management

### `pve node list`

List all cluster nodes. No flags.

### `pve node status`

Show node status and resource usage.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve node services`

List services on a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve node service start`

Start a service.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--service` | string | | **yes** | Service name |

### `pve node service stop`

Stop a service.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--service` | string | | **yes** | Service name |

### `pve node service restart`

Restart a service.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--service` | string | | **yes** | Service name |

### `pve node service status`

Get service status.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--service` | string | | **yes** | Service name |

### `pve node reboot`

Reboot a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve node shutdown`

Shut down a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve node version`

Get node PVE version.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

---

## Node Resources

### `pve node resource stats`

Show cluster resource summary. No flags.

### `pve node resource list`

List all cluster resources with filters.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | | Filter by type: `node`, `qemu`, `lxc`, `storage` |
| `--node` | string | | Filter by node |
| `--status` | string | | Filter by status |

### `pve node resource nodes`

List node resources.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--node` | string | | Filter by specific node |
| `--status` | string | | Filter by status: `online`, `offline` |

### `pve node resource vms`

List VM resources.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | | Filter by type: `qemu`, `lxc` |
| `--node` | string | | Filter by node |
| `--status` | string | | Filter by status: `running`, `stopped` |

### `pve node resource storages`

List storage resources.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--node` | string | | Filter by node |
| `--status` | string | | Filter by status |

### `pve node resource node`

Show detailed resource usage for a node.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |

### `pve node resource vm`

Show detailed resource usage for a VM.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | VM ID |
| `--type` | string | `qemu` | | VM type: `qemu`, `lxc` |

### `pve node resource history`

Show resource usage history (RRD data).

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | | VM ID (for VM-specific history) |
| `--type` | string | `qemu` | | VM type: `qemu`, `lxc` |
| `--timeframe` | string | `hour` | | Time range: `hour`, `day`, `week`, `month`, `year` |

---

## Container Management

### `pve container list`

List LXC containers.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--node` | string | | Filter by node |
| `--status` | string | | Filter by status: `running`, `stopped` |
| `--name` | string | | Filter by name (substring match) |
| `--template` | bool | `false` | Show only templates |

### `pve container summary`

Show container summary statistics. No flags.

### `pve container show`

Show container configuration details.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container status`

Show container runtime status.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container create`

Create a new LXC container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--ostemplate` | string | | **yes** | OS template volume ID |
| `--storage` | string | | **yes** | Root disk storage |
| `--hostname` | string | | | Container hostname |
| `--description` | string | | | Description |
| `--password` | string | | | Root password |
| `--ssh-keys` | string | | | SSH public keys |
| `--cpus` | int | | | Number of CPU cores |
| `--memory` | int64 | | | Memory in MB |
| `--swap` | int64 | | | Swap in MB |
| `--nesting` | bool | `false` | | Enable nesting |
| `--unprivileged` | bool | `false` | | Unprivileged container |
| `--onboot` | bool | `false` | | Start on boot |
| `--protected` | bool | `false` | | Protection flag |

### `pve container start`

Start a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container stop`

Force-stop a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container shutdown`

Gracefully shut down a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--timeout` | int | `60` | | Shutdown timeout in seconds |

### `pve container restart`

Restart a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container delete`

Delete a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--purge` | bool | `false` | | Purge container data |

### `pve container clone`

Clone a container.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Source node |
| `--vmid` | int | | **yes** | Source container ID |
| `--newid` | int | | **yes** | New container ID |
| `--hostname` | string | | | New hostname |
| `--description` | string | | | Description |
| `--storage` | string | | | Target storage |
| `--target` | string | | | Target node |
| `--snapshot` | string | | | Snapshot to clone from |
| `--full` | bool | `false` | | Full clone (default: linked) |

---

## Container Snapshots

### `pve container snapshot create`

Create a container snapshot.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--name` | string | | **yes** | Snapshot name |
| `--description` | string | | | Description |

### `pve container snapshot list`

List container snapshots.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |

### `pve container snapshot rollback`

Rollback container to a snapshot.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--name` | string | | **yes** | Snapshot name |

### `pve container snapshot delete`

Delete a container snapshot.

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--node` | string | | **yes** | Node name |
| `--vmid` | int | | **yes** | Container ID |
| `--name` | string | | **yes** | Snapshot name |

---

## VM Selection Patterns

VM commands that accept `--vmid` support these selection patterns:

| Pattern | Example | Description |
|---------|---------|-------------|
| Single ID | `7303` | One VM |
| Comma-separated | `7301,7302,7303` | Multiple specific VMs |
| Range | `7201-7205` | All VMs in range |
| Wildcard | `72*` | Pattern matching |
| Interactive | `i` | Checkbox-style selection UI |

The `--vmname` flag accepts comma-separated VM names as an alternative to IDs.

---

## Backup File Format

Backup volume IDs use the format `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`.

Examples:
- File-based: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
- PBS: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`

---

## Snapshot Naming

- Maximum snapshot **name** length: **40 characters**, applied to the full assembled name `<prefix>-<vmname>-<YYYYMMDD-HHMM>` (there is no separate limit on the `--prefix` input)
- Invalid characters are automatically cleaned (replaced with `-`, runs collapsed, leading/trailing trimmed)
- When using `--prefix`, a timestamp is appended automatically; for long VM names the `-YYYYMMDD-HHMM` suffix may be **truncated** because the 40-char cap applies to the whole name (e.g. `bulkdemo-fsx-dev-workstation03-20260626` — the `-HHMM` is cut)
- The keyword `vmstate` in prefix or name enables RAM inclusion
