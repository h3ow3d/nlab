#!/bin/bash
set -euo pipefail
trap 'printf "\e[?25h"' EXIT

function print_table() {
    local title="$1"
    shift
    echo "\e[1;32m$title\e[0m"
    printf '%-25s %-20s %-20s %-20s %-12s\n' "KEY" "NETWORK" "VM" "RTT" "READINESS"
    printf '%-25s %-20s %-20s %-20s %-12s\n' "---" "---" "---" "---" "---"
    while true; do
        # Live updating logic goes here
        # Fetch data and format it
        # Example:
        printf '%-25s %-20s %-20s %-20s %-12s\n' "key1" "network1" "vm1" "100ms" "Ready"
        sleep 1
        # Clear the line for repainting
        printf '\r';
    done
}

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <stack> <network> [roles...]"
    exit 1
fi

stack="$1"
network="$2"
roles="${@:3}"

print_table "Live Dashboard for Stack: $stack and Network: $network"
