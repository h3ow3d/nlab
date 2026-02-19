#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../scripts/lib.sh
source "$SCRIPT_DIR/../scripts/lib.sh"

BASE_DIR="/var/lib/libvirt/images"
BASE_IMAGE="$BASE_DIR/ubuntu-base.qcow2"
IMAGE_NAME="jammy-server-cloudimg-amd64.img"
IMAGE_URL="https://cloud-images.ubuntu.com/jammy/current/${IMAGE_NAME}"
SUMS_URL="https://cloud-images.ubuntu.com/jammy/current/SHA256SUMS"

download_image() {
    log_info "Downloading Ubuntu cloud image"
    wget "$IMAGE_URL" -O ubuntu.img
    wget "$SUMS_URL" -O ubuntu.SHA256SUMS
}

verify_checksum() {
    log_info "Verifying checksum"
    EXPECTED_SHA256=$(grep "$IMAGE_NAME" ubuntu.SHA256SUMS | awk '{print $1}')

    if [ -z "$EXPECTED_SHA256" ]; then
        log_error "Could not find checksum for $IMAGE_NAME in SHA256SUMS"
        rm -f ubuntu.img ubuntu.SHA256SUMS
        exit 1
    fi

    ACTUAL_SHA256=$(sha256sum ubuntu.img | awk '{print $1}')

    if [ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]; then
        log_error "Checksum mismatch — expected $EXPECTED_SHA256 but got $ACTUAL_SHA256"
        rm -f ubuntu.img ubuntu.SHA256SUMS
        exit 1
    fi

    rm -f ubuntu.SHA256SUMS
    log_ok "Checksum OK"
}

install_image() {
    log_info "Moving image to libvirt storage"
    sudo mv ubuntu.img "$BASE_IMAGE"
    sudo chown libvirt-qemu:kvm "$BASE_IMAGE"

    log_ok "Base image ready"
}

# ---- main ----

if [ -f "$BASE_IMAGE" ]; then
    log_skip "Base image already exists"
    exit 0
fi

download_image
verify_checksum
install_image
