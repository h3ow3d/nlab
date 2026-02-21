package lab

import (
	"fmt"
	"os"
	"os/exec"
)

// CheckResult holds the outcome of a single doctor check.
type CheckResult struct {
	Name     string
	OK       bool
	Message  string
	HowToFix string
}

// RunDoctorChecks performs all prerequisite checks and returns the results.
// It never returns an error itself; pass/fail is encoded in each CheckResult.
func RunDoctorChecks(dirs XDGDirs) []CheckResult {
	return []CheckResult{
		checkCommand("virsh", "virsh", "--version"),
		checkLibvirtConn(),
		checkKVM(),
		checkCommand("tmux", "tmux", "-V"),
		checkCommand("tcpdump", "tcpdump", "--version"),
		checkXDGWrite(dirs),
	}
}

// checkCommand verifies that an executable is on PATH and runs without error.
func checkCommand(name, bin string, args ...string) CheckResult {
	path, err := exec.LookPath(bin)
	if err != nil {
		return CheckResult{
			Name:     name,
			OK:       false,
			Message:  fmt.Sprintf("%s not found in PATH", bin),
			HowToFix: ubuntuInstallHint(bin),
		}
	}
	cmd := exec.Command(path, args...) //nolint:gosec // path is resolved via LookPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return CheckResult{
			Name:     name,
			OK:       false,
			Message:  fmt.Sprintf("%s found but failed: %s", bin, string(out)),
			HowToFix: ubuntuInstallHint(bin),
		}
	}
	return CheckResult{Name: name, OK: true, Message: fmt.Sprintf("%s found", path)}
}

// checkLibvirtConn verifies that virsh can contact the local libvirt daemon.
func checkLibvirtConn() CheckResult {
	const name = "libvirt connectivity"
	path, err := exec.LookPath("virsh")
	if err != nil {
		return CheckResult{
			Name:     name,
			OK:       false,
			Message:  "virsh not found; cannot check libvirt connectivity",
			HowToFix: "sudo apt install libvirt-clients",
		}
	}
	cmd := exec.Command(path, "--connect", "qemu:///system", "version") //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		return CheckResult{
			Name:    name,
			OK:      false,
			Message: fmt.Sprintf("cannot connect to qemu:///system: %s", string(out)),
			HowToFix: "Ensure libvirtd is running and your user is in the 'libvirt' group:\n" +
				"  sudo systemctl start libvirtd\n" +
				"  sudo usermod -aG libvirt \"$USER\"   # then log out and back in",
		}
	}
	return CheckResult{Name: name, OK: true, Message: "connected to qemu:///system"}
}

// checkKVM verifies that /dev/kvm exists and is accessible.
func checkKVM() CheckResult {
	const name = "qemu/kvm"
	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		return CheckResult{
			Name:     name,
			OK:       false,
			Message:  "/dev/kvm not found; KVM may not be available",
			HowToFix: "Ensure your CPU supports virtualisation and it is enabled in BIOS/UEFI.\nInstall: sudo apt install qemu-kvm",
		}
	}
	// Check that the current user can open the device.
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return CheckResult{
			Name:    name,
			OK:      false,
			Message: fmt.Sprintf("/dev/kvm exists but is not accessible: %v", err),
			HowToFix: "Add your user to the 'kvm' group:\n" +
				"  sudo usermod -aG kvm \"$USER\"   # then log out and back in",
		}
	}
	f.Close()
	return CheckResult{Name: name, OK: true, Message: "/dev/kvm is accessible"}
}

// checkXDGWrite verifies that nlab can write to all required XDG directories.
func checkXDGWrite(dirs XDGDirs) CheckResult {
	const name = "XDG directory access"
	if err := dirs.EnsureDirs(); err != nil {
		return CheckResult{
			Name:     name,
			OK:       false,
			Message:  fmt.Sprintf("cannot create nlab directories: %v", err),
			HowToFix: "Check that your home directory is writable and you have sufficient disk space.",
		}
	}
	return CheckResult{
		Name:    name,
		OK:      true,
		Message: fmt.Sprintf("XDG dirs ready (config=%s data=%s state=%s)", dirs.Config, dirs.Data, dirs.State),
	}
}

// ubuntuInstallHint returns a human-friendly install hint for a known binary.
func ubuntuInstallHint(bin string) string {
	hints := map[string]string{
		"virsh":   "sudo apt install libvirt-clients",
		"tmux":    "sudo apt install tmux",
		"tcpdump": "sudo apt install tcpdump",
	}
	if hint, ok := hints[bin]; ok {
		return hint
	}
	return fmt.Sprintf("Install %q and ensure it is on your PATH.", bin)
}
