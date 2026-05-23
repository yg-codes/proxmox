#!/bin/bash
#
# Proxmox SSH Command Runner
# Run non-interactive SSH commands on one or more Proxmox nodes
#
# Usage:
#   ./pve-ssh-exec.sh [OPTIONS] -- <command>
#
# Options:
#   -n, --nodes      Comma-separated list of nodes (required)
#   -u, --user       SSH user (default: root)
#   -p, --parallel   Run commands in parallel
#   -o, --output     Output format: text, json (default: text)
#   -e, --env        Environment file to source on remote host
#   -t, --timeout    SSH timeout in seconds (default: 30)
#   --dry-run        Show commands without executing
#   -h, --help       Show this help
#
# Examples:
#   # Run command on single node
#   ./pve-ssh-exec.sh -n pve1 -- "pvesh get /version"
#
#   # Run on multiple nodes sequentially
#   ./pve-ssh-exec.sh -n pve1,pve2,pve3 -- "pvesh get /nodes"
#
#   # Run in parallel on all nodes
#   ./pve-ssh-exec.sh -n pve1,pve2,pve3 --parallel -- "uptime"
#
#   # Check cluster status on all nodes
#   ./pve-ssh-exec.sh -n pve1,pve2,pve3 -- "pvecm status"
#
#   # Run with JSON output for parsing
#   ./pve-ssh-exec.sh -n pve1 -o json -- "pvesh get /cluster/resources --type vm"
#

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
NODES=""
SSH_USER="root"
PARALLEL=false
OUTPUT="text"
ENV_FILE=""
TIMEOUT=30
DRY_RUN=false
COMMAND=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--nodes)
            NODES="$2"
            shift 2
            ;;
        -u|--user)
            SSH_USER="$2"
            shift 2
            ;;
        -p|--parallel)
            PARALLEL=true
            shift
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -e|--env)
            ENV_FILE="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --)
            shift
            COMMAND="$*"
            break
            ;;
        -h|--help)
            sed -n '/^# Usage:/,/^$/p' "$0" | sed 's/^# //'
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$NODES" ]]; then
    echo -e "${RED}Error: --nodes is required${NC}"
    exit 1
fi

if [[ -z "$COMMAND" ]]; then
    echo -e "${RED}Error: No command specified (use -- <command>)${NC}"
    exit 1
fi

# Convert nodes to array
IFS=',' read -ra NODE_ARRAY <<< "$NODES"

# SSH options
SSH_OPTS="-o StrictHostKeyChecking=no -o ConnectTimeout=${TIMEOUT} -o BatchMode=yes"

# Run command on single node
run_on_node() {
    local node="$1"
    local cmd="$2"
    local result
    local exit_code

    if $DRY_RUN; then
        if [[ "$OUTPUT" == "json" ]]; then
            echo "{\"node\":\"$node\",\"dry_run\":true,\"command\":\"$cmd\"}"
        else
            echo -e "${BLUE}[DRY-RUN]${NC} ssh ${SSH_USER}@${node} \"${cmd}\""
        fi
        return 0
    fi

    # Build SSH command
    local ssh_cmd="ssh $SSH_OPTS ${SSH_USER}@${node}"

    # Add env file if specified
    if [[ -n "$ENV_FILE" && -f "$ENV_FILE" ]]; then
        local env_content
        env_content=$(cat "$ENV_FILE" | base64 -w 0)
        ssh_cmd="$ssh_cmd \"echo '$env_content' | base64 -d > /tmp/env.sh && source /tmp/env.sh && $cmd\""
    else
        ssh_cmd="$ssh_cmd \"$cmd\""
    fi

    # Execute and capture output
    if result=$(eval "$ssh_cmd" 2>&1); then
        exit_code=0
    else
        exit_code=$?
    fi

    # Format output
    if [[ "$OUTPUT" == "json" ]]; then
        # Escape result for JSON
        local escaped_result
        escaped_result=$(echo "$result" | jq -Rs . 2>/dev/null || echo "\"$result\"")
        echo "{\"node\":\"$node\",\"exit_code\":$exit_code,\"result\":$escaped_result}"
    else
        echo -e "\n${GREEN}=== $node ===${NC}"
        if [[ $exit_code -eq 0 ]]; then
            echo -e "${GREEN}✓ Success${NC}"
        else
            echo -e "${RED}✗ Failed (exit code: $exit_code)${NC}"
        fi
        echo "$result"
    fi

    return $exit_code
}

# Main execution
main() {
    local pids=()
    local results=()
    local failed=0

    echo -e "${BLUE}Running on ${#NODE_ARRAY[@]} node(s)${NC}"
    echo -e "Command: ${YELLOW}${COMMAND}${NC}"
    echo ""

    if $PARALLEL; then
        # Parallel execution
        local temp_dir
        temp_dir=$(mktemp -d)
        trap "rm -rf $temp_dir" EXIT

        for i in "${!NODE_ARRAY[@]}"; do
            local node="${NODE_ARRAY[$i]}"
            local output_file="${temp_dir}/node_${i}.out"

            (
                run_on_node "$node" "$COMMAND" > "$output_file" 2>&1
                echo $? > "${output_file}.exit"
            ) &
            pids+=($!)
        done

        # Wait for all to complete
        for pid in "${pids[@]}"; do
            wait "$pid" || true
        done

        # Collect results
        for i in "${!NODE_ARRAY[@]}"; do
            local output_file="${temp_dir}/node_${i}.out"
            local exit_file="${temp_dir}/node_${i}.exit"
            local node="${NODE_ARRAY[$i]}"
            local exit_code=0

            if [[ -f "$exit_file" ]]; then
                exit_code=$(cat "$exit_file")
            fi

            if [[ "$OUTPUT" != "json" ]]; then
                cat "$output_file"
            else
                cat "$output_file"
            fi

            if [[ $exit_code -ne 0 ]]; then
                ((failed++))
            fi
        done
    else
        # Sequential execution
        for node in "${NODE_ARRAY[@]}"; do
            if ! run_on_node "$node" "$COMMAND"; then
                ((failed++))
            fi
        done
    fi

    # Summary
    if [[ "$OUTPUT" != "json" ]]; then
        echo ""
        echo -e "${BLUE}=== Summary ===${NC}"
        echo "Total nodes: ${#NODE_ARRAY[@]}"
        echo -e "Successful:  ${GREEN}$((${#NODE_ARRAY[@]} - failed))${NC}"
        if [[ $failed -gt 0 ]]; then
            echo -e "Failed:      ${RED}${failed}${NC}"
        fi
    fi

    if [[ $failed -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
