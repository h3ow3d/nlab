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
internal/engine/               # orchestration (apply/delete/get/status)
internal/provider/libvirt/     # virsh runner + XML patching + discovery
internal/storage/              # base image cache, overlays, paths, purge behavior
internal/cloudinit/            # cloud-init generation + seed ISO creation
internal/cli/                  # command wiring + printers (optional split)
internal/tui/                  # bubbletea UI (later, shells out to CLI)
docs/
```

---

## Filesystem integration (single-user, Ubuntu, XDG)

Defaults (overridable):
- **Binary:** `~/.local/bin/nlab`
- **Config:** `~/.config/nlab/config.yaml`
- **Data:** `~/.local/share/nlab/`
  - base images cache: `~/.local/share/nlab/images/`
  - stacks library (optional): `~/.local/share/nlab/stacks/`
  - generated cloud-init seeds: `~/.local/share/nlab/cloudinit/`
- **State:** `~/.local/state/nlab/`
  - logs: `~/.local/state/nlab/logs/`
  - captures: `~/.local/state/nlab/pcap/`

---

## Manifest format (v1alpha1, single file)

### Shape (golden path)
A single YAML document, Kubernetes-inspired header, plus top-level keys:

- `apiVersion: nlab.io/v1alpha1`
- `kind: Stack`
- `metadata: { name, labels, annotations }`
- `spec:`
  - `networks:` map (name → network spec)
  - `vms:` map (name → vm spec)
  - `storage:` settings (base images, overlays, locations)
  - `tmux:` layout spec
  - `defaults:` convenience defaults

### VM + Network XML embedding
- `spec.networks.<name>.xml: |` (libvirt network XML)
- `spec.vms.<name>.xml: |` (libvirt domain XML)

nlab may patch/augment XML to insert:
- ownership markers
- cloud-init disk attachment
- qemu-guest-agent channel device (where appropriate)

**Important:** nlab should keep the resulting XML discoverable:
- `nlab vm dumpxml <vm>` should show effective XML
- and/or store generated XML under state for debugging.

---

## Ownership markers (safety)

nlab must mark managed libvirt resources, e.g. in `<description>`:

- `nlab.io/managed=true`
- `nlab.io/stack=<stackName>`
- `nlab.io/resource=vm|network`
- `nlab.io/name=<resourceName>`
- `nlab.io/manifest-hash=<sha256>` (optional)

Delete rules:
- `nlab delete -f stack.yaml` deletes managed resources for that stack.
- `--purge` additionally removes overlays and stack-managed volumes/images.
- Refuse to delete unmarked resources unless `--force`.

---

## Storage (batteries included)

### Baseline behavior
- nlab manages a directory-backed storage model under XDG data/state.
- For each VM, nlab can create an overlay qcow2 referencing a base image.

### Custom images
Users can customize images by:
- specifying an image source in the stack manifest (URL/path/name), OR
- placing images in the cache directory and referencing by name.

### Purge semantics
- Default delete: remove libvirt definitions/resources; keep cached base images.
- `--purge`: remove VM overlays and stack-associated artifacts (pcaps optional).

---

## Cloud-init + qemu-guest-agent

### Assumptions
- Primary supported VM source: Ubuntu-compatible cloud images.

### Behavior
- nlab generates per-VM cloud-init user-data/meta-data (or per-stack defaults).
- nlab builds a seed ISO (or equivalent) and attaches it to the VM.
- nlab attempts to ensure `qemu-guest-agent` is installed/enabled on supported images.

Graceful degradation:
- If guest agent isn’t available, nlab still functions; IP discovery and richer status may be limited.

---

## tmux integration (manifest-driven)

### Intent
`nlab stack tmux <stack>` creates/attaches `tmux` session `nlab:<stack>`.

### Manifest control
The stack manifest defines either:
- a preset layout + options (recommended for v1), and/or
- a declarative layout (later).

v1 recommended approach:
- `spec.tmux.preset: default|grid|wide`
- `spec.tmux.windows:` optional overrides (commands per pane)

---

## tcpdump integration

### Default hotkey behavior
- `P` in TUI starts **live view**.
- Users can toggle “stream to file” / pcap output (CLI flag or TUI toggle later).

### Privilege model goal
- One-time setup on Ubuntu to avoid recurring sudo prompts.
- Documented and as safe as possible:
  - prefer a guided setup command (e.g. `nlab setup`) that configures the chosen model.

---

## CLI (minimum surface)

### Manifest lifecycle
- `nlab validate -f stack.yaml`
- `nlab apply -f stack.yaml`
- `nlab delete -f stack.yaml [--purge] [--force]`

### Ops (TUI consumes these via --json)
- `nlab stack ls --json`
- `nlab stack status <stack> --json`
- `nlab stack up|down|restart <stack>`
- `nlab vm ls --stack <stack> --json`
- `nlab vm start|stop|reboot <vm>`
- `nlab vm console <vm>`

### Observability
- `nlab logs vm <vm> [--follow]`
- `nlab stack tmux <stack>`
- `nlab stack tcpdump <stack> [--network <name>] [--filter ...] [--pcap ...]`

---

## JSON output contract (for TUI)

All JSON responses consumed by the TUI MUST include:
- `apiVersion` (e.g. `nlab.io/v1alpha1`)
- `kind` (e.g. `StackList`, `StackStatus`, `VMList`)
- `generatedAt` (RFC3339 recommended)
- `errors` array recommended (empty on success)

TUI must only depend on these JSON outputs (not human output).

---

## Future scope (explicitly deferred)

- Remote libvirt URIs (`qemu+ssh://...`)
- Multi-file manifest bundles as first-class UX
- YAML-native VM/network spec that renders to XML
- Multi-user mode
