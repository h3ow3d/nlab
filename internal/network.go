package lab

import (
	"fmt"
	"os"
	"strings"
)

// CreateNetwork defines and starts a libvirt network from an XML file.
func CreateNetwork(xmlPath, networkName string) error {
	if _, err := os.Stat(xmlPath); err != nil {
		return fmt.Errorf("network XML not found: %s", xmlPath)
	}

	if networkDefined(networkName) {
		Skip(fmt.Sprintf("Network %s already defined", networkName))
	} else {
		Info(fmt.Sprintf("Defining network %s", networkName))
		if err := virsh("net-define", xmlPath); err != nil {
			return fmt.Errorf("net-define: %w", err)
		}
	}

	if networkActive(networkName) {
		Skip(fmt.Sprintf("Network %s already active", networkName))
	} else {
		Info(fmt.Sprintf("Starting network %s", networkName))
		if err := virsh("net-start", networkName); err != nil {
			return fmt.Errorf("net-start: %w", err)
		}
	}

	if err := virsh("net-autostart", networkName); err != nil {
		return fmt.Errorf("net-autostart: %w", err)
	}

	Ok(fmt.Sprintf("Network %s ready", networkName))
	return nil
}

// DestroyNetwork stops and undefines a libvirt network.
func DestroyNetwork(networkName string) error {
	if !networkDefined(networkName) {
		Skip(fmt.Sprintf("Network %s does not exist", networkName))
		return nil
	}

	Info(fmt.Sprintf("Destroying network %s", networkName))
	_ = virsh("net-destroy", networkName)
	if err := virsh("net-undefine", networkName); err != nil {
		return fmt.Errorf("net-undefine: %w", err)
	}

	Ok(fmt.Sprintf("Network %s removed", networkName))
	return nil
}

func networkDefined(name string) bool {
	return virshCmd("net-info", name).Run() == nil
}

func networkActive(name string) bool {
	out, err := virshCmd("net-info", name).Output()
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
