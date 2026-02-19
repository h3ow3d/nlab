#!/bin/bash
# Shared helpers sourced by nlab scripts.

export LIBVIRT_DEFAULT_URI=qemu:///system

log_info()  { echo "[+] $*"; }
log_ok()    { echo "[✓] $*"; }
log_skip()  { echo "[=] $*"; }
log_error() { echo "[!] $*" >&2; }

# emit_event <stack> <source> <message>
# Appends a timestamped EVENT line to logs/<stack>-events.log.
# Does not write to stdout/stderr so it is safe inside redirected subshells.
emit_event() {
  local stack="$1" source="$2" msg="$3"
  mkdir -p logs
  printf 'EVENT %s [%s] %s\n' "$(date '+%H:%M:%S')" "$source" "$msg" \
    >> "logs/${stack}-events.log"
}
