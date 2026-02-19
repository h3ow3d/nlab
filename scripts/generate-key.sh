#!/bin/bash
set -euo pipefail

STACK=$1
KEY_DIR="keys/$STACK"
KEY_PATH="$KEY_DIR/id_ed25519"

mkdir -p "$KEY_DIR"

if [ -f "$KEY_PATH" ]; then
    echo "[=] SSH key already exists for stack $STACK"
else
    echo "[+] Generating SSH key for stack $STACK"
    ssh-keygen -t ed25519 -f "$KEY_PATH" -N "" -q
    echo "[✓] Key generated at $KEY_PATH"
fi
