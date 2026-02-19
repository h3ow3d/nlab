# nlab — Red-Team Lab Framework

A lightweight framework for spinning up isolated virtual-machine stacks using
**libvirt / KVM** and **cloud-init**.  Each *stack* defines an attacker VM and
a target VM connected through a private NAT network so you can practice
offensive techniques in a self-contained environment.

---

## Requirements

| Dependency | Notes |
|---|---|
| `libvirt` / `virsh` | KVM virtualisation back-end |
| `virt-install` | VM provisioning tool |
| `cloud-localds` | Builds cloud-init seed ISOs (`cloud-image-utils` package) |
| `tmux` | Terminal multiplexer used by the launch script |
| `wget` | Image download |
| `ssh` | Key-based access to VMs |
| `python3-yaml` | YAML parsing for `layout.yaml` |

Install on Ubuntu / Debian:

```bash
sudo apt install qemu-kvm libvirt-daemon-system virtinst cloud-image-utils tmux wget python3-yaml
sudo usermod -aG libvirt,kvm "$USER"   # log out and back in
```

---

## Quick Start

```bash
# 1. Download the Ubuntu base image (one-time)
./images/download_base.sh

# 2. Bring up the "basic" stack (attacker + target)
make basic

# 3. Tear everything down when finished
make basic-destroy
```

To start from the **template** stack instead:

```bash
make template         # bring up template stack
make template-destroy # tear it down
```

`make basic` will:
1. Generate a per-stack ed25519 SSH key under `keys/basic/`
2. Create the isolated `basic_net` libvirt network (`10.10.10.0/24`)
3. Provision **basic-attacker** (4 GB RAM, 2 vCPUs — nmap, tcpdump, curl)
4. Provision **basic-target** (2 GB RAM, 2 vCPUs — apache2)
5. Wait for both VMs to become SSH-reachable, then open a tmux session

### tmux Layout

Each stack defines its tmux panes in `stacks/<name>/layout.yaml`.  The basic
stack ships with this layout:

```
┌─────────────────────┬──────────────────────┐
│  attacker (SSH)     │  tcpdump on virbr-   │
├─────────────────────│  basic               │
│  target   (SSH)     │                      │
└─────────────────────┴──────────────────────┘
```

`layout.yaml` format:

```yaml
layout: even-horizontal   # any tmux select-layout value
panes:
  - name: attacker        # optional label
    type: ssh             # opens an SSH session to the named VM
    vm: attacker
  - name: target
    type: ssh
    vm: target
  - name: monitor
    type: command         # runs an arbitrary shell command
    command: "sudo tcpdump -i virbr-{stack} -nn -tttt -vvv"
```

`{stack}` in a `command` value is substituted with the stack name at runtime.
Add as many panes as you like — the script will wait for every `ssh` VM to
become reachable before opening the session.

---

## Repository Layout

```
nlab/
├── images/
│   └── download_base.sh      # Downloads Ubuntu 22.04 cloud image
├── keys/                     # Per-stack SSH key pairs (git-ignored)
├── scripts/
│   ├── lib.sh                # Shared helpers (LIBVIRT_DEFAULT_URI, log_*)
│   ├── create-network.sh     # Define & start a libvirt network
│   ├── create-vm.sh          # Provision a VM from the base image
│   ├── destroy-network.sh    # Stop & undefine a libvirt network
│   ├── destroy-vm.sh         # Destroy a VM and remove its storage
│   ├── generate-key.sh       # Generate a per-stack ed25519 key pair
│   └── launch-tmux.sh        # Wait for VMs then open tmux session
└── stacks/
    ├── basic/
    │   ├── stack.mk           # Make targets: basic / basic-destroy
    │   ├── network.xml        # Libvirt network definition
    │   ├── layout.yaml        # tmux pane layout definition
    │   ├── attacker/
    │   │   ├── meta-data      # cloud-init meta-data
    │   │   └── user-data      # cloud-init user-data template
    │   └── target/
    │       ├── meta-data
    │       └── user-data
    └── template/
        ├── stack.mk           # Make targets: template / template-destroy
        ├── network.xml        # Libvirt network definition (10.10.20.0/24)
        ├── layout.yaml        # tmux pane layout definition
        ├── attacker/
        │   ├── meta-data      # cloud-init meta-data
        │   └── user-data      # cloud-init user-data (customise packages here)
        └── target/
            ├── meta-data
            └── user-data      # cloud-init user-data (customise packages here)
```

---

## Adding a New Stack

1. Create `stacks/<name>/` with `network.xml`, `stack.mk`, `layout.yaml`, and
   VM cloud-init directories for each VM in the stack.
2. Use `__SSH_PUBLIC_KEY__` as the placeholder in `user-data` — it is
   substituted at VM creation time with the stack's public key.
3. Include `stacks/<name>/stack.mk` automatically via the top-level
   `Makefile` wildcard.
4. Define panes in `layout.yaml` — add an `ssh` entry for each VM and any
   `command` entries (e.g. tcpdump, logs) you want in the session.
