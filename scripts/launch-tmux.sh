#!/bin/bash
set -euo pipefail

export LIBVIRT_DEFAULT_URI=qemu:///system

if [ $# -lt 2 ]; then
  echo "[!] Usage: $0 <stack> <network>"
  exit 1
fi

STACK=$1
NETWORK=$2

SESSION="red-team-${STACK}"
USER="ubuntu"
KEY="keys/${STACK}/id_ed25519"

ATTACKER="${STACK}-attacker"
TARGET="${STACK}-target"

MAX_WAIT=180
ELAPSED=0

echo "[+] Waiting for VMs to become ready..."
echo
echo
echo

# Resolve MAC addresses once (stable identity)
ATTACKER_MAC=""
TARGET_MAC=""

while true; do
  ATTACKER_MAC=$(virsh domiflist "$ATTACKER" 2>/dev/null | awk '/network/ {print $5}' || true)
  TARGET_MAC=$(virsh domiflist "$TARGET" 2>/dev/null | awk '/network/ {print $5}' || true)

  if [ -n "$ATTACKER_MAC" ] && [ -n "$TARGET_MAC" ]; then
    break
  fi

  if [ "$ELAPSED" -ge "$MAX_WAIT" ]; then
    echo "[!] Timeout waiting for VM network interfaces"
    exit 1
  fi

  sleep 1
  ELAPSED=$((ELAPSED + 1))
done

while true; do
  ATTACKER_STATE=$(virsh domstate "$ATTACKER" 2>/dev/null || echo "unknown")
  TARGET_STATE=$(virsh domstate "$TARGET" 2>/dev/null || echo "unknown")

  ATTACKER_IP=$(virsh net-dhcp-leases "$NETWORK" \
    | awk -v mac="$ATTACKER_MAC" '$0 ~ mac {print $5}' \
    | cut -d/ -f1)

  TARGET_IP=$(virsh net-dhcp-leases "$NETWORK" \
    | awk -v mac="$TARGET_MAC" '$0 ~ mac {print $5}' \
    | cut -d/ -f1)

  ATTACKER_SSH="pending"
  TARGET_SSH="pending"

  if [ -n "${ATTACKER_IP:-}" ]; then
    if ssh -o BatchMode=yes \
           -o ConnectTimeout=1 \
           -o StrictHostKeyChecking=no \
           -i "$KEY" \
           ${USER}@"$ATTACKER_IP" true >/dev/null 2>&1; then
      ATTACKER_SSH="ready"
    fi
  fi

  if [ -n "${TARGET_IP:-}" ]; then
    if ssh -o BatchMode=yes \
           -o ConnectTimeout=1 \
           -o StrictHostKeyChecking=no \
           -i "$KEY" \
           ${USER}@"$TARGET_IP" true >/dev/null 2>&1; then
      TARGET_SSH="ready"
    fi
  fi

  # Move cursor up 3 lines (no full screen clear)
  printf "\033[3F"

  printf "%-18s | state: %-12s | ip: %-15s | ssh: %-8s\n" \
    "$ATTACKER" "${ATTACKER_STATE:-unknown}" "${ATTACKER_IP:-pending}" "$ATTACKER_SSH"

  printf "%-18s | state: %-12s | ip: %-15s | ssh: %-8s\n" \
    "$TARGET" "${TARGET_STATE:-unknown}" "${TARGET_IP:-pending}" "$TARGET_SSH"

  printf "\n"

  if [ "$ATTACKER_SSH" = "ready" ] && [ "$TARGET_SSH" = "ready" ]; then
    break
  fi

  if [ "$ELAPSED" -ge "$MAX_WAIT" ]; then
    echo
    echo "[!] Timeout waiting for VM readiness"
    exit 1
  fi

  sleep 2
  ELAPSED=$((ELAPSED + 2))
done

echo
echo "[✓] All VMs ready"
sleep 1

# ---- TMUX LAYOUT ----

tmux has-session -t "$SESSION" 2>/dev/null && tmux kill-session -t "$SESSION"

tmux new-session -d -s "$SESSION"

# Split right pane
tmux split-window -h -t "$SESSION"

# Split left pane vertically
tmux select-pane -t 0
tmux split-window -v -t "$SESSION"

# Pane layout:
# 0 = attacker (top-left)
# 1 = target   (bottom-left)
# 2 = tcpdump  (right)

tmux select-pane -t 0
tmux send-keys "ssh -i $KEY -o StrictHostKeyChecking=no ${USER}@${ATTACKER_IP}" C-m

tmux select-pane -t 1
tmux send-keys "ssh -i $KEY -o StrictHostKeyChecking=no ${USER}@${TARGET_IP}" C-m

tmux select-pane -t 2
tmux send-keys "sudo tcpdump -i virbr-${STACK} -nn -tttt -vvv" C-m

tmux select-layout even-horizontal

tmux attach-session -t "$SESSION"

