# nlab Architecture (Target / End-State) — Opinionated, Batteries-Included

## Product intent (gist)

nlab is a **single-user** tool for Ubuntu that creates and operates **self-contained lab stacks** on the **local libvirt/KVM host**. It is intentionally **opinionated**: nlab should “just work” out of the box with minimal user ceremony, while still exposing underlying primitives (libvirt XML, virsh) so users can learn.

---

## Goals

- **One command** to deploy a lab from a **single YAML manifest**.
- Stacks are **self-contained** (no cross-stack wiring required).
- nlab **manages storage** (images/overlays/paths) with safe defaults.
- nlab prefers **cloud images + cloud-init**, and can enable **qemu-guest-agent** automatically when possible.
- Provide a stable operational CLI and a **k9s-like TUI**.
- Provide tmux and tcpdump workflows for observing network behavior.

---

## Non-goals (v1)

- Remote libvirt (may be added later).
- Multi-user tenancy / RBAC.
- Full YAML→XML generation (v1 uses **XML-in-YAML**).
- TUI for applying manifests (TUI is ops-focused; manifests remain CLI).

---

## Primary design decisions (hard requirements)

### D1) TUI shells out to CLI JSON (hard contract)
- `nlab tui` MUST execute `nlab ... --json` commands and parse JSON.
- TUI MUST NOT call internal engine/provider packages directly.
- TUI MUST NOT parse human-formatted output.

Why: keeps CLI stable/scriptable and mirrors k9s behavior.

### D2) Single manifest is the “golden path”
The standard workflow is a **single stack YAML** that includes **top-level** `networks:` and `vms:` keys (plus stack-level config like storage, tmux layout).

nlab may later support multi-file bundles, but v1 docs and UX optimize for:
- `nlab apply -f stack.yaml`

### D3) VM is the user-facing term (libvirt “domain” internally)
- Manifest uses `vms:` and `kind: VM` (user friendly).
- Provider may still use libvirt terminology internally (domain operations).

### D4) XML-in-YAML for libvirt resources (learning + flexibility)
VMs and networks are specified using libvirt XML embedded in YAML (or as file references later).
This keeps the full power of libvirt available and teaches users the native format.

### D5) Batteries-included storage management
nlab owns a predictable on-disk layout and can:
- fetch base images (optional but recommended)
- create qcow2 overlays per VM
- keep images cached
- delete storage safely (default safe; `--purge` to remove)

### D6) Cloud-init first; guest agent enabled when possible
For supported cloud images, nlab will generate/provide cloud-init to:
- ensure qemu-guest-agent is installed and enabled (when image supports apt)
- configure users/ssh keys (optional)
- set hostname, etc.

If cloud-init/agent is unavailable, nlab degrades gracefully and clearly reports limitations (e.g., IP discovery).

### D7) tmux layout is part of the stack manifest
The stack manifest defines the tmux experience:
- either a preset layout with options, or a declarative window/pane spec (see below).

### D8) tcpdump should be seamless and safe
Aim for “configure once, run without sudo prompts” on Ubuntu:
- either via guided setup (one-time) or controlled capability approach
- without weakening host security unnecessarily.

---

## High-level component diagram

```mermaid
flowchart LR
  U[User] -->|writes/edits| MAN[stack.yaml]
  U -->|runs| CLI[nlab CLI]
  U -->|runs| TUI[nlab tui]

  CLI --> L[Manifest Loader + Validator]
  MAN --> L

  L --> ENG[Engine: apply/delete/get]
  ENG --> IMG[Image+Storage Manager]
  ENG --> P[Provider: libvirt (virsh initially)]
  P <--> LV[(libvirtd local)]

  P --> STAT[Status Collectors]
  STAT --> CLI

  TUI -->|subprocess| CLIJSON[nlab ... --json]
  CLIJSON --> CLI

  TUI -->|hotkey T| TMUX[tmux: nlab:<stack>]
  TUI -->|hotkey P| TCPD[tcpdump live view / optional pcap]
```

---

## Repository layout (recommended target)

```
cmd/nlab/                      # main entrypoint
internal/manifest/             # YAML stack parsing + validation
internal/types/                # typed model: StackSpec, NetworkSpec, VMSpec, tmux spec, etc.
intern...