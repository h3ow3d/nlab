#!/bin/bash
# create-dashboard.sh – live creation dashboard for nlab stacks.
# Renders in-place tables for KEYS, NETWORKS, VMS, ARTIFACTS, and EVENTS,
# refreshing every second until signalled.
#
# Usage: ./scripts/create-dashboard.sh <stack> <network>

set -euo pipefail

# ── colors ────────────────────────────────────────────────────────────────────
C_RESET=$'\033[0m'
C_BOLD=$'\033[1m'
C_DIM=$'\033[2m'
C_RED=$'\033[31m'
C_GREEN=$'\033[32m'
C_YELLOW=$'\033[33m'
C_CYAN=$'\033[36m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

if [ $# -lt 2 ]; then
  log_error "Usage: $0 <stack> <network>"
  exit 1
fi

STACK=$1
NETWORK=$2
EVENTS_FILE="logs/${STACK}-events.log"
KEY="keys/${STACK}/id_ed25519"
SSH_USER="ubuntu"
MAX_EVENTS=8
REFRESH=1

declare -A VM_SSH   # persists SSH readiness across iterations

# ── terminal control ──────────────────────────────────────────────────────────

hide_cursor() { printf '\033[?25l'; }
show_cursor()  { printf '\033[?25h'; }

cleanup() {
  local skip=$((PREV_LINES - TOTAL_LINES))
  [ "$skip" -gt 0 ] && printf '\033[%dB' "$skip"
  show_cursor
  printf '\n'
}
trap cleanup EXIT

# ── rendering helpers ─────────────────────────────────────────────────────────

TOTAL_LINES=0

# Print a line and increment the line counter used for cursor-up on next frame.
pline() {
  printf '%s\033[K\n' "${1:-}"
  TOTAL_LINES=$((TOTAL_LINES + 1))
}

# Print a section heading padded to 70 chars with dashes.
section() {
  local title="$1"
  local prefix="-- ${title} "
  local pad_len=$((70 - ${#prefix}))
  [ "$pad_len" -lt 1 ] && pad_len=1
  local pad
  pad=$(printf '%*s' "$pad_len" '' | tr ' ' '-')
  pline "${C_BOLD}${C_YELLOW}${prefix}${pad}${C_RESET}"
}

# Print a value colored by its meaning, optionally padded to <width> chars.
# Color scheme: green = good (yes/exists/ready/running),
#               red   = bad  (no/missing/failed),
#               yellow = in-progress (pending/booting/creating/waiting-ssh).
# Usage: cf <value> [<width>]
cf() {
  local val="$1" width="${2:-}"
  local color
  case "$val" in
    yes|exists|ready|running)         color="$C_GREEN"  ;;
    no|missing|failed)                color="$C_RED"    ;;
    pending|booting|creating|waiting-ssh) color="$C_YELLOW" ;;
    *)                                color=""          ;;
  esac
  if [ -n "$color" ]; then
    [ -n "$width" ] && printf '%s%-*s%s' "$color" "$width" "$val" "$C_RESET" \
                    || printf '%s%s%s'   "$color"          "$val" "$C_RESET"
  else
    [ -n "$width" ] && printf '%-*s' "$width" "$val" || printf '%s' "$val"
  fi
}

# ── section renderers ─────────────────────────────────────────────────────────

render_keys() {
  section "KEYS"
  pline "$(printf '  %-38s  %-8s  %s' 'FILE' 'STATUS' 'FINGERPRINT')"
  local priv="keys/${STACK}/id_ed25519"
  local pub="${priv}.pub"
  if [ -f "$priv" ]; then
    local fp
    # ssh-keygen -l output: "<bits> <hash> <comment> (<type>)"; field 2 is the fingerprint
    fp=$(ssh-keygen -l -f "$priv" 2>/dev/null | awk '{print $2}' || echo 'n/a')
    pline "$(printf '  %-38s  ' "$priv")$(cf exists 8)  ${C_DIM}${fp}${C_RESET}"
  else
    pline "$(printf '  %-38s  ' "$priv")$(cf missing)"
  fi
  if [ -f "$pub" ]; then
    pline "$(printf '  %-38s  ' "$pub")$(cf exists)"
  else
    pline "$(printf '  %-38s  ' "$pub")$(cf missing)"
  fi
  pline ""
}

render_networks() {
  section "NETWORKS"
  pline "$(printf '  %-20s  %-8s  %-8s  %-10s  %s' 'NAME' 'DEFINED' 'ACTIVE' 'AUTOSTART' 'BRIDGE')"
  local defined="no" active="no" autostart="no" bridge="n/a"
  if virsh net-info "$NETWORK" >/dev/null 2>&1; then
    local info
    info=$(virsh net-info "$NETWORK" 2>/dev/null || true)
    defined="yes"
    active=$(printf '%s'    "$info" | awk '/^Active:/{print $2}'    || echo 'no')
    autostart=$(printf '%s' "$info" | awk '/^Autostart:/{print $2}' || echo 'no')
    bridge=$(printf '%s'    "$info" | awk '/^Bridge:/{print $2}'    || echo 'n/a')
  fi
  pline "$(printf '  %-20s' "$NETWORK")  $(cf "$defined" 8)  $(cf "${active:-no}" 8)  $(cf "${autostart:-no}" 10)  ${bridge:-n/a}"
  pline ""
}

render_vms() {
  section "VMS"
  pline "$(printf '  %-22s  %-10s  %-18s  %-15s  %-8s  %s' \
    'NAME' 'STATE' 'MAC' 'IP' 'SSH' 'READINESS')"
  local domains
  domains=$(virsh list --all --name 2>/dev/null | grep "^${STACK}-" || true)
  if [ -z "$domains" ]; then
    pline "  (no domains yet)"
  else
    while IFS= read -r dom; do
      [ -z "$dom" ] && continue
      local state mac ip ssh_st readiness
      state=$(virsh domstate "$dom" 2>/dev/null | awk '{$1=$1;print}' || echo "unknown")
      mac=$(virsh domiflist "$dom" 2>/dev/null | awk '/network/{print $5; exit}' || echo "")
      ip=""
      if [ -n "$mac" ]; then
        ip=$(virsh net-dhcp-leases "$NETWORK" 2>/dev/null \
          | awk -v m="$mac" '$0 ~ m {print $5}' | cut -d/ -f1 | head -1 || echo "")
      fi
      ssh_st="${VM_SSH[$dom]:-pending}"
      if [ -n "$ip" ] && [ "$ssh_st" != "ready" ] && [ -f "$KEY" ]; then
        if ssh -n -o BatchMode=yes -o ConnectTimeout=1 \
               -o StrictHostKeyChecking=no \
               -o UserKnownHostsFile=/dev/null \
               -i "$KEY" "${SSH_USER}@${ip}" true >/dev/null 2>&1; then
          VM_SSH[$dom]="ready"
          ssh_st="ready"
        fi
      fi
      if   [ "$state" = "running" ] && [ "$ssh_st" = "ready" ]; then
        readiness="ready"
      elif [ "$state" = "running" ] && [ -n "$ip" ]; then
        readiness="waiting-ssh"
      elif [ "$state" = "running" ]; then
        readiness="booting"
      else
        readiness="creating"
      fi
      pline "$(printf '  %-22s' "$dom")  $(cf "$state" 10)  $(printf '%-18s' "${mac:-n/a}")  $(printf '%-15s' "${ip:-pending}")  $(cf "$ssh_st" 8)  $(cf "$readiness")"
    done <<< "$domains"
  fi
  pline ""
}

render_artifacts() {
  section "ARTIFACTS"
  pline "$(printf '  %-36s  %-8s  %s' 'FILE' 'EXISTS' 'SIZE')"
  local found=false
  for iso in "${STACK}"-*-seed.iso; do
    [ -f "$iso" ] || continue
    found=true
    local sz
    sz=$(du -sh "$iso" 2>/dev/null | cut -f1 || echo "?")
    pline "$(printf '  %-36s  ' "$iso")$(cf yes 8)  $sz"
  done
  if [ -d "logs" ]; then
    for logf in logs/*.log; do
      [ -f "$logf" ] || continue
      found=true
      local sz
      sz=$(du -sh "$logf" 2>/dev/null | cut -f1 || echo "?")
      pline "$(printf '  %-36s  ' "$logf")$(cf yes 8)  $sz"
    done
  fi
  $found || pline "  (no artifacts yet)"
  pline ""
}

render_events() {
  section "EVENTS (last ${MAX_EVENTS})"
  if [ -f "$EVENTS_FILE" ]; then
    local count=0
    while IFS= read -r ev; do
      # Color: dim prefix, cyan [source] tag, normal message.
      # Expected format: "EVENT HH:MM:SS [source] message"
      # Fall back to plain display if the format is unexpected.
      local prefix tag msg rest
      if [[ "$ev" == *'['*']'* ]]; then
        prefix="${ev%%\[*}"
        rest="${ev#*\[}"
        tag="[${rest%%\]*}]"
        msg="${ev#*\] }"
        pline "  ${C_DIM}${prefix}${C_RESET}${C_CYAN}${tag}${C_RESET} ${msg}"
      else
        pline "  $ev"
      fi
      count=$((count + 1))
    done < <(tail -n "$MAX_EVENTS" "$EVENTS_FILE")
    [ "$count" -eq 0 ] && pline "  (no events yet)"
  else
    pline "  (no events yet)"
  fi
  pline ""
}

# ── main loop ─────────────────────────────────────────────────────────────────

hide_cursor
printf '\033[s'   # save cursor position at start of dashboard
PREV_LINES=0

while true; do
  [ "$PREV_LINES" -gt 0 ] && printf '\033[u\033[J'   # restore + clear to end
  TOTAL_LINES=0

  pline "$(printf '%s== nlab  stack=%-12s  %s ==%s' \
    "${C_BOLD}${C_CYAN}" "$STACK" "$(date '+%H:%M:%S')" "$C_RESET")"
  pline ""
  render_keys
  render_networks
  render_vms
  render_artifacts
  render_events

  PREV_LINES=$TOTAL_LINES
  sleep "$REFRESH"
done
