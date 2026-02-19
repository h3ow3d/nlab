#!/bin/bash
set -euo pipefail

export LIBVIRT_DEFAULT_URI=qemu:///system

if [ $# -lt 5 ]; then
  echo "[!] Usage: $0 <stack> <role> <memory-mb> <vcpus> <network>"
  exit 1
fi

STACK=$1
ROLE=$2
MEMORY=$3
VCPUS=$4
NETWORK=$5

NAME="${STACK}-${ROLE}"
SEED="${NAME}-seed.iso"

BASE_IMAGE="/var/lib/libvirt/images/ubuntu-base.qcow2"

USER_DATA_TEMPLATE="stacks/${STACK}/${ROLE}/user-data"
META_DATA="stacks/${STACK}/${ROLE}/meta-data"

KEY_DIR="keys/$STACK"
PUB_KEY_FILE="$KEY_DIR/id_ed25519.pub"

if [ ! -f "$BASE_IMAGE" ]; then
  echo "[!] Base image not found at $BASE_IMAGE"
  exit 1
fi

if [ ! -f "$PUB_KEY_FILE" ]; then
  echo "[!] SSH public key not found for stack $STACK"
  exit 1
fi

if virsh dominfo "$NAME" >/dev/null 2>&1; then
  echo "[=] VM $NAME already exists"
  exit 0
fi

TMP_USER_DATA="/tmp/${NAME}-user-data"

sed "s|__SSH_PUBLIC_KEY__|$(cat "$PUB_KEY_FILE")|" \
  "$USER_DATA_TEMPLATE" > "$TMP_USER_DATA"

echo "[+] Creating cloud-init ISO"
cloud-localds "$SEED" "$TMP_USER_DATA" "$META_DATA"
rm -f "$TMP_USER_DATA"

echo "[+] Installing VM $NAME"

virt-install \
  --name "$NAME" \
  --memory "$MEMORY" \
  --vcpus "$VCPUS" \
  --disk size=20,backing_store="$BASE_IMAGE",format=qcow2 \
  --disk path="$SEED",device=cdrom,readonly=on \
  --os-variant ubuntu22.04 \
  --network network="$NETWORK" \
  --graphics none \
  --import \
  --noautoconsole

echo "[✓] VM $NAME deployed"
