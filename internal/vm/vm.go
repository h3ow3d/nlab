// Package vm creates and destroys libvirt virtual machines.
package vm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/h3ow3d/nlab/internal/log"
)

const (
	libvirtURI = "qemu:///system"
	baseImage  = "/var/lib/libvirt/images/ubuntu-base.qcow2"
	sshUser    = "ubuntu"
)

// Config holds the parameters needed to create one VM.
type Config struct {
	Stack   string
	Role    string
	Memory  int // MiB
	VCPUs   int
	Network string
}

// Create provisions a VM from the base cloud image using virt-install and
// cloud-init.  It mirrors create-vm.sh.
func Create(cfg Config) error {
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
		return fmt.Errorf("SSH public key not found at %s – run 'nlab up %s' which calls key generation", pubKeyFile, cfg.Stack)
	}

	if domainExists(name) {
		log.Skip(fmt.Sprintf("VM %s already exists", name))
		return nil
	}

	if err := prepareCloudInit(userDataTpl, metaData, pubKeyFile, tmpUserData, seed, name, cfg.Stack); err != nil {
		return err
	}

	if err := installVM(cfg, name, seed); err != nil {
		return err
	}

	log.Ok(fmt.Sprintf("VM %s deployed", name))
	return nil
}

// Destroy stops and undefines a VM and removes its storage.  It mirrors
// destroy-vm.sh.
func Destroy(stack, role string) error {
	name := stack + "-" + role
	seed := name + "-seed.iso"

	log.Info(fmt.Sprintf("Destroy request: %s", name))

	if domainExists(name) {
		log.Info(fmt.Sprintf("Stopping %s (if running)", name))
		_ = virsh("destroy", name) // ignore "domain not running" errors

		log.Info(fmt.Sprintf("Undefining %s (and removing storage)", name))
		if err := virsh("undefine", name, "--remove-all-storage"); err != nil {
			return fmt.Errorf("undefine %s: %w", name, err)
		}
	} else {
		log.Skip(fmt.Sprintf("Domain %s not found (already gone)", name))
	}

	_ = os.Remove(seed)

	if domainExists(name) {
		return fmt.Errorf("FAILED: %s still exists after destroy", name)
	}

	log.Ok(fmt.Sprintf("%s deleted", name))
	return nil
}

// DomainExists reports whether a libvirt domain is defined.
func DomainExists(name string) bool { return domainExists(name) }

// DomainState returns the running state string reported by virsh.
func DomainState(name string) string {
	out, err := exec.Command("virsh", "--connect", libvirtURI, "domstate", name).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// DomainMAC returns the first MAC address attached to network interface of a domain.
func DomainMAC(name string) string {
	out, err := exec.Command("virsh", "--connect", libvirtURI, "domiflist", name).Output()
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
	out, err := exec.Command("virsh", "--connect", libvirtURI, "net-dhcp-leases", network).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, mac) {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.Contains(f, ".") {
					// Strip CIDR prefix (e.g. 10.10.10.101/24)
					return strings.SplitN(f, "/", 2)[0]
				}
			}
		}
	}
	return ""
}

// prepareCloudInit generates the cloud-init ISO for the VM.
func prepareCloudInit(userDataTpl, metaData, pubKeyFile, tmpUserData, seed, name, stack string) error {
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

	log.Info(fmt.Sprintf("Creating cloud-init ISO for %s", name))
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

// installVM runs virt-install to provision the VM.
func installVM(cfg Config, name, seed string) error {
	log.Info(fmt.Sprintf("Installing VM %s", name))
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
	return cmd.Run()
}

// domainExists reports whether a libvirt domain is registered.
func domainExists(name string) bool {
	cmd := exec.Command("virsh", "--connect", libvirtURI, "dominfo", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// virsh runs a virsh subcommand.
func virsh(args ...string) error {
	full := append([]string{"--connect", libvirtURI}, args...)
	cmd := exec.Command("virsh", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
