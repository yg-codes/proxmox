# pve CLI — Demonstration & Validation Runbook (Phase 1: Snapshots)

**Date**: 2026-06-26

**Environment**: dev (`fsx-dev` cluster: `fsx-dev-pve21`, `fsx-dev-pve22`, `fsx-dev-pve23` — one cluster)

**Release under test**: `pve` **v1.5.0** (GitHub release `yg-codes/proxmox`; built via GoReleaser, ldflags stamp `v1.5.0`)

**Repo**: `github.com/yg-codes/proxmox` (personal GitHub mirror)

**Purpose**: Validate the `pve` snapshot subcommands end-to-end the way an end user receives the tool — install the released binary (or run via mise), authenticate via API token (optionally resolved from 1Password), then exercise the full snapshot lifecycle (list → create → rollback → delete) on a single VM, driven by both `--vmid` and `--vmname`.

> **This is Phase 1 of a phased runbook series.** Phase 1 covers snapshot functions only — the single-VM lifecycle. Later phases will cover bulk operations, backup management, VM power ops, nodes, storage, etc.

## Overview

This runbook installs the released `pve` binary, points it at the `fsx-dev` Proxmox cluster with an API token, and runs the four snapshot verbs against a single VM — once by VMID and once by VM name. All snapshots created here are **disk-only** (`--vmstate` omitted) and are deleted at the end, leaving the VM in its original snapshot state. Because a disk-only rollback powers the VM off, **every rollback is followed by a power-status check and a start if needed** (see Part 2.5 / 2B.5).

**Test target (running before the test):**

| VMID | Name | Node | Role in test |
|------|------|------|--------------|
| 8701 | fsx-dev-scraper01 | fsx-dev-pve22 | single-VM lifecycle (by `--vmid` and by `--vmname`) |

> **Why these results are pre-filled:** The ✅ Result lines record the *expected* outcome. The validator executing this runbook should **overwrite each Result** with their own observed output (or mark ❌ on deviation). Result lines marked `*(to be observed)*` have not yet been run against the live cluster.

> **⚠️ Two `pve`-specific gotchas baked into every command below (these differ from the sibling `proxmox-snapshot-manager` tool):**
> 1. **Command path is `pve vm snapshot <verb>`** — NOT `pve snapshot <verb>`. The snapshot verbs' own `--help` examples show `pve snapshot ...`, but that path is **stale help text and fails** with "unknown command". Only `vm`, `cluster`, `node`, `container` are attached to the root. Always use `pve vm snapshot ...`.
> 2. **The snapshot-name flag is `--snapshot`** — NOT `--snapshot_name` / `--snap`. (And VM selection is `--vmname`, one word — there is no `--vm-name`.)

---

## Prerequisites

- **`pve` v1.5.0** installed — either from the GitHub release archive, via `go install github.com/yg-codes/proxmox/pve@v1.5.0`, or via mise (`go:github.com/yg-codes/proxmox/pve` = `v1.5.0`). See Part 1.
- An **API token** on the `fsx-dev` cluster with at least `PVEVMAdmin` on the target VM (and the VM's node). The token name is the plain label; the token value is the secret (or an `op://` reference).
- **1Password CLI** (`op` / `op.exe`) authenticated — only required if you pass `op://` references for credentials. The tool resolves any credential env var whose value starts with `op://` at startup. Plaintext credentials skip this.
- SSH as `root` to `fsx-dev-pve22` — needed **only** for the post-rollback `qm start` (the `pve` snapshot verbs do **not** do power operations).
- **Approval:** per project policy, snapshot operations on VMs other than **7303** require explicit approval. **8701 is not on the no-approval list — obtain approval before executing this runbook against the live cluster.**

---

## Part 0: Pre-checks

### Step 0.1: Confirm the target VM is running and on the expected node

**Command**:
```bash
ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone \
  "pvesh get /cluster/resources --type vm 2>/dev/null | grep -E ' 8701 '"
```

**Expected**: VM 8701 (`fsx-dev-scraper01`) appears with `running` status on node `fsx-dev-pve22`.

**Result**: ✅ *(to be observed)*

### Step 0.2: (Optional) Verify 1Password CLI is authenticated

Only needed if credentials will be `op://` references.

**Command**:
```bash
op account list 2>&1 | head -3      # WSL/Windows: op.exe account list
```

**Expected**: Lists the `finstadiumx.1password.com` account.

**Result**: ✅ *(to be observed, or N/A if using plaintext credentials)*

---

## Part 1: Install From Release (end-user flow)

### Step 1.1: Install the binary

Pick **one** of the following. The release-archive path is the canonical end-user flow.

**Option A — GitHub release archive (canonical):**
```bash
rm -rf /tmp/pve-release-test && mkdir -p /tmp/pve-release-test && cd /tmp/pve-release-test
# download the archive matching your platform, e.g. linux-amd64:
gh release download v1.5.0 --repo yg-codes/proxmox --pattern 'proxmox-1.5.0-linux-amd64.tar.gz' --dir .
tar -xzf proxmox-1.5.0-linux-amd64.tar.gz
sudo install -m 0755 pve /usr/local/bin/pve
```

**Option B — `go install` (out-of-the-box since v1.5.0):**
```bash
go install github.com/yg-codes/proxmox/pve@v1.5.0
# → $(go env GOPATH)/bin/pve
```

**Option C — mise:**
```bash
mise install    # with "go:github.com/yg-codes/proxmox/pve" = "v1.5.0" in config
```

**Expected**: `pve` on `PATH`.

**Result**: ✅ *(to be observed)*

### Step 1.2: Verify version

**Command**:
```bash
pve --version
which pve
```

**Expected**: `pve v1.5.0 (commit <short>, built <date>)`. (Note: a `go install`/mise build without ldflags reports `pve dev (commit: none, built: unknown)` — the binary still works; only the version string is blank. Release-archive binaries stamp `v1.5.0` correctly.)

**Result**: ✅ *(to be observed)*

### Step 1.3: Export credentials

Point at any cluster node. `PVE_TOKEN_NAME` is the plain token name; `PVE_TOKEN_VALUE` is either the secret or an `op://` reference resolved at startup.

**Command**:
```bash
export PVE_HOST=fsx-dev-pve22.fsx.zone
export PVE_USER=snapshot@pve
export PVE_TOKEN_NAME=snapshot
export PVE_TOKEN_VALUE='op://SRE/fsx-dev-pve2x snapshot/credential'   # or the plaintext secret
```

**Expected**: No output. Variables set in the shell that runs the remaining commands.

**Result**: ✅ Environment exported

---

## Part 2: Single-VM Lifecycle by VMID — VM 8701

> **Reminder:** every command uses `pve vm snapshot <verb>`, and the snapshot-name flag is `--snapshot`.

### Step 2.1: List snapshots (read-only auth check)

Confirms credentials resolve and the API is reachable.

**Command**:
```bash
pve vm snapshot list --vmid 8701 2>&1 | tail -15
```

**Expected**: Output like `VM 8701: fsx-dev-scraper01`, `Status: 🟢 running`, then `Snapshots (N total):` (or `No snapshots found for VM 8701` if none). A `403 Permission check failed` means the token role is wrong.

**Result**: ✅ *(to be observed)*

### Step 2.2: Create a disk-only snapshot

**Command**:
```bash
pve vm snapshot create --vmid 8701 --prefix demo -y 2>&1 | tail -8
```

**Expected**: `✅ Snapshot 'demo-fsx-dev-scraper01-<YYYYMMDD-HHMM>' created successfully for VM 8701`. The generated name is `<prefix>-<vmname>-<YYYYMMDD-HHMM>` (timestamp is minute-granular). **Capture the exact name** for Steps 2.4–2.6:

```bash
SNAP=$(pve vm snapshot list --vmid 8701 2>&1 | grep -oP 'demo-fsx-dev-scraper01-\d+-\d+')
echo "Captured: $SNAP"
```

**Result**: ✅ *(to be observed — record the captured snapshot name)*

### Step 2.3: Verify the snapshot is listed

**Command**:
```bash
pve vm snapshot list --vmid 8701 2>&1 | grep -E 'demo-fsx-dev-scraper01'
```

**Expected**: One line showing the `demo-fsx-dev-scraper01-<TS>` snapshot, `VM State: ❌ Not included (disk only)`.

**Result**: ✅ *(to be observed)*

### Step 2.4: Roll back to the snapshot

Uses the `SNAP` variable captured in Step 2.2. `--snapshot` is **required** (single value).

**Command**:
```bash
pve vm snapshot rollback --vmid 8701 --snapshot "$SNAP" -y 2>&1 | tail -8
```

**Expected**: `✅ VM 8701 rolled back to snapshot '<SNAP>' successfully`.

**Result**: ✅ *(to be observed)*

### Step 2.5: Post-rollback power check — start VM if stopped (MANDATORY)

A disk-only snapshot has no RAM state, so rollback leaves the VM **powered off** even if it was running. This step is required after every rollback. (`pve` snapshot verbs do not start VMs.)

**Command**:
```bash
ST=$(ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm status 8701")
echo "8701: $ST"
echo "$ST" | grep -q stopped && ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm start 8701"
sleep 3
ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm status 8701"
```

**Expected**: Status reads `stopped` after rollback; after `qm start`, final status reads `running`.

**Result**: ✅ *(to be observed)*

### Step 2.6: Delete the snapshot

**Command**:
```bash
pve vm snapshot delete --vmid 8701 --snapshot "$SNAP" -y 2>&1 | tail -8
```

**Expected**: Success log line for the deletion of `<SNAP>` on VM 8701.

**Result**: ✅ *(to be observed)*

### Step 2.7: Verify clean state

**Command**:
```bash
pve vm snapshot list --vmid 8701 2>&1 | tail -8
```

**Expected**: `No snapshots found for VM 8701` (or only `current` state), VM `Status: 🟢 running`.

**Result**: ✅ *(to be observed)*

---

## Part 2B: Single-VM Lifecycle by VM Name — VM `fsx-dev-scraper01`

Mirrors Part 2's full lifecycle but drives every command with **`--vmname`** instead of `--vmid`, validating that the name selector works end-to-end across all four verbs. Uses a distinct prefix (`vndemo`) so these snapshots never collide with Part 2's `demo-*`. Non-destructive: the snapshot created here is deleted in Step 2B.6.

### Step 2B.1: List by VM name (auth + name resolution)

**Command**:
```bash
pve vm snapshot list --vmname fsx-dev-scraper01 2>&1 | tail -15
```

**Expected**: Identical to Step 2.1 — `VM 8701: fsx-dev-scraper01`, `Status: 🟢 running`. Confirms `--vmname fsx-dev-scraper01` resolves to VMID 8701.

**Result**: ✅ *(to be observed)*

### Step 2B.2: Create a disk-only snapshot by name

**Command**:
```bash
pve vm snapshot create --vmname fsx-dev-scraper01 --prefix vndemo -y 2>&1 | tail -8
```

**Expected**: `✅ Snapshot 'vndemo-fsx-dev-scraper01-<YYYYMMDD-HHMM>' created successfully for VM 8701`. **Capture the exact name** for Steps 2B.4–2B.6:

```bash
VNSNAP=$(pve vm snapshot list --vmname fsx-dev-scraper01 2>&1 | grep -oP 'vndemo-fsx-dev-scraper01-\d+-\d+')
echo "Captured: $VNSNAP"
```

**Result**: ✅ *(to be observed)*

### Step 2B.3: Verify the snapshot is listed (by name)

**Command**:
```bash
pve vm snapshot list --vmname fsx-dev-scraper01 2>&1 | grep -E 'vndemo-fsx-dev-scraper01'
```

**Expected**: One line showing the `vndemo-fsx-dev-scraper01-<TS>` snapshot, disk-only.

**Result**: ✅ *(to be observed)*

### Step 2B.4: Roll back by name

Uses the `VNSNAP` variable captured in Step 2B.2. The VM is located via `--vmname`; the snapshot via `--snapshot`.

**Command**:
```bash
pve vm snapshot rollback --vmname fsx-dev-scraper01 --snapshot "$VNSNAP" -y 2>&1 | tail -8
```

**Expected**: `✅ VM 8701 rolled back to snapshot '<VNSNAP>' successfully`.

**Result**: ✅ *(to be observed)*

### Step 2B.5: Post-rollback power check — start VM if stopped (MANDATORY)

**Command**:
```bash
ST=$(ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm status 8701")
echo "8701: $ST"
echo "$ST" | grep -q stopped && ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm start 8701"
sleep 3
ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm status 8701"
```

**Expected**: `stopped` after rollback → `running` after `qm start`.

**Result**: ✅ *(to be observed)*

### Step 2B.6: Delete by name

**Command**:
```bash
pve vm snapshot delete --vmname fsx-dev-scraper01 --snapshot "$VNSNAP" -y 2>&1 | tail -8
```

**Expected**: Success log line for the deletion of `<VNSNAP>`.

**Result**: ✅ *(to be observed)*

### Step 2B.7: Verify clean state (by name)

**Command**:
```bash
pve vm snapshot list --vmname fsx-dev-scraper01 2>&1 | tail -8
```

**Expected**: `No snapshots found for VM 8701`, VM `Status: 🟢 running`.

**Result**: ✅ *(to be observed)*

---

## Part 3: Post-checks (final state verification)

### Step 3.1: Verify all test snapshots removed and VM running

**Command**:
```bash
pve vm snapshot list --vmid 8701 2>&1 | grep -E 'demo-fsx-dev-scraper01|vndemo-fsx-dev-scraper01|No snapshots'
ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone "qm status 8701"
```

**Expected**: No `demo-*` / `vndemo-*` snapshots. VM 8701 `running`.

**Result**: ✅ *(to be observed)*

### Step 3.2: Clean up the scratch directory (Option A install only)

**Command**:
```bash
cd / && rm -rf /tmp/pve-release-test && ls -d /tmp/pve-release-test 2>&1 || echo "removed"
```

**Expected**: `removed`.

**Result**: ✅ *(to be observed)*

---

## Cleanup & Recovery Plan (if a step fails or leaves residue)

| Situation | Action |
|-----------|--------|
| A `demo-*`/`vndemo-*` snapshot left behind on 8701 | `pve vm snapshot delete --vmid 8701 --snapshot '<exact-name>' -y` |
| VM 8701 left `stopped` after rollback | `ssh root@fsx-dev-pve22.fsx.zone "qm start 8701"`, then confirm `qm status 8701` → running |
| Credentials fail (`op` not signed in / 403) | Re-check 1Password auth (Step 0.2) and the token role on Proxmox; do **not** proceed past Step 2.1 until `list` works |
| `pve snapshot <verb>` fails with "unknown command" | You hit the stale-help-text gotcha — use **`pve vm snapshot <verb>`** instead |
| `--snapshot_name` rejected | Wrong flag — use **`--snapshot`** |
| Generated name capture returns empty | The `grep -oP` pattern expects `<prefix>-<vmname>-<YYYYMMDD-HHMM>`; re-run `pve vm snapshot list --vmid 8701` and copy the exact name manually |

This runbook is non-destructive by design: every snapshot it creates it also deletes, and rollbacks target snapshots taken seconds earlier (no real disk-state change).

---

## Completion Summary

| Step | Description | Status |
|------|-------------|--------|
| 0.1 | Target VM 8701 running on pve22 | ⬜ |
| 0.2 | 1Password CLI authenticated (if `op://` used) | ⬜ |
| 1.1 | `pve` v1.5.0 installed | ⬜ |
| 1.2 | Version verified | ⬜ |
| 1.3 | Credentials exported | ⬜ |
| 2.1 | List by `--vmid` (auth) | ⬜ |
| 2.2 | Create by `--vmid` | ⬜ |
| 2.3 | Verify snapshot listed | ⬜ |
| 2.4 | Rollback by `--vmid` | ⬜ |
| 2.5 | Post-rollback start (8701) | ⬜ |
| 2.6 | Delete by `--vmid` | ⬜ |
| 2.7 | Verify clean (`--vmid`) | ⬜ |
| 2B.1 | List by `--vmname` | ⬜ |
| 2B.2 | Create by `--vmname` | ⬜ |
| 2B.3 | Verify snapshot listed (name) | ⬜ |
| 2B.4 | Rollback by `--vmname` | ⬜ |
| 2B.5 | Post-rollback start (8701) | ⬜ |
| 2B.6 | Delete by `--vmname` | ⬜ |
| 2B.7 | Verify clean (`--vmname`) | ⬜ |
| 3.1 | Final state clean | ⬜ |
| 3.2 | Scratch dir removed | ⬜ |

**Escalation**: If any step fails and the Cleanup & Recovery Plan does not resolve it, stop and seek approval before retrying destructive steps — VM 8701 is not on the project's no-approval testing list. Do not delete snapshots you did not create.
