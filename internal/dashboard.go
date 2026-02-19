package lab

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	dashMaxEvents = 8
	lineWidth     = 70
)

// RunDashboard renders a refreshing dashboard for the given stack and network
// until the done channel is closed.
func RunDashboard(stack, network string, done <-chan struct{}) {
	vmSSH := make(map[string]bool)
	prevLines := 0

	hideCursor()
	defer showCursor()

	for {
		select {
		case <-done:
			fmt.Printf("\033[%dB\n", prevLines)
			return
		default:
		}

		if prevLines > 0 {
			fmt.Printf("\033[%dF", prevLines)
		}

		lines := renderDashboard(stack, network, vmSSH)
		for _, l := range lines {
			fmt.Printf("%s\033[K\n", l)
		}
		prevLines = len(lines)
		time.Sleep(time.Second)
	}
}

func renderDashboard(stack, network string, vmSSH map[string]bool) []string {
	var out []string
	out = append(out, fmt.Sprintf("== nlab  stack=%-12s  %s ==",
		stack, time.Now().Format("15:04:05")))
	out = append(out, "")
	out = append(out, renderDashKeys(stack)...)
	out = append(out, renderDashNetworks(network)...)
	out = append(out, renderDashVMs(stack, network, vmSSH)...)
	out = append(out, renderDashArtifacts(stack)...)
	out = append(out, renderDashEvents(stack)...)
	return out
}

func dashSection(title string) string {
	prefix := "-- " + title + " "
	padLen := lineWidth - len(prefix)
	if padLen < 1 {
		padLen = 1
	}
	return prefix + strings.Repeat("-", padLen)
}

func renderDashKeys(stack string) []string {
	var out []string
	out = append(out, dashSection("KEYS"))
	out = append(out, fmt.Sprintf("  %-38s  %-8s  %s", "FILE", "STATUS", "FINGERPRINT"))
	priv := filepath.Join("keys", stack, "id_ed25519")
	pub := priv + ".pub"
	if _, err := os.Stat(priv); err == nil {
		fp := keyFingerprint(priv)
		out = append(out, fmt.Sprintf("  %-38s  %-8s  %s", priv, "exists", fp))
	} else {
		out = append(out, fmt.Sprintf("  %-38s  %s", priv, "missing"))
	}
	if _, err := os.Stat(pub); err == nil {
		out = append(out, fmt.Sprintf("  %-38s  %s", pub, "exists"))
	} else {
		out = append(out, fmt.Sprintf("  %-38s  %s", pub, "missing"))
	}
	out = append(out, "")
	return out
}

func renderDashNetworks(network string) []string {
	var out []string
	out = append(out, dashSection("NETWORKS"))
	out = append(out, fmt.Sprintf("  %-20s  %-8s  %-8s  %-10s  %s",
		"NAME", "DEFINED", "ACTIVE", "AUTOSTART", "BRIDGE"))
	defined, active, autostart, bridge := "no", "no", "no", "n/a"
	info, err := virshCmd("net-info", network).Output()
	if err == nil {
		defined = "yes"
		for _, line := range strings.Split(string(info), "\n") {
			switch {
			case strings.HasPrefix(line, "Active:"):
				active = strings.TrimSpace(strings.TrimPrefix(line, "Active:"))
			case strings.HasPrefix(line, "Autostart:"):
				autostart = strings.TrimSpace(strings.TrimPrefix(line, "Autostart:"))
			case strings.HasPrefix(line, "Bridge:"):
				bridge = strings.TrimSpace(strings.TrimPrefix(line, "Bridge:"))
			}
		}
	}
	out = append(out, fmt.Sprintf("  %-20s  %-8s  %-8s  %-10s  %s",
		network, defined, active, autostart, bridge))
	out = append(out, "")
	return out
}

func renderDashVMs(stack, network string, vmSSH map[string]bool) []string {
	var out []string
	out = append(out, dashSection("VMS"))
	out = append(out, fmt.Sprintf("  %-22s  %-10s  %-18s  %-15s  %-8s  %s",
		"NAME", "STATE", "MAC", "IP", "SSH", "READINESS"))
	domainsOut, err := virshCmd("list", "--all", "--name").Output()
	if err != nil {
		out = append(out, "  (virsh unavailable)")
		out = append(out, "")
		return out
	}
	var domains []string
	for _, d := range strings.Split(string(domainsOut), "\n") {
		d = strings.TrimSpace(d)
		if strings.HasPrefix(d, stack+"-") {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		out = append(out, "  (no domains yet)")
	} else {
		key := filepath.Join("keys", stack, "id_ed25519")
		for _, dom := range domains {
			state := DomainState(dom)
			mac := DomainMAC(dom)
			ip := ""
			if mac != "" {
				ip = DHCPLeaseIP(network, mac)
			}
			sshSt := "pending"
			if vmSSH[dom] {
				sshSt = "ready"
			} else if ip != "" {
				if sshReachable(key, ip) {
					vmSSH[dom] = true
					sshSt = "ready"
				}
			}
			readiness := "creating"
			switch {
			case state == "running" && sshSt == "ready":
				readiness = "ready"
			case state == "running" && ip != "":
				readiness = "waiting-ssh"
			case state == "running":
				readiness = "booting"
			}
			macStr := mac
			if macStr == "" {
				macStr = "n/a"
			}
			ipStr := ip
			if ipStr == "" {
				ipStr = "pending"
			}
			out = append(out, fmt.Sprintf("  %-22s  %-10s  %-18s  %-15s  %-8s  %s",
				dom, state, macStr, ipStr, sshSt, readiness))
		}
	}
	out = append(out, "")
	return out
}

func renderDashArtifacts(stack string) []string {
	var out []string
	out = append(out, dashSection("ARTIFACTS"))
	out = append(out, fmt.Sprintf("  %-36s  %-8s  %s", "FILE", "EXISTS", "SIZE"))
	found := false
	isos, _ := filepath.Glob(stack + "-*-seed.iso")
	for _, iso := range isos {
		found = true
		out = append(out, fmt.Sprintf("  %-36s  %-8s  %s", iso, "yes", fileSize(iso)))
	}
	logFiles, _ := filepath.Glob("logs/*.log")
	for _, lf := range logFiles {
		found = true
		out = append(out, fmt.Sprintf("  %-36s  %-8s  %s", lf, "yes", fileSize(lf)))
	}
	if !found {
		out = append(out, "  (no artifacts yet)")
	}
	out = append(out, "")
	return out
}

func renderDashEvents(stack string) []string {
	eventsFile := filepath.Join("logs", stack+"-events.log")
	var out []string
	out = append(out, dashSection(fmt.Sprintf("EVENTS (last %d)", dashMaxEvents)))
	f, err := os.Open(eventsFile)
	if err != nil {
		out = append(out, "  (no events yet)")
		out = append(out, "")
		return out
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		out = append(out, "  (no events yet)")
	} else {
		start := len(lines) - dashMaxEvents
		if start < 0 {
			start = 0
		}
		for _, l := range lines[start:] {
			out = append(out, "  "+l)
		}
	}
	out = append(out, "")
	return out
}

// sshReachable returns true if SSH can connect with BatchMode=yes.
func sshReachable(key, ip string) bool {
	cmd := exec.Command("ssh",
		"-n",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=1",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-i", key,
		sshUser+"@"+ip,
		"true",
	)
	return cmd.Run() == nil
}

// keyFingerprint returns the SHA256 fingerprint of an SSH key file.
func keyFingerprint(path string) string {
	out, err := exec.Command("ssh-keygen", "-l", "-f", path).Output()
	if err != nil {
		return "n/a"
	}
	fields := strings.Fields(string(out))
	if len(fields) >= 2 {
		return fields[1]
	}
	return "n/a"
}

// fileSize returns a human-readable size string for path.
func fileSize(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return "?"
	}
	const kb = 1024
	const mb = 1024 * kb
	sz := fi.Size()
	switch {
	case sz >= mb:
		return fmt.Sprintf("%.1fM", float64(sz)/mb)
	case sz >= kb:
		return fmt.Sprintf("%.1fK", float64(sz)/kb)
	default:
		return fmt.Sprintf("%dB", sz)
	}
}

func hideCursor() { fmt.Print("\033[?25l") }
func showCursor() { fmt.Print("\033[?25h") }
