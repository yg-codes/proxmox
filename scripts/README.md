# scripts/

Operator-run helper scripts for provisioning Proxmox API credentials and
validating the `pve` CLI's 1Password integration. These are **not** part of
`make test` / CI — they touch live Proxmox nodes and require real access.

## Scripts

| Script | Purpose |
|---|---|
| [`create-api-token.sh`](create-api-token.sh) | Fast, non-interactive one-shot user + API token creation |
| [`setup-pve-cli-user.sh`](setup-pve-cli-user.sh) | Interactive full setup: user, token, role, node discovery, verification, uninstall |
| [`test-1password-integration.sh`](test-1password-integration.sh) | Manual end-to-end test of `op://` credential resolution in the `pve` binary |

## Requirements

- **Run from a workstation** with SSH (`root@`) access to at least one Proxmox node.
- `jq` installed locally (used for reliable token-value parsing and node discovery).
- `openssl` for random password generation (interactive setup only).
- For 1Password testing: the 1Password CLI (`op` / `op.exe`), signed in, plus a
  vault item holding the Proxmox credentials.

## How-to

### Quick token creation — `create-api-token.sh`

Creates `pve-admin@pam` + `admin-token` (defaults) with the `PVEVMAdmin` role,
prints ready-to-use shell exports, and exits.

```bash
# Defaults: pve-admin@pam + admin-token on pve1
./scripts/create-api-token.sh pve1

# Custom user and token name
./scripts/create-api-token.sh pve1 automation api-token

# Run directly on a Proxmox node (no SSH)
./scripts/create-api-token.sh --local

# Remove the user and token
./scripts/create-api-token.sh pve1 --remove
./scripts/create-api-token.sh pve1 --remove automation api-token
```

> The token secret is shown **once** at creation. Save the printed export lines
> to `~/.bashrc` immediately — it cannot be retrieved later.

### Full interactive setup — `setup-pve-cli-user.sh`

Finer control than the fast script: realm selection, node auto-discovery,
verification steps, dry-run, and uninstall.

```bash
# Default user/token, auto-discover nodes
./scripts/setup-pve-cli-user.sh

# Specify nodes explicitly
./scripts/setup-pve-cli-user.sh --nodes pve1,pve2

# Custom user, token, realm, and role
./scripts/setup-pve-cli-user.sh --user-name automation --token-name prod-token \
    --realm pam --role PVEVMAdmin

# Preview commands without executing
./scripts/setup-pve-cli-user.sh --dry-run

# Remove the user and token
./scripts/setup-pve-cli-user.sh --uninstall
```

**Options:**

| Flag | Default | Description |
|---|---|---|
| `--user-name` | `pve-cli` | Username to create |
| `--token-name` | `cli-token` | API token name |
| `--realm` | `pam` | Auth realm: `pam`, `pve`, `ldap` |
| `--role` | `PVEVMAdmin` | Role to assign at `/` |
| `--nodes` | auto-discover | Comma-separated node list (`--node` also accepted) |
| `--dry-run` | off | Print commands without executing |
| `--uninstall` | off | Remove the user and token |

### Test 1Password integration — `test-1password-integration.sh`

Validates `op://` credential resolution end-to-end: all-op creds, mixed
plain + op creds, a bogus field label, and the signed-out state. Read-only
(runs only `pve node list`); never prints or stores resolved secrets.

```bash
./scripts/test-1password-integration.sh "op://SRE/pve-test"
```

The prefix is the common vault-item path; the script appends the field labels
`/host`, `/user`, `/token_name`, `/token_value`.

**Environment overrides:**

| Var | Description |
|---|---|
| `PVE_BIN` | Path to the `pve` binary (default: `./build/pve`, then `pve` on `PATH`) |
| `SKIP_NOTSIGNED=1` | Skip the subtest that signs you out of 1Password |

Exit code `0` = all assertions passed.

## Reference

### Which script to use?

- **`create-api-token.sh`** — automation / quick start. Non-interactive,
  positional args, fast. Use when you know the node and want defaults.
- **`setup-pve-cli-user.sh`** — first-time interactive setup or when you need
  realm/role/discovery control, verification, or a clean uninstall path.
- **`test-1password-integration.sh`** — only after configuring `op://` refs;
  confirms the `pve` binary resolves them correctly before relying on them.

### How tokens are created

Both provisioning scripts use `pveum user token add ... --privsep 0`
(privilege separation off — the token inherits the user's permissions), then
assign the role at the cluster root (`/`). Token value is parsed via
`--output-format json` + `jq`, with `grep` fallbacks for older Proxmox
releases that don't emit JSON for this subcommand.

### Multi-node SSH (removed)

A `pve-ssh-exec.sh` multi-node SSH runner previously lived here. It has been
removed — it duplicated [`parallel-ssh`](https://linux.die.net/man/1/parallel-ssh)
and carried an `eval`-based injection risk. For running commands across nodes
from the local workstation, use `parallel-ssh` directly:

```bash
parallel-ssh -H "pve1" -H "pve2" -H "pve3" -i "pvecm status"
```
