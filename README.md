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
make build        # or: go build -o nlab ./cmd/nlab

# 2. Download the Ubuntu base image (one-time)
./nlab image download

# 3. Bring up the "basic" stack (attacker + target)
make up           # or: ./nlab up basic

# 4. Tear everything down when finished
make down         # or: ./nlab down basic
```

To use the **template** stack instead:

```bash
make up STACK=template    # bring up template stack
make down STACK=template  # tear it down
```

`nlab up basic` will:
1. Generate a per-stack ed25519 SSH key under `keys/basic/`
2. Create the isolated `basic_net` libvirt network (`10.10.10.0/24`)
3. Provision **basic-attacker** (4 GB RAM, 2 vCPUs — nmap, tcpdump, curl)
4. Provision **basic-target** (2 GB RAM, 2 vCPUs — apache2)
5. Show a live dashboard while VMs boot, then open a tmux session

---

## Command Reference

| Command | Description |
|---|---|
| `nlab image download` | Download the Ubuntu 22.04 base cloud image |
| `nlab key generate <stack>` | Generate a per-stack ed25519 SSH key pair |
| `nlab network create <stack>` | Define and start the libvirt network |
| `nlab network destroy <stack>` | Stop and undefine the libvirt network |
| `nlab vm create <stack> <role>` | Provision a single VM |
| `nlab vm destroy <stack> <role>` | Destroy a single VM and remove its storage |
| `nlab session <stack>` | Wait for SSH readiness then open tmux session |
| `nlab dashboard <stack>` | Show the live creation dashboard |
| `nlab up <stack>` | Full stack bring-up (key + net + VMs + session) |
| `nlab down <stack>` | Full stack tear-down |
| `nlab list` | List all libvirt domains |

Use `nlab <command> --help` for detailed usage and examples.

### Examples

```bash
# Individual operations (granular control)
nlab key generate basic
nlab network create basic
nlab vm create basic attacker
nlab vm create basic target
nlab dashboard basic          # open live dashboard in separate terminal
nlab session basic            # wait for SSH then open tmux

# Tear down step-by-step
nlab vm destroy basic attacker
nlab vm destroy basic target
nlab network destroy basic

# Override VM specs at creation time
nlab vm create basic attacker --memory 8192 --vcpus 4
```

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
│   ├── dashboard.go              # Live creation dashboard
│   ├── image.go                  # Base image download + checksum
│   ├── keys.go                   # Per-stack ed25519 key generation
│   ├── layout.go                 # layout.yaml parser
│   ├── log.go                    # Shared logging helpers
│   ├── network.go                # libvirt network create / destroy
│   ├── stack.go                  # stack.yaml parser
│   ├── tmux.go                   # tmux session launcher
│   └── vm.go                     # VM create / destroy (virt-install / virsh)
├── keys/                         # Per-stack SSH key pairs (git-ignored)
└── stacks/
    ├── basic/
    │   ├── stack.yaml             # Stack config: network + VM specs
    │   ├── network.xml            # Libvirt network definition
    │   ├── layout.yaml            # tmux pane layout definition
    │   ├── attacker/
    │   │   ├── meta-data          # cloud-init meta-data
    │   │   └── user-data          # cloud-init user-data template
    │   └── target/
    │       ├── meta-data
    │       └── user-data
    └── template/
        ├── stack.yaml             # Stack config: network + VM specs
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
