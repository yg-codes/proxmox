#!/bin/bash
#
# Fast API Token Generator for Proxmox VE
# Creates a user + token with the chosen role in one shot
#
# Usage:
#   ./create-api-token.sh <node> [user] [token-name]
#   ./create-api-token.sh --local [user] [token-name]   # Run directly on Proxmox node
#   ./create-api-token.sh <node> --remove [user] [token-name]
#
# Examples:
#   ./create-api-token.sh pve1                         # pve-admin@pam + admin-token
#   ./create-api-token.sh pve1 automation api-token    # automation@pam + api-token
#   ./create-api-token.sh --local                      # Run on node itself
#   ./create-api-token.sh pve1 --remove                # Cleanup (defaults)
#   ./create-api-token.sh pve1 --remove automation api-token
#

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
USER_NAME="pve-admin"
TOKEN_NAME="admin-token"
REALM="pam"
ROLE="PVEVMAdmin"
NODE=""
LOCAL_MODE=false
REMOVE_MODE=false

print_usage() {
  sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# //; s/^#//'
  exit "${1:-0}"
}

# Separate flags from positional args so flags work in any position
# (fixes the old bug where `--remove`/`--local` as $1 was treated as NODE).
POSITIONAL=()
for arg in "$@"; do
  case "$arg" in
    --local)        LOCAL_MODE=true ;;
    --remove)       REMOVE_MODE=true ;;
    -h|--help)      print_usage 0 ;;
    --user)         : ;;   # consumed below
    *)              POSITIONAL+=("$arg") ;;
  esac
done

# Positional: [node] [user] [token-name]
# In --local mode the node is implicit (localhost), so positionals are [user] [token-name].
if $LOCAL_MODE; then
  NODE="localhost"
  USER_NAME="${POSITIONAL[0]:-$USER_NAME}"
  TOKEN_NAME="${POSITIONAL[1]:-$TOKEN_NAME}"
else
  if [[ ${#POSITIONAL[@]} -eq 0 ]]; then
    echo -e "${RED}Error: <node> is required (or use --local to run on a Proxmox node)${NC}" >&2
    print_usage 1
  fi
  NODE="${POSITIONAL[0]}"
  USER_NAME="${POSITIONAL[1]:-$USER_NAME}"
  TOKEN_NAME="${POSITIONAL[2]:-$TOKEN_NAME}"
fi

USER_ID="${USER_NAME}@${REALM}"
TOKEN_ID="${USER_ID}!${TOKEN_NAME}"

# Run command locally or via SSH
run_cmd() {
  if $LOCAL_MODE; then
    eval "$1"
  else
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes "root@${NODE}" "$1"
  fi
}

# Extract the token secret value from `pveum user token add` output.
# Prefer JSON output (structured, version-stable); fall back to grep for older
# PVE releases that don't emit JSON for this subcommand.
extract_token_value() {
  local output="$1"
  local value
  # JSON path: {"value": "..."}
  if value=$(echo "$output" | jq -r '.value // empty' 2>/dev/null) && [[ -n "$value" ]]; then
    echo "$value"
    return 0
  fi
  # Human-readable fallbacks (box-drawing table OR "value:" key).
  echo "$output" \
    | grep -oP 'value\s*│\s*\K\S+' 2>/dev/null \
    || echo "$output" | grep -oP '^\s*value\s*:?\s*\K[0-9a-f-]+' 2>/dev/null \
    || echo "$output" | grep -oP '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' 2>/dev/null \
    || true
}

# Remove mode
if $REMOVE_MODE; then
  echo -e "${RED}Removing token ${TOKEN_NAME} and user ${USER_ID} on ${NODE}...${NC}"
  run_cmd "pveum user token remove ${USER_ID} ${TOKEN_NAME} 2>/dev/null || true"
  run_cmd "pveum user delete ${USER_ID} 2>/dev/null || true"
  echo -e "${GREEN}Done. Token and user removed.${NC}"
  exit 0
fi

echo -e "${BLUE}Creating API token on ${NODE}...${NC}"
echo ""

# Step 1: Create user (ignore if exists)
if run_cmd "pveum user list" 2>/dev/null | grep -q "${USER_ID}"; then
  echo -e "${YELLOW}User ${USER_ID} already exists, skipping creation${NC}"
else
  run_cmd "pveum user add ${USER_ID} --comment 'PVE CLI admin user'"
  echo -e "${GREEN}Created user: ${USER_ID}${NC}"
fi

# Step 2: Create token (--privsep 0 = inherit user permissions)
# Use JSON output when supported for reliable parsing.
echo ""
TOKEN_OUTPUT=$(run_cmd "pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 --output-format json 2>/dev/null \
  || pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 2>&1")

TOKEN_VALUE=$(extract_token_value "$TOKEN_OUTPUT")

if [[ -z "$TOKEN_VALUE" ]]; then
  echo -e "${RED}Failed to parse token value. Raw output:${NC}"
  echo "$TOKEN_OUTPUT"
  echo ""
  echo -e "${YELLOW}If the token already exists, remove it first:${NC}"
  echo "  $0 ${NODE} --remove ${USER_NAME} ${TOKEN_NAME}"
  exit 1
fi

# Step 3: Assign role at root path
run_cmd "pveum aclmod / -tokens '${TOKEN_ID}' -roles ${ROLE}"
run_cmd "pveum aclmod / -users '${USER_ID}' -roles ${ROLE}"
echo -e "${GREEN}Assigned role: ${ROLE} at /${NC}"

# Step 4: Verify
echo ""
echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}  API Token Created Successfully${NC}"
echo -e "${GREEN}================================================${NC}"
echo ""
echo -e "  Node:        ${NODE}"
echo -e "  User:        ${USER_ID}"
echo -e "  Token:       ${TOKEN_NAME}"
echo -e "  Token ID:    ${TOKEN_ID}"
echo -e "  Token Value: ${YELLOW}${TOKEN_VALUE}${NC}"
echo -e "  Role:        ${ROLE}"
echo -e "  Path:        / (full access)"
echo ""
echo -e "${RED}Save the token value now - it cannot be retrieved later!${NC}"
echo ""

# Step 5: Output ready-to-use exports
echo -e "${BLUE}--- Shell exports (add to ~/.bashrc) ---${NC}"
cat <<EOF
export PVE_HOST="${NODE}"
export PVE_USER="${USER_ID}"
export PVE_TOKEN_NAME="${TOKEN_NAME}"
export PVE_TOKEN_VALUE="${TOKEN_VALUE}"
EOF

echo ""
echo -e "${BLUE}--- .env file format ---${NC}"
cat <<EOF
PVE_HOST=${NODE}
PVE_USER=${USER_ID}
PVE_TOKEN_NAME=${TOKEN_NAME}
PVE_TOKEN_VALUE=${TOKEN_VALUE}
EOF

echo ""
echo -e "${BLUE}--- Quick test ---${NC}"
echo "  pve vm list"
echo "  pve node list"
