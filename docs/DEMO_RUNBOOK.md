# pve CLI — Demonstration & Validation Runbook (Phase 1: Snapshots)

**Date**: 2026-06-26

**Environment**: dev (`fsx-dev` cluster: `fsx-dev-pve21`, `fsx-dev-pve22`, `fsx-dev-pve23` — one cluster)

**Release under test**: `pve` **v1.5.0** (GitHub release `yg-codes/proxmox`; built via GoReleaser, ldflags stamp `v1.5.0`)

**Repo**: `github.com/yg-codes/proxmox` (personal GitHub mirror)

**Purpose**: Validate the `pve` snapshot subcommands end-to-end the way an end user receives the tool — install the released binary (or run via mise), authenticate via API token (optionally resolved from 1Password), then exercise the full snapshot lifecycle (list → create → rollback → delete) on a single VM (Parts 2 / 2B) and as a concurrent bulk operation across three VMs on three nodes (Parts 3 / 3B), driven by both `--vmid` and `--vmname`.

> **This is Phase 1 of a phased runbook series.** Phase 1 covers snapshot functions — the single-VM lifecycle (Parts 2 / 2B) and the bulk snapshot lifecycle (Parts 3 / 3B). Later phases will cover backup management, VM power ops, nodes, storage, etc.

## Overview

This runbook installs the released `pve` binary, points it at the `fsx-dev` Proxmox cluster with an API token, and runs the four snapshot verbs against a single VM (Parts 2 / 2B) and against three VMs on three nodes as a concurrent bulk operation (Parts 3 / 3B) — once by VMID and once by VM name in each case. All snapshots created here are **disk-only** (`--vmstate` omitted) and are deleted at the end, leaving each VM in its original snapshot state. Because a disk-only rollback powers the VM off, **every rollback is followed by a power-status check and a start if needed** (see Parts 2.5 / 2B.5 / 3.5 / 3B.5).

**Test targets (all running before the test):**

| VMID | Name | Node | Role in test |
|------|------|------|--------------|
| 8701 | fsx-dev-scraper01 | fsx-dev-pve22 | single-VM lifecycle (Parts 2/2B) + bulk (Parts 3/3B) |
| 7303 | fsx-dev-workstation03 | fsx-dev-pve23 | bulk only (Parts 3/3B) — has a pre-existing snapshot `bf_modulejail` that must NOT be touched |
| 7305 | fsx-dev-workstation05 | fsx-dev-pve21 | bulk only (Parts 3/3B) |

> **`bf_modulejail` preservation invariant:** VM 7303 carries a pre-existing snapshot (`bf_modulejail`) that this runbook did not create. Every bulk create/delete loop targets snapshots by a `bulkdemo-*` / `bulkvn-*` name prefix, so `bf_modulejail` is never selected. The final-state checks (Steps 4.1) explicitly assert it remains. If it ever disappears, stop and escalate — do not attempt to recreate it.

> **Why these results are pre-filled:** The ✅ Result lines record the *expected* outcome. The validator executing this runbook should **overwrite each Result** with their own observed output (or mark ❌ on deviation). Result lines marked `*(to be observed)*` have not yet been run against the live cluster.

> **⚠️ `pve`-specific gotchas baked into every command below (these differ from the sibling `proxmox-snapshot-manager` tool):**
> 1. **Command path is `pve vm snapshot <verb>`** — NOT `pve snapshot <verb>`. The snapshot verbs' own `--help` examples show `pve snapshot ...`, but that path is **stale help text and fails** with "unknown command". Only `vm`, `cluster`, `node`, `container` are attached to the root. Always use `pve vm snapshot ...`.
> 2. **The snapshot-name flag is `--snapshot`** — NOT `--snapshot_name` / `--snap`. (And VM selection is `--vmname`, one word — there is no `--vm-name`.)
> 3. **Bulk is just multi-value `--vmid` / `--vmname`** — there is **no** `bulk` subcommand and **no** `--batch` flag for snapshot ops. Pass comma-separated values to the ordinary `create`/`rollback`/`delete` verbs (e.g. `--vmid 8701,7303,7305`). The two-VM/keyword/wildcard/range selectors (`running`, `stopped`, `all`, `72*`, `7201-7205`, `i`) also expand to bulk targets. `pve vm bulk ...` exists but is for **power/backup ops only** — not snapshots.
> 4. **Default concurrency is 2** (`MaxConcurrentSnapshots`). With 3 VMs targeted, the third VM completes shortly after the first two — this staggered completion is expected, not a hang. There is **no** `--workers` / `--concurrency` flag; raise it via config (`max_concurrent_snapshots`) if needed.
> 5. **Bulk output = per-VM lines + a summary block.** Each VM logs a `✅ VM <id> (<name>): ...` line; the run ends with a `BULK OPERATION SUMMARY` showing `Total Operations:`, `Successful: N (X.X%)`, `Failed:`. The sibling tool's summary looks similar but is not identical — do not grep the sibling's exact strings.
> 6. **`--all` on delete is a different (worse) path.** `pve vm snapshot delete --vmid ... --all` deletes *every* snapshot of each VM sequentially with **no concurrency and no summary block**, and would destroy `bf_modulejail` on 7303. **Never use `--all` in this runbook** — delete by explicit per-VM name via the loops in Steps 3.6 / 3B.6.

---

## Prerequisites

- **`pve` v1.5.0** installed — either from the GitHub release archive, via `go install github.com/yg-codes/proxmox/pve@v1.5.0`, or via mise (`go:github.com/yg-codes/proxmox/pve` = `v1.5.0`). See Part 1.
- An **API token** on the `fsx-dev` cluster with at least `PVEVMAdmin` on the target VM (and the VM's node). The token name is the plain label; the token value is the secret (or an `op://` reference).
- **1Password CLI** (`op` / `op.exe`) authenticated — only required if you pass `op://` references for credentials. The tool resolves any credential env var whose value starts with `op://` at startup. Plaintext credentials skip this.
- SSH as `root` to the cluster nodes — needed **only** for the post-rollback `qm start` (the `pve` snapshot verbs do **not** do power operations). Single-VM parts need only `fsx-dev-pve22`; the bulk parts need all three: `fsx-dev-pve21`, `fsx-dev-pve22`, `fsx-dev-pve23`.
- **Approval:** per project policy, snapshot operations on VMs other than **7303** require explicit approval. **8701 and 7305 are not on the no-approval list — obtain approval before executing the bulk parts (3 / 3B) against the live cluster.** (7303 itself needs no approval, but its pre-existing `bf_modulejail` snapshot must not be touched — see the preservation invariant above.)

---

## Part 0: Pre-checks

### Step 0.1: Confirm the target VMs are running and in one cluster

A single `PVE_HOST` resolves all VMs only if the nodes form one cluster. The bulk parts (3 / 3B) need all three VMs reachable.

**Command**:
```bash
ssh -o ConnectTimeout=5 root@fsx-dev-pve22.fsx.zone \
  "pvesh get /cluster/resources --type vm 2>/dev/null | grep -E ' 8701 | 7303 | 7305 '"
```

**Expected**: All three VMs appear with `running` status — 8701 (`fsx-dev-scraper01`) on pve22, 7303 (`fsx-dev-workstation03`) on pve23, 7305 (`fsx-dev-workstation05`) on pve21 — confirming one cluster.

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

## Part 3: Bulk Lifecycle by VMID — VMs 8701, 7303, 7305

Drives all four verbs against **three VMs on three different nodes** in a single cluster via comma-separated `--vmid`. This validates multi-target selection, the concurrent worker pool (default cap **2**), cross-node operation, and the `BULK OPERATION SUMMARY` output. Uses a distinct prefix (`bulkdemo`) so these never collide with Part 2's `demo-*` / Part 2B's `vndemo-*`.

> **Note on names:** generated snapshot names embed each VM's name and a per-VM timestamp (e.g. `bulkdemo-fsx-dev-workstation05-...`), so a single `--snapshot` value cannot match all three for rollback/delete. Capture each name from Step 3.2 and operate **per-VM by exact name** in 3.4 and 3.6 (the loops do this). Rollback takes exactly one `--snapshot` (String); delete takes one-or-more (StringSlice) — either way, per-VM loops are the clean approach.

### Step 3.1: Bulk list (read-only)

**Command**:
```bash
pve vm snapshot list --vmid 8701,7303,7305 2>&1 | grep -E 'VM [0-9]+:|bf_modulejail|No snapshots'
```

**Expected**: All three VMs listed `Status: 🟢 running`. VM 7303 shows the **pre-existing** `bf_modulejail` snapshot — leave it untouched. 8701 and 7305 show no snapshots (after Parts 2/2B cleaned up).

**Result**: ✅ *(to be observed)*

### Step 3.2: Bulk create

**Command**:
```bash
pve vm snapshot create --vmid 8701,7303,7305 --prefix bulkdemo -y 2>&1 | tail -20
```

**Expected**: Three `✅ VM <id> (<name>): Snapshot ... created successfully (<dur>s)` lines, then a `BULK OPERATION SUMMARY` block ending `Total Operations: 3`, `Successful: 3 (100.0%)`, `Failed: 0 (0.0%)`. With concurrency capped at 2, one VM finishes slightly after the other two. **Capture all three names** using:

```bash
declare -A SNAP
for vmid in 8701 7303 7305; do
  SNAP[$vmid]=$(pve vm snapshot list --vmid "$vmid" 2>&1 | grep -oP 'bulkdemo-\S+')
  echo "VM $vmid: ${SNAP[$vmid]}"
done
```

**Result**: ✅ *(to be observed — record the three captured snapshot names)*

### Step 3.3: Verify all three snapshots present

**Command**:
```bash
pve vm snapshot list --vmid 8701,7303,7305 2>&1 | grep -E 'VM [0-9]+:|bulkdemo|bf_modulejail'
```

**Expected**: One `bulkdemo-*` snapshot per VM (`VM State: ❌ Not included (disk only)`); 7303 also still shows `bf_modulejail` (the new snapshot is its child).

**Result**: ✅ *(to be observed)*

### Step 3.4: Bulk rollback (per-VM, by exact name)

Uses the `SNAP` map captured in Step 3.2. Per-VM loop because rollback's `--snapshot` takes a single name.

**Command**:
```bash
for vmid in 8701 7303 7305; do
  echo "=== rollback $vmid -> ${SNAP[$vmid]} ==="
  pve vm snapshot rollback --vmid "$vmid" --snapshot "${SNAP[$vmid]}" -y 2>&1 | tail -6
done
```

**Expected**: Each VM reports `✅ VM <id> ... rolled back to snapshot '<name>' successfully`. (Each single-VM rollback prints its own one-row summary.)

**Result**: ✅ *(to be observed)*

### Step 3.5: Post-rollback power check — start any stopped VM (MANDATORY)

Disk-only rollback leaves each VM **powered off**; required after every rollback.

**Command**:
```bash
declare -A NODE=( [8701]=fsx-dev-pve22 [7303]=fsx-dev-pve23 [7305]=fsx-dev-pve21 )
for vmid in 8701 7303 7305; do
  node="${NODE[$vmid]}.fsx.zone"
  st=$(ssh -o ConnectTimeout=5 root@"$node" "qm status $vmid")
  echo "VM $vmid @ ${NODE[$vmid]}: $st"
  echo "$st" | grep -q stopped && ssh -o ConnectTimeout=5 root@"$node" "qm start $vmid"
done
sleep 3
for vmid in 8701 7303 7305; do
  node="${NODE[$vmid]}.fsx.zone"
  echo "VM $vmid: $(ssh -o ConnectTimeout=5 root@"$node" "qm status $vmid")"
done
```

**Expected**: All three read `stopped` after rollback, then `running` after start. (VM 7303 may print a benign Proxmox EFI/secure-boot certificate **warning** on start — the start still succeeds.)

**Result**: ✅ *(to be observed)*

### Step 3.6: Bulk delete (per-VM, by exact name)

Reuses the `SNAP` map from Step 3.2. Per-VM loop — do **NOT** use `--all` (see gotcha 6: `--all` is sequential, summary-less, and would destroy `bf_modulejail`).

**Command**:
```bash
for vmid in 8701 7303 7305; do
  echo "=== delete $vmid -> ${SNAP[$vmid]} ==="
  pve vm snapshot delete --vmid "$vmid" --snapshot "${SNAP[$vmid]}" -y 2>&1 | tail -6
done
```

**Expected**: Each VM reports `✅ Snapshot '<name>' deleted successfully` for its `bulkdemo-*` snapshot.

**Result**: ✅ *(to be observed)*

### Step 3.7: Verify clean state

**Command**:
```bash
pve vm snapshot list --vmid 8701,7303,7305 2>&1 | grep -E 'VM [0-9]+:|bulkdemo|bf_modulejail|No snapshots'
```

**Expected**: No `bulkdemo-*` snapshots anywhere. 8701 and 7305 show `No snapshots`. 7303 shows **only** `bf_modulejail` (preserved). All three VMs `🟢 running`.

**Result**: ✅ *(to be observed)*

---

## Part 3B: Bulk Lifecycle by VM Name — VMs `fsx-dev-scraper01`, `fsx-dev-workstation03`, `fsx-dev-workstation05`

Mirrors Part 3's bulk lifecycle but drives every command with **`--vmname`** (comma-separated names) instead of `--vmid`, validating name resolution across all three nodes through the full create → rollback → delete cycle. Uses a distinct prefix (`bulkvn`) so these never collide with Part 3's `bulkdemo-*`. VM 7303's pre-existing `bf_modulejail` must remain untouched.

> **Note on names:** as in Part 3, generated names embed each VM's name and a per-VM timestamp, so rollback/delete operate **per-VM by exact name**. The capture and loops below resolve each VM by name.

### Step 3B.1: Bulk list by name (read-only)

**Command**:
```bash
pve vm snapshot list \
  --vmname fsx-dev-scraper01,fsx-dev-workstation03,fsx-dev-workstation05 \
  2>&1 | grep -E 'VM [0-9]+:|bf_modulejail|No snapshots'
```

**Expected**: All three VMs `Status: 🟢 running`; 7303 shows the pre-existing `bf_modulejail`. Confirms the three names resolve to VMIDs 8701, 7303, 7305.

**Result**: ✅ *(to be observed)*

### Step 3B.2: Bulk create by name

**Command**:
```bash
pve vm snapshot create \
  --vmname fsx-dev-scraper01,fsx-dev-workstation03,fsx-dev-workstation05 \
  --prefix bulkvn -y 2>&1 | tail -20
```

**Expected**: Three `✅ ... created successfully` lines + `BULK OPERATION SUMMARY` with `Total Operations: 3`, `Successful: 3 (100.0%)`. **Capture all three** (keyed by name):

```bash
declare -A VNSNAP
for name in fsx-dev-scraper01 fsx-dev-workstation03 fsx-dev-workstation05; do
  VNSNAP[$name]=$(pve vm snapshot list --vmname "$name" 2>&1 | grep -oP 'bulkvn-\S+')
  echo "$name: ${VNSNAP[$name]}"
done
```

**Result**: ✅ *(to be observed — record the three captured snapshot names)*

### Step 3B.3: Verify all three snapshots present (by name)

**Command**:
```bash
pve vm snapshot list \
  --vmname fsx-dev-scraper01,fsx-dev-workstation03,fsx-dev-workstation05 \
  2>&1 | grep -E 'VM [0-9]+:|bulkvn|bf_modulejail'
```

**Expected**: One `bulkvn-*` snapshot per VM (disk-only); 7303 still shows `bf_modulejail`.

**Result**: ✅ *(to be observed)*

### Step 3B.4: Bulk rollback by name (per-VM, by exact snapshot name)

Uses the `VNSNAP` map captured in Step 3B.2.

**Command**:
```bash
for name in fsx-dev-scraper01 fsx-dev-workstation03 fsx-dev-workstation05; do
  echo "=== rollback $name -> ${VNSNAP[$name]} ==="
  pve vm snapshot rollback --vmname "$name" --snapshot "${VNSNAP[$name]}" -y 2>&1 | tail -6
done
```

**Expected**: Each VM reports `✅ VM <id> ... rolled back to snapshot '<name>' successfully`.

**Result**: ✅ *(to be observed)*

### Step 3B.5: Post-rollback power check — start any stopped VM (MANDATORY)

**Command**:
```bash
declare -A NODE=( [fsx-dev-scraper01]=fsx-dev-pve22 [fsx-dev-workstation03]=fsx-dev-pve23 [fsx-dev-workstation05]=fsx-dev-pve21 )
declare -A VMID=( [fsx-dev-scraper01]=8701 [fsx-dev-workstation03]=7303 [fsx-dev-workstation05]=7305 )
for name in fsx-dev-scraper01 fsx-dev-workstation03 fsx-dev-workstation05; do
  node="${NODE[$name]}.fsx.zone"; id="${VMID[$name]}"
  st=$(ssh -o ConnectTimeout=5 root@"$node" "qm status $id")
  echo "$name ($id) @ ${NODE[$name]}: $st"
  echo "$st" | grep -q stopped && ssh -o ConnectTimeout=5 root@"$node" "qm start $id"
done
sleep 3
for name in fsx-dev-scraper01 fsx-dev-workstation03 fsx-dev-workstation05; do
  node="${NODE[$name]}.fsx.zone"; id="${VMID[$name]}"
  echo "$name: $(ssh -o ConnectTimeout=5 root@"$node" "qm status $id")"
done
```

**Expected**: All three read `stopped` after rollback, then `running` after start. (VM 7303 may emit the benign EFI cert warning on start.)

**Result**: ✅ *(to be observed)*

### Step 3B.6: Bulk delete by name (per-VM, by exact snapshot name)

Reuses the `VNSNAP` map from Step 3B.2. Per-VM loop — not `--all`.

**Command**:
```bash
for name in fsx-dev-scraper01 fsx-dev-workstation03 fsx-dev-workstation05; do
  echo "=== delete $name -> ${VNSNAP[$name]} ==="
  pve vm snapshot delete --vmname "$name" --snapshot "${VNSNAP[$name]}" -y 2>&1 | tail -6
done
```

**Expected**: Each VM reports `✅ Snapshot '<name>' deleted successfully` for its `bulkvn-*` snapshot.

**Result**: ✅ *(to be observed)*

### Step 3B.7: Verify clean state (by name)

**Command**:
```bash
pve vm snapshot list \
  --vmname fsx-dev-scraper01,fsx-dev-workstation03,fsx-dev-workstation05 \
  2>&1 | grep -E 'VM [0-9]+:|bulkvn|bf_modulejail|No snapshots'
```

**Expected**: No `bulkvn-*` snapshots anywhere. 8701 and 7305 show `No snapshots`. 7303 shows **only** `bf_modulejail` (preserved). All three VMs `🟢 running`.

**Result**: ✅ *(to be observed)*

---

## Part 4: Post-checks (final state verification)

### Step 4.1: Verify all test snapshots removed and VMs running

**Command**:
```bash
pve vm snapshot list --vmid 8701,7303,7305 2>&1 | grep -E 'VM [0-9]+:|demo-fsx-dev-scraper01|vndemo-fsx-dev-scraper01|bulkdemo|bulkvn|bf_modulejail|No snapshots'
for n in fsx-dev-pve21 fsx-dev-pve22 fsx-dev-pve23; do
  case $n in *pve21) id=7305;; *pve22) id=8701;; *pve23) id=7303;; esac
  echo "$n / $id: $(ssh -o ConnectTimeout=5 root@$n.fsx.zone "qm status $id")"
done
```

**Expected**: No `demo-*` / `vndemo-*` / `bulkdemo-*` / `bulkvn-*` snapshots anywhere. 8701 and 7305 show `No snapshots`. 7303 shows **only** `bf_modulejail` (pre-existing, preserved). All three VMs `running`.

**Result**: ✅ *(to be observed)*

### Step 4.2: Clean up the scratch directory (Option A install only)

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
| A `demo-*` / `vndemo-*` snapshot left behind on 8701 | `pve vm snapshot delete --vmid 8701 --snapshot '<exact-name>' -y` |
| A `bulkdemo-*` / `bulkvn-*` snapshot left behind on 8701 / 7303 / 7305 | `pve vm snapshot delete --vmid <id> --snapshot '<exact-name>' -y` (per-VM) — **never** `--all` |
| A VM left `stopped` after rollback | `ssh root@<node>.fsx.zone "qm start <vmid>"`, then confirm `qm status <vmid>` → running (node map: 8701→pve22, 7303→pve23, 7305→pve21) |
| `bf_modulejail` on 7303 accidentally removed | **Stop.** This snapshot is not created by this runbook; recover from Proxmox backup (PBS) or escalate. Do not attempt to recreate it |
| Bulk create shows `Successful: 2 (66.7%)` / one `Failed:` | One VM errored mid-batch (e.g. node unreachable, VM locked). The summary's `FAILED OPERATIONS:` block names the VM and cause; fix and re-run create for that single VM |
| Third bulk VM appears to "hang" | It is not hung — concurrency cap is 2, so the 3rd VM starts only after one of the first two finishes. Wait for the `BULK OPERATION SUMMARY` block |
| Credentials fail (`op` not signed in / 403) | Re-check 1Password auth (Step 0.2) and the token role on Proxmox; do **not** proceed past Step 2.1 until `list` works |
| `pve snapshot <verb>` fails with "unknown command" | You hit the stale-help-text gotcha — use **`pve vm snapshot <verb>`** instead |
| `--snapshot_name` rejected | Wrong flag — use **`--snapshot`** |
| Generated name capture returns empty | The `grep -oP` pattern expects `<prefix>-<vmname>-<TS>`; re-run `pve vm snapshot list --vmid <id>` and copy the exact name manually |

This runbook is non-destructive by design: every snapshot it creates it also deletes, and rollbacks target snapshots taken seconds earlier (no real disk-state change). The only irreversible operation would be deleting a snapshot **not** created here — hence the explicit "do not touch `bf_modulejail`" and "never `--all`" callouts.

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
| 3.1 | Bulk list (3 VMs) | ⬜ |
| 3.2 | Bulk create (3) | ⬜ |
| 3.3 | Verify 3 snapshots | ⬜ |
| 3.4 | Bulk rollback (3) | ⬜ |
| 3.5 | Post-rollback start (3) | ⬜ |
| 3.6 | Bulk delete (3) | ⬜ |
| 3.7 | Verify clean (bulk) | ⬜ |
| 3B.1 | Bulk list by `--vmname` | ⬜ |
| 3B.2 | Bulk create (3) by `--vmname` | ⬜ |
| 3B.3 | Verify 3 snapshots (name) | ⬜ |
| 3B.4 | Bulk rollback (3) by `--vmname` | ⬜ |
| 3B.5 | Post-rollback start (3) | ⬜ |
| 3B.6 | Bulk delete (3) by `--vmname` | ⬜ |
| 3B.7 | Verify clean (name) | ⬜ |
| 4.1 | Final state clean (3 VMs, `bf_modulejail` intact) | ⬜ |
| 4.2 | Scratch dir removed | ⬜ |

**Escalation**: If any step fails and the Cleanup & Recovery Plan does not resolve it, stop and seek approval before retrying destructive steps — VMs 8701 and 7305 are not on the project's no-approval testing list. Do not delete snapshots you did not create, and never use `--all` on delete.
