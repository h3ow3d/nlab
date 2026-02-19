#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

if [ $# -lt 2 ]; then
  log_error "Usage: $0 <stack> <role>"
  exit 1
fi

STACK=$1
ROLE=$2
NAME="${STACK}-${ROLE}"
SEED="${NAME}-seed.iso"

log_info "Destroy request: $NAME"

if virsh dominfo "$NAME" >/dev/null 2>&1; then
  log_info "Stopping $NAME (if running)"
  virsh destroy "$NAME" 2>/dev/null || true

  log_info "Undefining $NAME (and removing storage)"
  virsh undefine "$NAME" --remove-all-storage
else
  log_skip "Domain $NAME not found (already gone)"
fi

# Remove local seed ISO (option 1 keeps these in repo root)
rm -f "$SEED" || true

# Hard verification
if virsh dominfo "$NAME" >/dev/null 2>&1; then
  log_error "FAILED: $NAME still exists after destroy"
  exit 1
fi

log_ok "$NAME deleted"

