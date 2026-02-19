package lab

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	libvirtURI = "qemu:///system"
	baseImage  = "/var/lib/libvirt/images/ubuntu-base.qcow2"
	sshUser    = "ubuntu"
)

// VMConfig holds the parameters needed to create one VM.
type VMConfig struct {
	Stack   string
	Role    string
	Memory  int // MiB
	VCPUs   int
	Network string
}

// CreateVM provisions a VM from the base cloud image using virt-install and
// cloud-init.
func CreateVM(cfg VMConfig) error {
	name := cfg.Stack + "-" + cfg.Role
	seed := name + "-seed.iso"
	pubKeyFile := fmt.Sprintf("keys/%s/id_ed25519.pub", cfg.Stack)
	userDataTpl := fmt.Sprintf("stacks/%s/%s/user-data", cfg.Stack, cfg.Role)
	metaData := fmt.Sprintf("stacks/%s/%s/meta-data", cfg.Stack, cfg.Role)
	tmpUserData := fmt.Sprintf("/tmp/%s-user-data", name)

	if _, err := os.Stat(baseImage); err != nil {
		return fmt.Errorf("base image not found at %s – run 'nlab image download' first", baseImage)
	}
	if _, err := os.Stat(pubKeyFile); err != nil {
		return fmt.Errorf("SSH public key not found at %s – run 'nlab key generate %s' first", pubKeyFile, cfg.Stack)
	}

	if DomainExists(name) {
		Skip(fmt.Sprintf("VM %s already exists", name))
		return nil
	}

	if err := prepareCloudInit(userDataTpl, metaData, pubKeyFile, tmpUserData, seed, name); err != nil {
		return err
	}
	return installVM(cfg, name, seed)
}

// DestroyVM stops and undefines a VM and removes its storage.
func DestroyVM(stack, role string) error {
	name := stack + "-" + role
	seed := name + "-seed.iso"

	Info(fmt.Sprintf("Destroy request: %s", name))

	if DomainExists(name) {
		Info(fmt.Sprintf("Stopping %s (if running)", name))
		_ = virsh("destroy", name)
		Info(fmt.Sprintf("Undefining %s (and removing storage)", name))
		if err := virsh("undefine", name, "--remove-all-storage"); err != nil {
			return fmt.Errorf("undefine %s: %w", name, err)
		}
	} else {
		Skip(fmt.Sprintf("Domain %s not found (already gone)", name))
	}

	_ = os.Remove(seed)

	if DomainExists(name) {
		return fmt.Errorf("FAILED: %s still exists after destroy", name)
	}

	Ok(fmt.Sprintf("%s deleted", name))
	return nil
}

// DomainExists reports whether a libvirt domain is defined.
func DomainExists(name string) bool {
	cmd := virshCmd("dominfo", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// DomainState returns the running state reported by virsh.
func DomainState(name string) string {
	out, err := virshCmd("domstate", name).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// DomainMAC returns the first MAC address for a domain's network interface.
func DomainMAC(name string) string {
	out, err := virshCmd("domiflist", name).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "network") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				return fields[4]
			}
		}
	}
	return ""
}

// DHCPLeaseIP looks up the IP for a MAC address in the named network's DHCP leases.
func DHCPLeaseIP(network, mac string) string {
	out, err := virshCmd("net-dhcp-leases", network).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, mac) {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.Contains(f, ".") {
					return strings.SplitN(f, "/", 2)[0]
				}
			}
		}
	}
	return ""
}

func prepareCloudInit(userDataTpl, metaData, pubKeyFile, tmpUserData, seed, name string) error {
	pubKey, err := os.ReadFile(pubKeyFile)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	tplBytes, err := os.ReadFile(userDataTpl)
	if err != nil {
		return fmt.Errorf("read user-data template: %w", err)
	}
	rendered := strings.ReplaceAll(string(tplBytes), "__SSH_PUBLIC_KEY__", strings.TrimSpace(string(pubKey)))
	if err := os.WriteFile(tmpUserData, []byte(rendered), 0o600); err != nil {
		return fmt.Errorf("write temp user-data: %w", err)
	}
	Info(fmt.Sprintf("Creating cloud-init ISO for %s", name))
	cmd := exec.Command("cloud-localds", seed, tmpUserData, metaData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmpUserData)
		return fmt.Errorf("cloud-localds: %w", err)
	}
	_ = os.Remove(tmpUserData)
	return nil
}

func installVM(cfg VMConfig, name, seed string) error {
	Info(fmt.Sprintf("Installing VM %s", name))
	cmd := exec.Command("virt-install",
		"--connect", libvirtURI,
		"--name", name,
		"--memory", fmt.Sprintf("%d", cfg.Memory),
		"--vcpus", fmt.Sprintf("%d", cfg.VCPUs),
		"--disk", fmt.Sprintf("size=20,backing_store=%s,format=qcow2", baseImage),
		"--disk", fmt.Sprintf("path=%s,device=cdrom,readonly=on", seed),
		"--os-variant", "ubuntu22.04",
		"--network", fmt.Sprintf("network=%s", cfg.Network),
		"--graphics", "none",
		"--import",
		"--noautoconsole",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("virt-install: %w", err)
	}
	Ok(fmt.Sprintf("VM %s deployed", name))
	return nil
}

// virsh runs a virsh subcommand against qemu:///system with output attached.
func virsh(args ...string) error {
	cmd := virshCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// virshCmd builds a virsh *exec.Cmd without attaching output.
func virshCmd(args ...string) *exec.Cmd {
	full := append([]string{"--connect", libvirtURI}, args...)
	return exec.Command("virsh", full...)
}
