#!/bin/bash
# Shared helpers sourced by nlab scripts.

export LIBVIRT_DEFAULT_URI=qemu:///system

log_info()  { echo "[+] $*"; }
log_ok()    { echo "[✓] $*"; }
log_skip()  { echo "[=] $*"; }
log_error() { echo "[!] $*" >&2; }
