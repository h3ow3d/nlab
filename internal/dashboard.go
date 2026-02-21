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

// ── Dashboard ANSI palette ────────────────────────────────────────────────────

const (
	// Colors used only inside the dashboard (not exported).
	dReset   = "\033[0m"
	dBold    = "\033[1m"
	dDim     = "\033[2m"
	dCyan    = "\033[36m"
	dGreen   = "\033[32m"
	dYellow  = "\033[33m"
	dRed     = "\033[31m"
	dMagenta = "\033[35m"
	dBlue    = "\033[34m"
	dWhite   = "\033[97m"
)

const dashMaxEvents = 8

// dashWidth is the total inner width (excluding the │ borders).
const dashWidth = 78

func dc(color, s string) string { return color + s + dReset }

// ── Public entry point ────────────────────────────────────────────────────────

// RunDashboard renders a refreshing dashboard for the given stack and network
// until the done channel is closed.
func RunDashboard(stack, network string, done <-chan struct{}) {
	vmSSH := make(map[string]bool)
	startTime := time.Now()
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

		lines := renderDashboard(stack, network, vmSSH, startTime)
		for _, l := range lines {
			fmt.Printf("%s\033[K\n", l)
		}
		prevLines = len(lines)
		time.Sleep(time.Second)
	}
}

// ── Top-level renderer ────────────────────────────────────────────────────────

func renderDashboard(stack, network string, vmSSH map[string]bool, start time.Time) []string {
	var out []string
	out = append(out, dashHeader(stack, start))
	out = append(out, "")
	out = append(out, renderDashKeys(stack)...)
	out = append(out, renderDashNetworks(network)...)
	out = append(out, renderDashVMs(stack, network, vmSSH)...)
	out = append(out, renderDashArtifacts(stack)...)
	out = append(out, renderDashEvents(stack)...)
	return out
}

// ── Header ────────────────────────────────────────────────────────────────────

func dashHeader(stack string, start time.Time) string {
	elapsed := time.Since(start).Truncate(time.Second)
	h := int(elapsed.Hours())
	m := int(elapsed.Minutes()) % 60
	s := int(elapsed.Seconds()) % 60
	uptime := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	clock := time.Now().Format("15:04:05")

	left := dc(dCyan+dBold, " ◆ nlab") + dc(dDim, " │") + " stack=" + dc(dWhite+dBold, stack)
	right := dc(dDim, "up "+uptime) + "  " + dc(dDim, clock) + " "

	// Build a fixed-width header line.
	// Strip ANSI for length calculation.
	visLeft := " ◆ nlab │ stack=" + stack
	visRight := "up " + uptime + "  " + clock + " "
	pad := dashWidth - len(visLeft) - len(visRight)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

// ── Section chrome ────────────────────────────────────────────────────────────

func dashSectionHeader(title string) []string {
	label := " " + dc(dCyan+dBold, title) + " "
	// visible length of label
	visLabel := " " + title + " "
	leftLine := dc(dDim, strings.Repeat("─", 2))
	rightLen := dashWidth - 2 - len(visLabel)
	if rightLen < 0 {
		rightLen = 0
	}
	rightLine := dc(dDim, strings.Repeat("─", rightLen))
	return []string{leftLine + label + rightLine}
}

func dashColHeader(cols string) string {
	return dc(dDim+dBold, cols)
}

// ── Keys section ─────────────────────────────────────────────────────────────

func renderDashKeys(stack string) []string {
	var out []string
	out = append(out, dashSectionHeader("KEYS")...)
	out = append(out, dashColHeader(fmt.Sprintf("  %-38s  %-8s  %s", "FILE", "STATUS", "FINGERPRINT")))

	priv := filepath.Join("keys", stack, "id_ed25519")
	pub := priv + ".pub"

	if _, err := os.Stat(priv); err == nil {
		fp := keyFingerprint(priv)
		out = append(out, fmt.Sprintf("  %-38s  %s  %s",
			priv, dc(dGreen, "✓ ok    "), dc(dDim, fp)))
	} else {
		out = append(out, fmt.Sprintf("  %-38s  %s", priv, dc(dYellow, "⚠ missing")))
	}
	if _, err := os.Stat(pub); err == nil {
		out = append(out, fmt.Sprintf("  %-38s  %s", pub, dc(dGreen, "✓ ok")))
	} else {
		out = append(out, fmt.Sprintf("  %-38s  %s", pub, dc(dYellow, "⚠ missing")))
	}
	out = append(out, "")
	return out
}

// ── Networks section ──────────────────────────────────────────────────────────

func renderDashNetworks(network string) []string {
	var out []string
	out = append(out, dashSectionHeader("NETWORK")...)
	out = append(out, dashColHeader(fmt.Sprintf("  %-22s  %-8s  %-8s  %-10s  %s",
		"NAME", "DEFINED", "ACTIVE", "AUTOSTART", "BRIDGE")))

	defined, active, autostart, bridge := false, false, false, "n/a"
	info, err := virshCmd("net-info", network).Output()
	if err == nil {
		defined = true
		for _, line := range strings.Split(string(info), "\n") {
			switch {
			case strings.HasPrefix(line, "Active:"):
				active = strings.TrimSpace(strings.TrimPrefix(line, "Active:")) == "yes"
			case strings.HasPrefix(line, "Autostart:"):
				autostart = strings.TrimSpace(strings.TrimPrefix(line, "Autostart:")) == "yes"
			case strings.HasPrefix(line, "Bridge:"):
				bridge = strings.TrimSpace(strings.TrimPrefix(line, "Bridge:"))
			}
		}
	}

	out = append(out, fmt.Sprintf("  %-22s  %-8s  %-8s  %-10s  %s",
		dc(dWhite, network),
		boolBadge(defined),
		boolBadge(active),
		boolBadge(autostart),
		dc(dDim, bridge),
	))
	out = append(out, "")
	return out
}

// ── VMs section ───────────────────────────────────────────────────────────────

func renderDashVMs(stack, network string, vmSSH map[string]bool) []string {
	var out []string
	out = append(out, dashSectionHeader("VMS")...)
	out = append(out, dashColHeader(fmt.Sprintf("  %-24s  %-10s  %-17s  %-15s  %s",
		"NAME", "STATE", "IP", "SSH", "READINESS")))

	domainsOut, err := virshCmd("list", "--all", "--name").Output()
	if err != nil {
		out = append(out, dc(dRed, "  (virsh unavailable)"))
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
		out = append(out, dc(dDim, "  (no VMs yet — provisioning…)"))
	} else {
		key := filepath.Join("keys", stack, "id_ed25519")
		for _, dom := range domains {
			state := DomainState(dom)
			mac := DomainMAC(dom)
			ip := ""
			if mac != "" {
				ip = DHCPLeaseIP(network, mac)
			}

			sshReady := vmSSH[dom]
			if !sshReady && ip != "" {
				if sshReachable(key, ip) {
					vmSSH[dom] = true
					sshReady = true
				}
			}

			ipStr := ip
			if ipStr == "" {
				ipStr = dc(dDim, "pending")
			}

			out = append(out, fmt.Sprintf("  %-24s  %-10s  %-17s  %-15s  %s",
				dc(dWhite, dom),
				stateBadge(state),
				ipStr,
				sshBadge(sshReady),
				readinessBadge(state, sshReady, ip),
			))
		}
	}
	out = append(out, "")
	return out
}

// ── Artifacts section ─────────────────────────────────────────────────────────

func renderDashArtifacts(stack string) []string {
	var out []string
	out = append(out, dashSectionHeader("ARTIFACTS")...)
	out = append(out, dashColHeader(fmt.Sprintf("  %-38s  %s", "FILE", "SIZE")))

	found := false
	isos, _ := filepath.Glob(stack + "-*-seed.iso")
	for _, iso := range isos {
		found = true
		out = append(out, fmt.Sprintf("  %s  %s",
			dc(dDim, fmt.Sprintf("%-38s", iso)),
			dc(dDim, fileSize(iso))))
	}
	logFiles, _ := filepath.Glob("logs/*.log")
	for _, lf := range logFiles {
		found = true
		out = append(out, fmt.Sprintf("  %s  %s",
			dc(dDim, fmt.Sprintf("%-38s", lf)),
			dc(dDim, fileSize(lf))))
	}
	if !found {
		out = append(out, dc(dDim, "  (none yet)"))
	}
	out = append(out, "")
	return out
}

// ── Events section ────────────────────────────────────────────────────────────

func renderDashEvents(stack string) []string {
	eventsFile := filepath.Join("logs", stack+"-events.log")
	var out []string
	out = append(out, dashSectionHeader(fmt.Sprintf("EVENTS  (last %d)", dashMaxEvents))...)

	f, err := os.Open(eventsFile)
	if err != nil {
		out = append(out, dc(dDim, "  (no events yet)"))
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
		out = append(out, dc(dDim, "  (no events yet)"))
	} else {
		start := len(lines) - dashMaxEvents
		if start < 0 {
			start = 0
		}
		for _, l := range lines[start:] {
			out = append(out, "  "+colorizeEvent(l))
		}
	}
	out = append(out, "")
	return out
}

// ── Badge helpers ─────────────────────────────────────────────────────────────

// boolBadge returns a colored yes/no indicator.
func boolBadge(v bool) string {
	if v {
		return dc(dGreen, "yes")
	}
	return dc(dDim, "no")
}

// stateBadge colors the virsh domain state string.
func stateBadge(state string) string {
	switch state {
	case "running":
		return dc(dGreen, "● running ")
	case "shut off":
		return dc(dDim, "○ off     ")
	case "paused":
		return dc(dYellow, "⏸ paused  ")
	default:
		return dc(dDim, "? "+state)
	}
}

// sshBadge returns a colored SSH readiness indicator.
func sshBadge(ready bool) string {
	if ready {
		return dc(dGreen, "✓ ready  ")
	}
	return dc(dYellow, "⏳ pending")
}

// readinessBadge returns a summary badge for the VM's overall readiness.
func readinessBadge(state string, sshReady bool, ip string) string {
	switch {
	case state == "running" && sshReady:
		return dc(dGreen+dBold, "✓ ready")
	case state == "running" && ip != "":
		return dc(dYellow, "⏳ waiting for ssh")
	case state == "running":
		return dc(dYellow, "⏳ booting")
	default:
		return dc(dDim, "… creating")
	}
}

// colorizeEvent highlights timestamp and source tags inside an event line.
func colorizeEvent(line string) string {
	// Color the timestamp (first token that looks like HH:MM:SS).
	fields := strings.SplitN(line, " ", 4)
	if len(fields) >= 3 {
		// "EVENT 23:25:18 [create-vm] Installing VM basic-attacker"
		//   ^       ^        ^
		_ = fields[0] // "EVENT" label — dimmed
		ts := fields[1]
		src := fields[2]
		rest := ""
		if len(fields) == 4 {
			rest = fields[3]
		}
		return dc(dDim, fields[0]) + " " +
			dc(dCyan, ts) + " " +
			dc(dMagenta, src) + " " +
			rest
	}
	return dc(dDim, line)
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

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
