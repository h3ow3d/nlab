#!/bin/bash
set -euo pipefail

export LIBVIRT_DEFAULT_URI=qemu:///system

if [ $# -lt 1 ]; then
  echo "[!] Usage: $0 <network>"
  exit 1
fi

NETWORK=$1

if virsh net-info "$NETWORK" >/dev/null 2>&1; then
  echo "[+] Destroying network $NETWORK"
  virsh net-destroy "$NETWORK" 2>/dev/null || true
  virsh net-undefine "$NETWORK"
  echo "[✓] Network $NETWORK removed"
else
  echo "[=] Network $NETWORK does not exist"
fi

