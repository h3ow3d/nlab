#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

if [ $# -lt 2 ]; then
  log_error "Usage: $0 <xml-path> <network-name>"
  exit 1
fi

XML_PATH=$1
NETWORK_NAME=$2

if [ ! -f "$XML_PATH" ]; then
    log_error "Network XML not found: $XML_PATH"
    exit 1
fi

if virsh net-info "$NETWORK_NAME" >/dev/null 2>&1; then
    log_skip "Network $NETWORK_NAME already defined"
else
    log_info "Defining network $NETWORK_NAME"
    virsh net-define "$XML_PATH"
fi

if virsh net-info "$NETWORK_NAME" | grep -q "Active:.*yes"; then
    log_skip "Network $NETWORK_NAME already active"
else
    log_info "Starting network $NETWORK_NAME"
    virsh net-start "$NETWORK_NAME"
fi

virsh net-autostart "$NETWORK_NAME"

log_ok "Network $NETWORK_NAME ready"
