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

Install on Ubuntu / Debian:

```bash
sudo apt install qemu-kvm libvirt-daemon-system virtinst cloud-image-utils tmux wget
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

`make basic` will:
1. Generate a per-stack ed25519 SSH key under `keys/basic/`
2. Create the isolated `basic_net` libvirt network (`10.10.10.0/24`)
3. Provision **basic-attacker** (4 GB RAM, 2 vCPUs — nmap, tcpdump, curl)
4. Provision **basic-target** (2 GB RAM, 2 vCPUs — apache2)
5. Wait for both VMs to become SSH-reachable, then open a tmux session

### tmux Layout

```
┌─────────────────────┬──────────────────────┐
│  attacker (SSH)     │  tcpdump on virbr-   │
├─────────────────────│  basic               │
│  target   (SSH)     │                      │
└─────────────────────┴──────────────────────┘
```

---

## Repository Layout

```
nlab/
├── images/
│   └── download_base.sh      # Downloads Ubuntu 22.04 cloud image
├── keys/                     # Per-stack SSH key pairs (git-ignored)
├── scripts/
│   ├── create-network.sh     # Define & start a libvirt network
│   ├── create-vm.sh          # Provision a VM from the base image
│   ├── destroy-network.sh    # Stop & undefine a libvirt network
│   ├── destroy-vm.sh         # Destroy a VM and remove its storage
│   ├── generate-key.sh       # Generate a per-stack ed25519 key pair
│   └── launch-tmux.sh        # Wait for VMs then open tmux session
└── stacks/
    └── basic/
        ├── stack.mk           # Make targets: basic / basic-destroy
        ├── network.xml        # Libvirt network definition
        ├── attacker/
        │   ├── meta-data      # cloud-init meta-data
        │   └── user-data      # cloud-init user-data template
        └── target/
            ├── meta-data
            └── user-data
```

---

## Adding a New Stack

1. Create `stacks/<name>/` with `network.xml`, `stack.mk`, and
   `attacker/` + `target/` cloud-init directories.
2. Use `__SSH_PUBLIC_KEY__` as the placeholder in `user-data` — it is
   substituted at VM creation time with the stack's public key.
3. Include `stacks/<name>/stack.mk` automatically via the top-level
   `Makefile` wildcard.
