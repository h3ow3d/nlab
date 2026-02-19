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

LAYOUT_FILE="stacks/${STACK}/layout.yaml"

if [ ! -f "$LAYOUT_FILE" ]; then
  echo "[!] No layout.yaml found at $LAYOUT_FILE"
  exit 1
fi

# Parse layout.yaml into shell variables
eval "$(python3 - "$LAYOUT_FILE" <<'PYEOF'
import sys, yaml, shlex

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

panes = data.get('panes', [])
print("TMUX_LAYOUT=" + shlex.quote(data.get('layout', 'tiled')))
print("PANE_COUNT=" + str(len(panes)))

for i, pane in enumerate(panes):
    ptype = pane.get('type', 'command')
    print("PANE_{}_TYPE={}".format(i, shlex.quote(ptype)))
    if ptype == 'ssh':
        print("PANE_{}_VM={}".format(i, shlex.quote(pane.get('vm', ''))))
    cmd = pane.get('command', '')
    print("PANE_{}_CMD={}".format(i, shlex.quote(cmd)))
PYEOF
)"

MAX_WAIT=180
ELAPSED=0

# Build ordered list of unique SSH VMs from the pane definitions
SSH_VMS=()
for i in $(seq 0 $((PANE_COUNT - 1))); do
  ptype_var="PANE_${i}_TYPE"
  if [ "${!ptype_var}" = "ssh" ]; then
    vm_var="PANE_${i}_VM"
    vm="${!vm_var}"
    already=false
    for existing in "${SSH_VMS[@]}"; do
      if [ "$existing" = "$vm" ]; then
        already=true
        break
      fi
    done
    if ! $already; then
      SSH_VMS+=("$vm")
    fi
  fi
done

VM_COUNT=${#SSH_VMS[@]}
declare -A VM_MAC
declare -A VM_IP
declare -A VM_SSH

for vm in "${SSH_VMS[@]}"; do
  VM_MAC[$vm]=""
  VM_IP[$vm]=""
  VM_SSH[$vm]="pending"
done

echo "[+] Waiting for VMs to become ready..."
# Reserve lines for per-VM status output (VM_COUNT lines + 1 blank)
for i in $(seq 1 $((VM_COUNT + 1))); do echo; done

# Resolve MAC addresses (stable identity)
while true; do
  all_macs=true
  for vm in "${SSH_VMS[@]}"; do
    if [ -z "${VM_MAC[$vm]}" ]; then
      mac=$(virsh domiflist "${STACK}-${vm}" 2>/dev/null | awk '/network/ {print $5}' || true)
      if [ -n "$mac" ]; then
        VM_MAC[$vm]="$mac"
      fi
    fi
    if [ -z "${VM_MAC[$vm]}" ]; then
      all_macs=false
    fi
  done

  if $all_macs; then break; fi

  if [ "$ELAPSED" -ge "$MAX_WAIT" ]; then
    echo "[!] Timeout waiting for VM network interfaces"
    exit 1
  fi

  sleep 1
  ELAPSED=$((ELAPSED + 1))
done

# Wait for all SSH VMs to become reachable
while true; do
  for vm in "${SSH_VMS[@]}"; do
    ip=$(virsh net-dhcp-leases "$NETWORK" \
      | awk -v mac="${VM_MAC[$vm]}" '$0 ~ mac {print $5}' \
      | cut -d/ -f1)
    VM_IP[$vm]="${ip:-}"

    if [ -n "${VM_IP[$vm]}" ] && [ "${VM_SSH[$vm]}" != "ready" ]; then
      if ssh -o BatchMode=yes \
             -o ConnectTimeout=1 \
             -o StrictHostKeyChecking=no \
             -i "$KEY" \
             "${USER}@${VM_IP[$vm]}" true >/dev/null 2>&1; then
        VM_SSH[$vm]="ready"
      fi
    fi
  done

  # Move cursor up (VM_COUNT lines + 1 blank) to overwrite status
  printf "\033[%dF" $((VM_COUNT + 1))

  for vm in "${SSH_VMS[@]}"; do
    state=$(virsh domstate "${STACK}-${vm}" 2>/dev/null || echo "unknown")
    printf "%-18s | state: %-12s | ip: %-15s | ssh: %-8s\n" \
      "${STACK}-${vm}" "${state:-unknown}" "${VM_IP[$vm]:-pending}" "${VM_SSH[$vm]:-pending}"
  done

  printf "\n"

  all_ready=true
  for vm in "${SSH_VMS[@]}"; do
    if [ "${VM_SSH[$vm]}" != "ready" ]; then
      all_ready=false
      break
    fi
  done

  if $all_ready; then break; fi

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

# Create additional panes (first pane created automatically by new-session)
for i in $(seq 1 $((PANE_COUNT - 1))); do
  tmux split-window -t "$SESSION"
done

# Send commands to each pane
for i in $(seq 0 $((PANE_COUNT - 1))); do
  ptype_var="PANE_${i}_TYPE"
  ptype="${!ptype_var}"
  tmux select-pane -t "$i"

  if [ "$ptype" = "ssh" ]; then
    vm_var="PANE_${i}_VM"
    vm="${!vm_var}"
    tmux send-keys "ssh -i $KEY -o StrictHostKeyChecking=no ${USER}@${VM_IP[$vm]}" C-m
  else
    cmd_var="PANE_${i}_CMD"
    cmd="${!cmd_var}"
    cmd="${cmd//\{stack\}/${STACK}}"
    tmux send-keys "$cmd" C-m
  fi
done

tmux select-layout "$TMUX_LAYOUT"

tmux attach-session -t "$SESSION"

