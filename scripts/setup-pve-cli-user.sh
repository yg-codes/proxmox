#!/bin/bash
#
# Setup PVE CLI User and API Token
# Creates a dedicated user and token with proper permissions for pve CLI tools
#
# Usage:
#   ./setup-pve-cli-user.sh [OPTIONS]
#
# Options:
#   --user-name      Username to create (default: pve-cli)
#   --token-name     API token name (default: cli-token)
#   --realm          Authentication realm: pam, pve, ldap (default: pam)
#   --role           Role to assign (default: PVEVMAdmin)
#   --nodes          Comma-separated list of nodes (default: auto-discover)
#   --dry-run        Show commands without executing
#   --uninstall      Remove the user and token
#   -h, --help       Show this help
#
# Requirements:
#   - SSH root access to Proxmox nodes
#   - jq installed locally (for reliable token-value parsing)
#
# Examples:
#   ./setup-pve-cli-user.sh --nodes pve1,pve2
#   ./setup-pve-cli-user.sh --user-name automation --token-name prod-token
#   ./setup-pve-cli-user.sh --dry-run
#   ./setup-pve-cli-user.sh --uninstall
#

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
USER_NAME="${USER_NAME:-pve-cli}"
TOKEN_NAME="${TOKEN_NAME:-cli-token}"
REALM="${REALM:-pam}"
ROLE="${ROLE:-PVEVMAdmin}"
NODES=""
DRY_RUN=false
UNINSTALL=false
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=10"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --user-name)
            USER_NAME="$2"
            shift 2
            ;;
        --token-name)
            TOKEN_NAME="$2"
            shift 2
            ;;
        --realm)
            REALM="$2"
            shift 2
            ;;
        --role)
            ROLE="$2"
            shift 2
            ;;
        --nodes|--node)            # accept singular form too
            NODES="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --help|-h)
            sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# //'
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}" >&2
            exit 1
            ;;
    esac
done

# Full user ID
USER_ID="${USER_NAME}@${REALM}"
FULL_TOKEN_ID="${USER_ID}!${TOKEN_NAME}"

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "=============================================="
    echo "  Proxmox CLI User Setup Script"
    echo "=============================================="
    echo -e "${NC}"
}

# Print current configuration
print_config() {
    echo -e "${GREEN}Configuration:${NC}"
    echo "  User:        ${USER_ID}"
    echo "  Token Name:  ${TOKEN_NAME}"
    echo "  Full Token:  ${FULL_TOKEN_ID}"
    echo "  Role:        ${ROLE}"
    echo "  Nodes:       ${NODES:-<auto-discover>}"
    echo "  Dry Run:     ${DRY_RUN}"
    echo "  Uninstall:   ${UNINSTALL}"
    echo ""
}

# Run command on Proxmox node via SSH
run_ssh() {
    local node="$1"
    local cmd="$2"
    local description="$3"

    echo -e "${YELLOW}[$node]${NC} $description"

    if $DRY_RUN; then
        echo -e "  ${BLUE}[DRY-RUN]${NC} ssh $node \"$cmd\""
        return 0
    fi

    if ssh $SSH_OPTS "$node" "$cmd" 2>&1; then
        echo -e "  ${GREEN}✓ Success${NC}"
        return 0
    else
        echo -e "  ${RED}✗ Failed${NC}"
        return 1
    fi
}

# Get list of nodes
get_nodes() {
    if [[ -n "$NODES" ]]; then
        echo "$NODES" | tr ',' ' '
        return
    fi

    # Auto-discover: query the cluster via the first reachable common node.
    local first_node=""
    for node in pve pve1 pve1.local proxmox; do
        if ssh $SSH_OPTS "$node" "echo ok" &>/dev/null; then
            first_node="$node"
            break
        fi
    done

    if [[ -z "$first_node" ]]; then
        echo -e "${RED}Error: Could not auto-discover nodes. Please specify with --nodes${NC}" >&2
        exit 1
    fi

    # Query the cluster; fall back to the single discovered node if the query fails.
    local discovered
    if discovered=$(ssh $SSH_OPTS "$first_node" "pvesh get /nodes --output-format json" 2>/dev/null | jq -r '.[].node' 2>/dev/null); then
        echo "$discovered" | tr '\n' ' '
    else
        echo "$first_node"
    fi
}

# Create user on Proxmox
create_user() {
    local node="$1"

    echo -e "\n${BLUE}=== Creating User ===${NC}"

    # Create user
    run_ssh "$node" "pveum user add ${USER_ID}" "Creating user ${USER_ID}" || true

    # Set password only for pam (system) realm
    if [[ "$REALM" == "pam" ]]; then
        local password
        echo -e "\n${YELLOW}Enter password for ${USER_ID} (leave empty for random):${NC}"
        read -rs password
        echo ""

        if [[ -z "$password" ]]; then
            password=$(openssl rand -base64 24)
            echo -e "${GREEN}Generated random password${NC}"
        fi

        if ! $DRY_RUN; then
            # Pipe via stdin to avoid leaking the password through argv/SSH command line.
            printf '%s:%s\n' "$USER_NAME" "$password" | ssh $SSH_OPTS "$node" "chpasswd"
            echo -e "  ${GREEN}✓ Password set${NC}"
        fi
    fi
}

# Extract token secret value from pveum output (JSON preferred, grep fallback).
extract_token_value() {
    local output="$1"
    local value
    if value=$(echo "$output" | jq -r '.value // empty' 2>/dev/null) && [[ -n "$value" ]]; then
        echo "$value"
        return 0
    fi
    echo "$output" \
        | grep -oP 'value\s*│\s*\K\S+' 2>/dev/null \
        || echo "$output" | grep -oP '^\s*value\s*:?\s*\K[0-9a-f-]+' 2>/dev/null \
        || echo "$output" | grep -oP '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' 2>/dev/null \
        || true
}

# Create API token
create_token() {
    local node="$1"

    echo -e "\n${BLUE}=== Creating API Token ===${NC}"

    # Create token (JSON output for reliable parsing; human-readable fallback for old PVE)
    local output
    if $DRY_RUN; then
        echo -e "  ${BLUE}[DRY-RUN]${NC} ssh $node \"pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0\""
        echo -e "  ${YELLOW}Token value would be displayed here${NC}"
        return 0
    fi

    output=$(ssh $SSH_OPTS "$node" "pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 --output-format json 2>/dev/null || pveum user token add ${USER_ID} ${TOKEN_NAME} --privsep 0 2>&1")

    local token_value
    token_value=$(extract_token_value "$output")

    if [[ -n "$token_value" ]]; then
        echo -e "  ${GREEN}✓ Token created${NC}"
        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}  API TOKEN CREATED SUCCESSFULLY${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo -e "  User:        ${USER_ID}"
        echo -e "  Token Name:  ${TOKEN_NAME}"
        echo -e "  Token Value: ${YELLOW}${token_value}${NC}"
        echo ""
        echo -e "${RED}⚠️  SAVE THIS TOKEN VALUE - IT WON'T BE SHOWN AGAIN${NC}"
        echo ""

        # Generate environment export
        echo -e "${BLUE}Add to your ~/.bashrc or ~/.zshrc:${NC}"
        echo ""
        echo "export PVE_HOST=<your-proxmox-host>"
        echo "export PVE_USER='${USER_ID}'"
        echo "export PVE_TOKEN_NAME='${TOKEN_NAME}'"
        echo "export PVE_TOKEN_VALUE='${token_value}'"
        echo ""

        # Generate .env file
        echo -e "${BLUE}.env file content:${NC}"
        echo ""
        echo "PVE_HOST=<your-proxmox-host>"
        echo "PVE_USER=${USER_ID}"
        echo "PVE_TOKEN_NAME=${TOKEN_NAME}"
        echo "PVE_TOKEN_VALUE=${token_value}"
        echo ""
    else
        echo -e "  ${YELLOW}Token may already exist or creation failed${NC}"
        echo "$output"
    fi
}

# Assign permissions
assign_permissions() {
    local node="$1"

    echo -e "\n${BLUE}=== Assigning Permissions ===${NC}"

    # Assign role to token at root (/)
    run_ssh "$node" "pveum aclmod / -tokens ${FULL_TOKEN_ID} -roles ${ROLE}" \
        "Assigning ${ROLE} role to token at /"

    # Also assign to user (for some operations that check user perms)
    run_ssh "$node" "pveum aclmod / -users ${USER_ID} -roles ${ROLE}" \
        "Assigning ${ROLE} role to user at /"

    echo -e "\n${GREEN}Permissions assigned:${NC}"
    echo "  Token: ${FULL_TOKEN_ID}"
    echo "  Role:  ${ROLE}"
    echo "  Path:  / (root - full access)"
}

# Verify setup
verify_setup() {
    local node="$1"

    echo -e "\n${BLUE}=== Verifying Setup ===${NC}"

    # Check user exists
    run_ssh "$node" "pveum user list | grep -q ${USER_ID}" "User ${USER_ID} exists"

    # Check token exists
    run_ssh "$node" "pveum user token list ${USER_ID} | grep -q ${TOKEN_NAME}" "Token ${TOKEN_NAME} exists"

    # Check ACL
    run_ssh "$node" "pveum acl list | grep -q ${FULL_TOKEN_ID}" "ACL entry for token exists"
}

# Uninstall user and token
uninstall() {
    local node="$1"

    echo -e "\n${RED}=== Uninstalling User ===${NC}"

    # Remove token
    run_ssh "$node" "pveum user token remove ${USER_ID} ${TOKEN_NAME}" \
        "Removing token ${TOKEN_NAME}" || true

    # Remove user
    run_ssh "$node" "pveum user delete ${USER_ID}" \
        "Removing user ${USER_ID}" || true

    echo -e "\n${GREEN}Uninstall complete${NC}"
}

# Main execution
main() {
    print_banner
    print_config

    # Get nodes
    echo -e "${BLUE}Discovering nodes...${NC}"
    NODE_LIST=$(get_nodes)
    echo -e "Found nodes: ${GREEN}${NODE_LIST}${NC}"

    # Use first node for user creation (users are cluster-wide)
    FIRST_NODE=$(echo "$NODE_LIST" | awk '{print $1}')

    if [[ -z "$FIRST_NODE" ]]; then
        echo -e "${RED}Error: No accessible nodes found${NC}" >&2
        exit 1
    fi

    echo -e "Using ${FIRST_NODE} for user management${NC}"

    if $UNINSTALL; then
        uninstall "$FIRST_NODE"
        exit 0
    fi

    # Create user
    create_user "$FIRST_NODE"

    # Create token
    create_token "$FIRST_NODE"

    # Assign permissions
    assign_permissions "$FIRST_NODE"

    # Verify
    verify_setup "$FIRST_NODE"

    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}  Setup Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "Test your setup with:"
    echo -e "  ${BLUE}./build/pve vm list${NC}"
    echo ""
}

# Run main
main "$@"
