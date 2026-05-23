#!/bin/bash
#
# Fast API Token Generator for Proxmox VE
# Creates a user + token with Administrator role in one shot
#
# Usage:
#   ./create-api-token.sh <node>
#   ./create-api-token.sh <node> [user] [token-name]
#   ./create-api-token.sh --local              # Run directly on Proxmox node
#   ./create-api-token.sh <node> --remove      # Remove user and token
#
# Examples:
#   ./create-api-token.sh pve1                         # pve-admin@pam + admin-token
#   ./create-api-token.sh pve1 automation api-token    # automation@pam + api-token
#   ./create-api-token.sh --local                      # Run on node itself
#   ./create-api-token.sh pve1 --remove                # Cleanup
#

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
NODE="${1:?Usage: $0 <node> [user] [token-name]}"
USER_NAME="${2:-pve-admin}"
TOKEN_NAME="${3:-admin-token}"
REALM="pam"
ROLE="Administrator"
USER_ID="${USER_NAME}@${REALM}"
TOKEN_ID="${USER_ID}!${TOKEN_NAME}"

# Detect --local or --remove in any position
LOCAL_MODE=false
REMOVE_MODE=false
for arg in "$@"; do
  case "$arg" in
    --local)  LOCAL_MODE=true; NODE="localhost" ;;
    --remove) REMOVE_MODE=true ;;
  esac
done

# Run command locally or via SSH
run_cmd() {
  if $LOCAL_MODE; then
    eval "$1"
  else
    ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o BatchMode=yes "root@${NODE}" "$1"
  fi
}

# Remove mode
if $REMOVE_MODE; then
  echo -e "${RED}Removing ${USER_ID} and token ${TOKEN_NAME}...${NC}"
  run_cmd "pveum user token remove ${USER_ID} ${TOKEN_NAME} 2>/dev/null || true"
  run_cmd "pveum user delete ${USER_ID} 2>/dev/null || true"
  echo -e "${GREEN}Done. User and token removed.${NC}"
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
echo ""
TOKEN_OUTPUT=$(run_cmd "pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0" 2>&1)

# Parse token value from pveum output
# Output format varies: try common patterns
TOKEN_VALUE=$(echo "$TOKEN_OUTPUT" | grep -oP 'value\s*│\s*\K\S+' 2>/dev/null \
  || echo "$TOKEN_OUTPUT" | grep -oP 'value[:\s]+\K[0-9a-f-]+' 2>/dev/null \
  || echo "$TOKEN_OUTPUT" | grep -oP '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' 2>/dev/null \
  || echo "")

if [[ -z "$TOKEN_VALUE" ]]; then
  echo -e "${RED}Failed to parse token value. Raw output:${NC}"
  echo "$TOKEN_OUTPUT"
  echo ""
  echo -e "${YELLOW}If token already exists, remove it first:${NC}"
  echo "  $0 ${NODE} --remove"
  exit 1
fi

# Step 3: Assign Administrator role at root path
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
