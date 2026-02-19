#!/bin/bash
set -euo pipefail
export LIBVIRT_DEFAULT_URI=qemu:///system

STACK=$1
ROLE=$2
NAME="${STACK}-${ROLE}"
SEED="${NAME}-seed.iso"

echo "[+] Destroy request: $NAME"

if virsh dominfo "$NAME" >/dev/null 2>&1; then
  echo "[+] Stopping $NAME (if running)"
  virsh destroy "$NAME" 2>/dev/null || true

  echo "[+] Undefining $NAME (and removing storage)"
  virsh undefine "$NAME" --remove-all-storage
else
  echo "[=] Domain $NAME not found (already gone)"
fi

# Remove local seed ISO (option 1 keeps these in repo root)
rm -f "$SEED" || true

# Hard verification
if virsh dominfo "$NAME" >/dev/null 2>&1; then
  echo "[!] FAILED: $NAME still exists after destroy"
  exit 1
fi

echo "[✓] $NAME deleted"

