#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

if [ $# -lt 1 ]; then
  log_error "Usage: $0 <network>"
  exit 1
fi

NETWORK=$1

if virsh net-info "$NETWORK" >/dev/null 2>&1; then
  log_info "Destroying network $NETWORK"
  virsh net-destroy "$NETWORK" 2>/dev/null || true
  virsh net-undefine "$NETWORK"
  log_ok "Network $NETWORK removed"
else
  log_skip "Network $NETWORK does not exist"
fi

