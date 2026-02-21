# nlab Install Guide (Ubuntu)

This guide covers installing nlab on Ubuntu, verifying prerequisites, and understanding how nlab stores its data.

---

## Prerequisites

nlab requires the following packages on the **host** system:

| Dependency | Purpose | Install |
|---|---|---|
| `qemu-kvm` | KVM hypervisor | `sudo apt install qemu-kvm` |
| `libvirt-daemon-system` | libvirt daemon | `sudo apt install libvirt-daemon-system` |
| `libvirt-clients` / `virsh` | libvirt CLI | `sudo apt install libvirt-clients` |
| `virtinst` / `virt-install` | VM provisioning | `sudo apt install virtinst` |
| `cloud-image-utils` | Builds cloud-init seed ISOs | `sudo apt install cloud-image-utils` |
| `tmux` | Terminal multiplexer | `sudo apt install tmux` |
| `tcpdump` | Packet capture | `sudo apt install tcpdump` |
| Go ≥ 1.21 | Build nlab (not needed at runtime) | https://go.dev/dl/ |

Install everything at once:

```bash
sudo apt update
sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients virtinst \
    cloud-image-utils tmux tcpdump
```

### User group membership

Add your user to the `libvirt` and `kvm` groups, then log out and back in:

```bash
sudo usermod -aG libvirt,kvm "$USER"
# Log out and log back in (or newgrp libvirt)
```

---

## Install nlab

### Option A — Build and install with `make install`

```bash
git clone https://github.com/h3ow3d/nlab.git
cd nlab
make install
```

This builds nlab and places the binary at `~/.local/bin/nlab`.

If `~/.local/bin` is not yet in your `PATH`, follow the hint printed by `make install`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Option B — Build manually

```bash
go build -o ~/.local/bin/nlab ./cmd/nlab
```

---

## Verify the install

```bash
nlab version   # prints the nlab version string
nlab doctor    # checks all host prerequisites
```

`nlab doctor` output looks like this on a healthy system:

```
[✓] virsh: /usr/bin/virsh found
[✓] libvirt connectivity: connected to qemu:///system
[✓] qemu/kvm: /dev/kvm is accessible
[✓] tmux: /usr/bin/tmux found
[✓] tcpdump: /usr/sbin/tcpdump found
[✓] XDG directory access: XDG dirs ready (config=... data=... state=...)
```

If any check fails, `nlab doctor` prints the failure and a suggested fix.
The command exits with a non-zero status so it can be used in scripts.

---

## Filesystem layout (XDG)

nlab follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) for single-user installs:

| Purpose | Default path | Override |
|---|---|---|
| Binary | `~/.local/bin/nlab` | `$PATH` |
| Config file | `~/.config/nlab/config.yaml` | `$XDG_CONFIG_HOME` |
| Base image cache | `~/.local/share/nlab/images/` | `$XDG_DATA_HOME` |
| Stacks library | `~/.local/share/nlab/stacks/` | `$XDG_DATA_HOME` |
| Cloud-init seeds | `~/.local/share/nlab/cloudinit/` | `$XDG_DATA_HOME` |
| Logs | `~/.local/state/nlab/logs/` | `$XDG_STATE_HOME` |
| Packet captures | `~/.local/state/nlab/pcap/` | `$XDG_STATE_HOME` |

nlab creates all required directories on first use (with mode `0700`).

### Overriding paths

Set the standard XDG environment variables before running nlab:

```bash
export XDG_DATA_HOME=/mnt/bigdisk/.local/share
nlab image download   # images go to /mnt/bigdisk/.local/share/nlab/images/
```

---

## tcpdump privilege model

By default, `tcpdump` requires `sudo` or `CAP_NET_RAW`.  To avoid repeated
`sudo` prompts during lab sessions, Ubuntu provides a guided approach:

### Option 1 — Add yourself to the `pcap` group (recommended)

```bash
sudo groupadd -f pcap
sudo chown root:pcap /usr/sbin/tcpdump
sudo chmod 750 /usr/sbin/tcpdump
sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump
sudo usermod -aG pcap "$USER"
# Log out and back in
```

This grants packet capture capability to members of the `pcap` group without
making the binary world-executable or weakening overall host security.

### Option 2 — Use `sudo` (simpler, less automated)

Leave `tcpdump` as-is.  nlab stack layouts that invoke tcpdump will prefix the
command with `sudo`.  You will be prompted for your password on first use per
session (standard `sudo` TTY caching applies).

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `nlab doctor` reports virsh not found | `libvirt-clients` not installed | `sudo apt install libvirt-clients` |
| `nlab doctor` reports cannot connect to qemu:///system | libvirtd not running or wrong group | `sudo systemctl start libvirtd` and add user to `libvirt` group |
| `nlab doctor` reports /dev/kvm not accessible | KVM not enabled or wrong group | Enable VT-x/AMD-V in BIOS; `sudo usermod -aG kvm "$USER"` |
| `nlab: command not found` | `~/.local/bin` not in `PATH` | Add `export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc` |
| Permission denied writing to XDG dirs | Home directory issue | Check disk space and `ls -la ~` |
