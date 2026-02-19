#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

if [ $# -lt 1 ]; then
  log_error "Usage: $0 <stack>"
  exit 1
fi

STACK=$1
KEY_DIR="keys/$STACK"
KEY_PATH="$KEY_DIR/id_ed25519"

mkdir -p "$KEY_DIR"

if [ -f "$KEY_PATH" ]; then
    log_skip "SSH key already exists for stack $STACK"
else
    log_info "Generating SSH key for stack $STACK"
    ssh-keygen -t ed25519 -f "$KEY_PATH" -N "" -q
    log_ok "Key generated at $KEY_PATH"
fi
