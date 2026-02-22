// Package libvirt implements engine.Provider using local virsh commands.
package libvirt

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/h3ow3d/nlab/internal/engine"
)

const libvirtURI = "qemu:///system"

// Provider implements engine.Provider by shelling out to virsh.
type Provider struct{}

// New returns a Provider connected to qemu:///system.
func New() *Provider { return &Provider{} }

// ── Network operations ─────────────────────────────────────────────────────────

// NetworkDefined reports whether the named network is defined in libvirt.
func (p *Provider) NetworkDefined(name string) bool {
	return virshCmd("net-info", name).Run() == nil
}

// NetworkActive reports whether the named network is currently active.
func (p *Provider) NetworkActive(name string) bool {
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

// NetworkMarkers returns the nlab ownership markers for the named network by
// calling virsh net-dumpxml and parsing the <description> element.
func (p *Provider) NetworkMarkers(name string) (engine.Markers, error) {
	out, err := virshCmd("net-dumpxml", name).Output()
	if err != nil {
		return engine.Markers{}, fmt.Errorf("net-dumpxml %s: %w", name, err)
	}
	return engine.ParseMarkersFromXML(string(out)), nil
}

// DefineNetwork writes xmlStr to a temp file and calls virsh net-define.
func (p *Provider) DefineNetwork(xmlStr string) error {
	return withTempXML(xmlStr, func(path string) error {
		return virsh("net-define", path)
	})
}

// StartNetwork calls virsh net-start.
func (p *Provider) StartNetwork(name string) error { return virsh("net-start", name) }

// AutostartNetwork calls virsh net-autostart.
func (p *Provider) AutostartNetwork(name string) error { return virsh("net-autostart", name) }

// StopNetwork calls virsh net-destroy (force-stops the network).
func (p *Provider) StopNetwork(name string) error { return virsh("net-destroy", name) }

// UndefineNetwork calls virsh net-undefine.
func (p *Provider) UndefineNetwork(name string) error { return virsh("net-undefine", name) }

// ── Domain operations ──────────────────────────────────────────────────────────

// DomainDefined reports whether the named domain is defined in libvirt.
func (p *Provider) DomainDefined(name string) bool {
	cmd := virshCmd("dominfo", name)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// DomainActive reports whether the named domain is currently running.
func (p *Provider) DomainActive(name string) bool {
	out, err := virshCmd("domstate", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "running"
}

// DomainMarkers returns the nlab ownership markers for the named domain by
// calling virsh dumpxml and parsing the <description> element.
func (p *Provider) DomainMarkers(name string) (engine.Markers, error) {
	out, err := virshCmd("dumpxml", name).Output()
	if err != nil {
		return engine.Markers{}, fmt.Errorf("dumpxml %s: %w", name, err)
	}
	return engine.ParseMarkersFromXML(string(out)), nil
}

// DefineDomain writes xmlStr to a temp file and calls virsh define.
func (p *Provider) DefineDomain(xmlStr string) error {
	return withTempXML(xmlStr, func(path string) error {
		return virsh("define", path)
	})
}

// StartDomain calls virsh start.
func (p *Provider) StartDomain(name string) error { return virsh("start", name) }

// StopDomain calls virsh destroy (force-stop; equivalent to power-off).
func (p *Provider) StopDomain(name string) error { return virsh("destroy", name) }

// UndefineDomain calls virsh undefine, adding --remove-all-storage when purge
// is true.
func (p *Provider) UndefineDomain(name string, purge bool) error {
	args := []string{"undefine", name}
	if purge {
		args = append(args, "--remove-all-storage")
	}
	return virsh(args...)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// withTempXML writes xmlStr to a temporary file, calls fn with the file path,
// then removes the file.
func withTempXML(xmlStr string, fn func(string) error) error {
	tmp, err := os.CreateTemp("", "nlab-*.xml")
	if err != nil {
		return fmt.Errorf("create temp XML file: %w", err)
	}
	path := tmp.Name()
	defer os.Remove(path)

	if _, err := tmp.WriteString(xmlStr); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp XML file: %w", err)
	}
	tmp.Close()
	return fn(path)
}

// virsh runs a virsh subcommand against qemu:///system with stdout/stderr
// attached to the process output.
func virsh(args ...string) error {
	cmd := virshCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// virshCmd builds a virsh *exec.Cmd without attaching output streams.
func virshCmd(args ...string) *exec.Cmd {
	full := append([]string{"--connect", libvirtURI}, args...)
	return exec.Command("virsh", full...)
}
