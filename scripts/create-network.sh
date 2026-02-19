#!/bin/bash
set -euo pipefail

XML_PATH=$1
NETWORK_NAME=$2

if [ ! -f "$XML_PATH" ]; then
    echo "[!] Network XML not found: $XML_PATH"
    exit 1
fi

if virsh net-info "$NETWORK_NAME" >/dev/null 2>&1; then
    echo "[=] Network $NETWORK_NAME already defined"
else
    echo "[+] Defining network $NETWORK_NAME"
    virsh net-define "$XML_PATH"
fi

if virsh net-info "$NETWORK_NAME" | grep -q "Active:.*yes"; then
    echo "[=] Network $NETWORK_NAME already active"
else
    echo "[+] Starting network $NETWORK_NAME"
    virsh net-start "$NETWORK_NAME"
fi

virsh net-autostart "$NETWORK_NAME"

echo "[✓] Network $NETWORK_NAME ready"
