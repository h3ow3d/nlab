// Package network manages libvirt networks via virsh.
package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/h3ow3d/nlab/internal/log"
)

const libvirtURI = "qemu:///system"

// Create defines and starts a libvirt network from an XML file if it does not
// already exist.  It mirrors create-network.sh.
func Create(xmlPath, networkName string) error {
	if _, err := os.Stat(xmlPath); err != nil {
		return fmt.Errorf("network XML not found: %s", xmlPath)
	}

	if isDefined(networkName) {
		log.Skip(fmt.Sprintf("Network %s already defined", networkName))
	} else {
		log.Info(fmt.Sprintf("Defining network %s", networkName))
		if err := virsh("net-define", xmlPath); err != nil {
			return fmt.Errorf("net-define: %w", err)
		}
	}

	if isActive(networkName) {
		log.Skip(fmt.Sprintf("Network %s already active", networkName))
	} else {
		log.Info(fmt.Sprintf("Starting network %s", networkName))
		if err := virsh("net-start", networkName); err != nil {
			return fmt.Errorf("net-start: %w", err)
		}
	}

	if err := virsh("net-autostart", networkName); err != nil {
		return fmt.Errorf("net-autostart: %w", err)
	}

	log.Ok(fmt.Sprintf("Network %s ready", networkName))
	return nil
}

// Destroy stops and undefines a libvirt network.  It mirrors destroy-network.sh.
func Destroy(networkName string) error {
	if !isDefined(networkName) {
		log.Skip(fmt.Sprintf("Network %s does not exist", networkName))
		return nil
	}

	log.Info(fmt.Sprintf("Destroying network %s", networkName))
	// net-destroy returns an error if already inactive; ignore it.
	_ = virsh("net-destroy", networkName)
	if err := virsh("net-undefine", networkName); err != nil {
		return fmt.Errorf("net-undefine: %w", err)
	}

	log.Ok(fmt.Sprintf("Network %s removed", networkName))
	return nil
}

// isDefined reports whether a network is known to libvirt (defined or transient).
func isDefined(name string) bool {
	cmd := exec.Command("virsh", "--connect", libvirtURI, "net-info", name)
	return cmd.Run() == nil
}

// isActive reports whether the named network is currently active.
func isActive(name string) bool {
	out, err := exec.Command("virsh", "--connect", libvirtURI, "net-info", name).Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Active:") {
			return strings.Contains(line, "yes")
		}
	}
	return false
}

// virsh runs a virsh sub-command against qemu:///system.
func virsh(args ...string) error {
	full := append([]string{"--connect", libvirtURI}, args...)
	cmd := exec.Command("virsh", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
