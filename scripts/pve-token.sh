#!/bin/bash
#
# pve-token.sh — Proxmox VE API user/token manager.
#
# One script covering the full token lifecycle. The --action flag selects the
# scope so there is no silent over-reach (e.g. revoking a token never deletes
# the user, adding a token never recreates the user):
#
#   create        Q3  Create a user (if missing) AND a token, assign role, verify
#   add-token     Q2  Add a token to an EXISTING user (user is not created)
#   revoke-token  Q1  Revoke ONE token (user and other tokens are preserved)
#   remove        --  Delete the user (and all its tokens) entirely
#   list          --  List a user's tokens (read-only)
#
# Usage:
#   ./pve-token.sh [OPTIONS] --action <ACTION> [POSITIONAL...]
#
# Options:
#   -a, --action <ACTION>   create | add-token | revoke-token | remove | list
#                           (default: create, for backward compatibility)
#   -n, --node <node>       Proxmox node (SSH target). Required unless --local
#                           or --nodes auto-discovery succeeds.
#   --nodes <n1,n2,...>     Comma-separated node list (uses the first as target;
#                           user/token objects are cluster-wide).
#   -u, --user <name>       Username, or "user@realm" (required for all actions)
#   -t, --token <name>      Token name (required for create/add-token/revoke-token)
#   --realm <realm>         Auth realm: pam, pve, ldap (default: pve)
#                           An explicit --realm overrides any realm embedded in --user.
#   --role <role>           Role to assign at / (default: PVEVMAdmin)
#   --local                 Run directly on this host (no SSH) — for use ON a node
#   --dry-run               Show commands without executing
#   -h, --help              Show this help
#
# Positional shortcut (no --action needed): the common "create user + token"
# path keeps the fast-script ergonomics:
#   ./pve-token.sh <node> [user] [token]        # == --action create
#   ./pve-token.sh --local [user] [token]       # == --action create, local
#
# Requirements:
#   - SSH root access to a Proxmox node (or run with --local on a node)
#   - jq installed locally (reliable token-value parsing + node discovery)
#
# Examples:
#   # Q3 — create user + token
#   ./pve-token.sh pve1                                 # pve-admin@pam + admin-token
#   ./pve-token.sh pve1 automation prod-token           # custom names
#   ./pve-token.sh --action create --node pve1 --user automation
#
#   # Q2 — add a SECOND token to an existing user
#   ./pve-token.sh --action add-token --node pve1 --user automation --token ci
#
#   # Q1 — revoke ONE token (user kept)
#   ./pve-token.sh --action revoke-token --node pve1 --user automation --token ci
#
#   # Remove the whole user and all its tokens
#   ./pve-token.sh --action remove --node pve1 --user automation
#
#   # List a user's tokens
#   ./pve-token.sh --action list --node pve1 --user automation
#

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ---- defaults ---------------------------------------------------------------

ACTION=""
USER_NAME="pve-admin"
TOKEN_NAME="admin-token"
REALM="pve"
ROLE="PVEVMAdmin"
REALM_EXPLICIT=false
USER_INPUT=false
TOKEN_INPUT=false
NODE=""
NODES=""
LOCAL_MODE=false
DRY_RUN=false
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes"

print_help() {
  sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

# ---- parse flags, collect positionals separately ----------------------------
# Positional args drive the backward-compatible "create" shortcut. Flags can
# appear in any order.

POSITIONAL=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -a|--action)        [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; ACTION="$2"; shift 2 ;;
    -n|--node)          [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; NODE="$2"; shift 2 ;;
    --nodes)            [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; NODES="$2"; shift 2 ;;
    -u|--user|--user-name) [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; USER_NAME="$2"; USER_INPUT=true; shift 2 ;;
    -t|--token|--token-name) [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; TOKEN_NAME="$2"; TOKEN_INPUT=true; shift 2 ;;
    --realm)            [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; REALM="$2"; REALM_EXPLICIT=true; shift 2 ;;
    --role)             [[ $# -ge 2 ]] || { echo -e "${RED}Error: $1 requires an argument${NC}" >&2; exit 2; }; ROLE="$2"; shift 2 ;;
    --local)            LOCAL_MODE=true; shift ;;
    --dry-run)          DRY_RUN=true; shift ;;
    -h|--help)          print_help 0 ;;
    --)                 shift; while [[ $# -gt 0 ]]; do POSITIONAL+=("$1"); shift; done ;;
    -*)                 echo -e "${RED}Unknown option: $1${NC}" >&2; print_help 1 ;;
    *)                  POSITIONAL+=("$1"); shift ;;
  esac
done

# Backward-compatible positional shortcut: <node> [user] [token] => create.
# Flip USER_INPUT/TOKEN_INPUT only when the positional is actually present, so a
# bare <node> (no user) still fails the required-arg check rather than silently
# targeting the default user.
if [[ -z "$ACTION" ]]; then
  if $LOCAL_MODE; then
    # --local with positionals: [user] [token]
    [[ -n "${POSITIONAL[0]:-}" ]] && { USER_NAME="${POSITIONAL[0]}"; USER_INPUT=true; }
    [[ -n "${POSITIONAL[1]:-}" ]] && { TOKEN_NAME="${POSITIONAL[1]}"; TOKEN_INPUT=true; }
  elif [[ ${#POSITIONAL[@]} -ge 1 ]]; then
    NODE="${POSITIONAL[0]}"
    [[ -n "${POSITIONAL[1]:-}" ]] && { USER_NAME="${POSITIONAL[1]}"; USER_INPUT=true; }
    [[ -n "${POSITIONAL[2]:-}" ]] && { TOKEN_NAME="${POSITIONAL[2]}"; TOKEN_INPUT=true; }
  fi
  ACTION="create"
fi

# Resolve node from positionals for explicit --action calls too (so
# `--action list pve1` works the same as `--action list --node pve1`).
# NOTE: $LOCAL_MODE is a boolean string ("true"/"false"); test it as a command,
# not with -z (a non-empty "false" would wrongly short-circuit -z).
if [[ -z "$NODE" ]] && ! $LOCAL_MODE && [[ ${#POSITIONAL[@]} -ge 1 && "$ACTION" != "create" ]]; then
  NODE="${POSITIONAL[0]}"
fi

# Normalize user input: accept "user@realm" via --user (or positional), and
# split the realm out of it. Precedence: explicit --realm > embedded @realm >
# default. Runs once, after both parse paths converge, before USER_ID is built.
if [[ "$USER_NAME" == *@* ]]; then
  embedded_realm="${USER_NAME##*@}"     # part after the last @
  USER_NAME="${USER_NAME%@*}"           # part before the last @
  if ! $REALM_EXPLICIT; then
    REALM="$embedded_realm"
  fi
fi

case "$ACTION" in
  create|add-token|revoke-token|remove|list) : ;;
  *) echo -e "${RED}Error: unknown --action '$ACTION'${NC}" >&2; print_help 1 ;;
esac

# Require --user for every action — there is no sensible default target for a
# token operation, so a missing flag must fail loudly, not act on pve-admin.
if ! $USER_INPUT; then
  echo -e "${RED}Error: --user is required for action '$ACTION'.${NC}" >&2
  exit 1
fi
# Require --token for actions that name/create a specific token.
case "$ACTION" in
  create|add-token|revoke-token)
    if ! $TOKEN_INPUT; then
      echo -e "${RED}Error: --token is required for action '$ACTION'.${NC}" >&2
      exit 1
    fi
    ;;
esac

USER_ID="${USER_NAME}@${REALM}"
TOKEN_ID="${USER_ID}!${TOKEN_NAME}"

# ---- helpers ----------------------------------------------------------------

# Resolve the target node. Explicit --node wins; else --nodes first entry;
# else auto-discover via the first reachable common node name.
resolve_node() {
  if [[ -n "$NODE" ]]; then
    echo "$NODE"
    return
  fi
  if [[ -n "$NODES" ]]; then
    echo "${NODES%%,*}"
    return
  fi
  if $LOCAL_MODE; then
    echo "localhost"
    return
  fi
  for n in pve pve1 pve1.local proxmox; do
    if ssh $SSH_OPTS "$n" "echo ok" &>/dev/null; then
      echo "$n"
      return
    fi
  done
  echo -e "${RED}Error: could not determine node. Use --node, --nodes, or --local.${NC}" >&2
  exit 1
}

# Run a command locally (--local) or over SSH. $1 = shell command string.
run_cmd() {
  local cmd="$1"
  if $LOCAL_MODE; then
    bash -c "$cmd"
  else
    ssh $SSH_OPTS "root@${NODE}" "$cmd"
  fi
}

# Run a command with a labeled status line (used for visible provisioning steps).
run_labeled() {
  local node="$1"; local cmd="$2"; local desc="$3"
  echo -e "${YELLOW}[$node]${NC} $desc"
  if $DRY_RUN; then
    echo -e "  ${BLUE}[DRY-RUN]${NC} $cmd"
    return 0
  fi
  if run_cmd "$cmd" 2>&1; then
    echo -e "  ${GREEN}✓ Success${NC}"
    return 0
  else
    echo -e "  ${RED}✗ Failed${NC}"
    return 1
  fi
}

# Extract the token secret from `pveum user token add` output. JSON first
# (version-stable), grep fallbacks for older PVE.
extract_token_value() {
  local output="$1"
  local value
  if value=$(echo "$output" | jq -r '.value // empty' 2>/dev/null) && [[ -n "$value" ]]; then
    echo "$value"; return 0
  fi
  echo "$output" \
    | grep -oP 'value\s*│\s*\K\S+' 2>/dev/null \
    || echo "$output" | grep -oP '^\s*value\s*:?\s*\K[0-9a-f-]+' 2>/dev/null \
    || echo "$output" | grep -oP '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' 2>/dev/null \
    || true
}

# Print the token result block + ready-to-use exports. $1 = resolved secret.
print_token_result() {
  local token_value="$1"
  echo -e "\n${GREEN}================================================${NC}"
  echo -e "${GREEN}  API Token Created Successfully${NC}"
  echo -e "${GREEN}================================================${NC}"
  echo ""
  echo -e "  Node:        ${NODE}"
  echo -e "  User:        ${USER_ID}"
  echo -e "  Token:       ${TOKEN_NAME}"
  echo -e "  Token ID:    ${TOKEN_ID}"
  echo -e "  Token Value: ${YELLOW}${token_value}${NC}"
  echo -e "  Role:        ${ROLE}"
  echo ""
  echo -e "${RED}Save the token value now - it cannot be retrieved later!${NC}"
  echo ""
  echo -e "${BLUE}--- Shell exports (add to ~/.bashrc) ---${NC}"
  cat <<EOF
export PVE_HOST="${NODE}"
export PVE_USER="${USER_ID}"
export PVE_TOKEN_NAME="${TOKEN_NAME}"
export PVE_TOKEN_VALUE="${token_value}"
EOF
  echo ""
  echo -e "${BLUE}--- .env file format ---${NC}"
  cat <<EOF
PVE_HOST=${NODE}
PVE_USER=${USER_ID}
PVE_TOKEN_NAME=${TOKEN_NAME}
PVE_TOKEN_VALUE=${token_value}
EOF
  echo ""
  echo -e "${BLUE}--- Quick test ---${NC}"
  echo "  pve vm list"
  echo "  pve node list"
}

# ---- action: create (Q3) ----------------------------------------------------
do_create() {
  local node="$1"
  echo -e "${BLUE}Creating API token on ${node}...${NC}"

  # User: create only if missing.
  if run_cmd "pveum user list" 2>/dev/null | grep -q "${USER_ID}"; then
    echo -e "${YELLOW}User ${USER_ID} already exists, skipping creation${NC}"
  else
    run_labeled "$node" "pveum user add ${USER_ID} --comment 'PVE CLI admin user'" \
      "Creating user ${USER_ID}" || true
  fi

  do_add_token "$node"   # token + role assignment + result block

  # Verify
  echo -e "\n${BLUE}=== Verifying Setup ===${NC}"
  run_labeled "$node" "pveum user list | grep -q ${USER_ID}" "User ${USER_ID} exists"
  run_labeled "$node" "pveum user token list ${USER_ID} | grep -q ${TOKEN_NAME}" "Token ${TOKEN_NAME} exists"
  run_labeled "$node" "pveum acl list | grep -q ${TOKEN_ID}" "ACL entry for token exists"
}

# ---- action: add-token (Q2) -------------------------------------------------
# Adds a token to an EXISTING user. Errors clearly if the user does not exist.
do_add_token() {
  local node="$1"

  # Guard: refuse to create the user in this action.
  if ! $DRY_RUN; then
    if ! run_cmd "pveum user list" 2>/dev/null | grep -q "${USER_ID}"; then
      echo -e "${RED}Error: user ${USER_ID} does not exist. Use --action create to provision it first.${NC}" >&2
      exit 1
    fi
  fi

  echo -e "\n${BLUE}=== Creating API Token for ${USER_ID} ===${NC}"

  local output token_value
  if $DRY_RUN; then
    run_labeled "$node" \
      "pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0" \
      "Adding token ${TOKEN_NAME}"
    echo -e "  ${YELLOW}[DRY-RUN] token value would be shown here${NC}"
    return 0
  fi

  output=$(run_cmd "pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 --output-format json 2>/dev/null \
    || pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 2>&1")
  token_value=$(extract_token_value "$output")

  if [[ -z "$token_value" ]]; then
    echo -e "${RED}Failed to parse token value. Raw output:${NC}"
    echo "$output"
    echo ""
    echo -e "${YELLOW}If the token already exists, revoke it first:${NC}"
    echo "  $0 --action revoke-token --node ${node} --user ${USER_NAME} --token ${TOKEN_NAME}"
    exit 1
  fi

  # Assign role to the token (and the user, for ops that check user perms).
  run_labeled "$node" "pveum aclmod / -tokens '${TOKEN_ID}' -roles ${ROLE}" \
    "Assigning ${ROLE} to token at /"
  run_labeled "$node" "pveum aclmod / -users '${USER_ID}' -roles ${ROLE}" \
    "Assigning ${ROLE} to user at /"

  print_token_result "$token_value"
}

# ---- action: revoke-token (Q1) ----------------------------------------------
# Removes ONE token. The user and all OTHER tokens are preserved.
do_revoke_token() {
  local node="$1"
  echo -e "${RED}Revoking token ${TOKEN_NAME} for ${USER_ID} on ${node}...${NC}"
  echo -e "${YELLOW}(the user and any other tokens are preserved)${NC}"
  run_labeled "$node" "pveum user token remove ${USER_ID} ${TOKEN_NAME}" \
    "Revoking token ${TOKEN_NAME}" || true
  echo -e "\n${GREEN}Done.${NC}"
}

# ---- action: remove ---------------------------------------------------------
# Deletes the user entirely (tokens are removed with it).
do_remove() {
  local node="$1"
  echo -e "${RED}Removing user ${USER_ID} (and all its tokens) on ${node}...${NC}"
  run_labeled "$node" "pveum user delete ${USER_ID}" "Deleting user ${USER_ID}" || true
  echo -e "\n${GREEN}Done.${NC}"
}

# ---- action: list -----------------------------------------------------------
do_list() {
  local node="$1"
  echo -e "${BLUE}Tokens for ${USER_ID} on ${node}:${NC}"
  run_cmd "pveum user token list ${USER_ID}" || true
}

# ---- main -------------------------------------------------------------------

NODE="$(resolve_node)"

echo -e "${BLUE}==============================================${NC}"
echo -e "${BLUE}  pve-token  | action: ${ACTION}  | node: ${NODE}${NC}"
echo -e "${BLUE}==============================================${NC}"

case "$ACTION" in
  create)       do_create       "$NODE" ;;
  add-token)    do_add_token    "$NODE" ;;
  revoke-token) do_revoke_token "$NODE" ;;
  remove)       do_remove       "$NODE" ;;
  list)         do_list         "$NODE" ;;
esac
