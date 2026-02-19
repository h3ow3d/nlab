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
| `ssh` / `ssh-keygen` | Key-based access to VMs |
| Go ≥ 1.21 | Build the `nlab` binary (not needed at runtime) |

Install on Ubuntu / Debian:

```bash
sudo apt install qemu-kvm libvirt-daemon-system virtinst cloud-image-utils tmux
sudo usermod -aG libvirt,kvm "$USER"   # log out and back in
```

---

## Quick Start

```bash
# 1. Build the nlab CLI (one-time)
go build -o nlab ./cmd/nlab

# 2. Download the Ubuntu base image (one-time)
./nlab image download

# 3. Bring up the "basic" stack (attacker + target)
./nlab up basic

# 4. Tear everything down when finished
./nlab down basic
```

To use the **template** stack instead:

```bash
./nlab up template    # bring up template stack
./nlab down template  # tear it down
```

`nlab up basic` will:
1. Generate a per-stack ed25519 SSH key under `keys/basic/`
2. Create the isolated `basic_net` libvirt network (`10.10.10.0/24`)
3. Provision **basic-attacker** (4 GB RAM, 2 vCPUs — nmap, tcpdump, curl)
4. Provision **basic-target** (2 GB RAM, 2 vCPUs — apache2)
5. Show a live dashboard while VMs boot, then open a tmux session

### Legacy Make Targets

The original `make basic` / `make basic-destroy` targets still work and continue
to call the shell scripts in `scripts/`.  The Go CLI is the recommended interface
going forward.

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
├── cmd/
│   └── nlab/
│       └── main.go               # nlab CLI entry point (cobra subcommands)
├── internal/
│   ├── dashboard/dashboard.go    # Live creation dashboard
│   ├── image/download.go         # Base image download + checksum
│   ├── keys/keys.go              # Per-stack ed25519 key generation
│   ├── layout/layout.go          # layout.yaml parser
│   ├── log/log.go                # Shared logging helpers
│   ├── network/network.go        # libvirt network create / destroy
│   ├── stack/stack.go            # stack.yaml parser
│   ├── tmux/tmux.go              # tmux session launcher
│   └── vm/vm.go                  # VM create / destroy (virt-install / virsh)
├── images/
│   └── download_base.sh          # Legacy bash image downloader (still works)
├── keys/                         # Per-stack SSH key pairs (git-ignored)
├── scripts/
│   ├── lib.sh                    # Shared helpers (LIBVIRT_DEFAULT_URI, log_*)
│   ├── create-network.sh         # Define & start a libvirt network
│   ├── create-vm.sh              # Provision a VM from the base image
│   ├── destroy-network.sh        # Stop & undefine a libvirt network
│   ├── destroy-vm.sh             # Destroy a VM and remove its storage
│   ├── generate-key.sh           # Generate a per-stack ed25519 key pair
│   └── launch-tmux.sh            # Wait for VMs then open tmux session
└── stacks/
    ├── basic/
    │   ├── stack.yaml             # Stack config: network + VM specs for nlab CLI
    │   ├── stack.mk               # Make targets: basic / basic-destroy (legacy)
    │   ├── network.xml            # Libvirt network definition
    │   ├── layout.yaml            # tmux pane layout definition
    │   ├── attacker/
    │   │   ├── meta-data          # cloud-init meta-data
    │   │   └── user-data          # cloud-init user-data template
    │   └── target/
    │       ├── meta-data
    │       └── user-data
    └── template/
        ├── stack.yaml             # Stack config: network + VM specs for nlab CLI
        ├── stack.mk               # Make targets: template / template-destroy (legacy)
        ├── network.xml            # Libvirt network definition (10.10.20.0/24)
        ├── layout.yaml            # tmux pane layout definition
        ├── attacker/
        │   ├── meta-data
        │   └── user-data
        └── target/
            ├── meta-data
            └── user-data
```

---

## Adding a New Stack

1. Create `stacks/<name>/` with:
   - `stack.yaml` – network name and list of VMs (name, memory, vcpus)
   - `network.xml` – libvirt network definition
   - `layout.yaml` – tmux pane layout
   - A cloud-init directory for each VM (`meta-data` + `user-data`)
2. Use `__SSH_PUBLIC_KEY__` as the placeholder in `user-data` — it is
   substituted at VM creation time with the stack's public key.
3. Run `nlab up <name>` to bring up the stack.

### stack.yaml format

```yaml
network: mystack_net        # libvirt network name (matches network.xml)
vms:
  - name: attacker          # role name; must match the cloud-init directory
    memory: 4096            # MiB
    vcpus: 2
  - name: target
    memory: 2048
    vcpus: 2
```
