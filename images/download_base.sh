#!/bin/bash
set -euo pipefail

BASE_DIR="/var/lib/libvirt/images"
BASE_IMAGE="$BASE_DIR/ubuntu-base.qcow2"

if [ -f "$BASE_IMAGE" ]; then
    echo "[=] Base image already exists"
    exit 0
fi

echo "[+] Downloading Ubuntu cloud image"
wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img -O ubuntu.img

echo "[+] Moving image to libvirt storage"
sudo mv ubuntu.img "$BASE_IMAGE"
sudo chown libvirt-qemu:kvm "$BASE_IMAGE"

echo "[✓] Base image ready"
