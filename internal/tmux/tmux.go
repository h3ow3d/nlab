// Package tmux launches a tmux session defined by a layout.yaml file.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/h3ow3d/nlab/internal/layout"
	"github.com/h3ow3d/nlab/internal/log"
	"github.com/h3ow3d/nlab/internal/vm"
)

const (
	maxWaitSeconds = 180
	sshUser        = "ubuntu"
)

// Launch waits for all SSH VMs defined in layout.yaml to become reachable, then
// opens a tmux session with the configured pane layout.  It mirrors launch-tmux.sh.
func Launch(stack, network string) error {
	layoutFile := fmt.Sprintf("stacks/%s/layout.yaml", stack)
	if _, err := os.Stat(layoutFile); err != nil {
		return fmt.Errorf("no layout.yaml found at %s", layoutFile)
	}

	l, err := layout.Load(layoutFile)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("keys/%s/id_ed25519", stack)
	session := fmt.Sprintf("red-team-%s", stack)

	sshVMs := l.SSHVMs()

	// Maps for tracking per-VM state.
	vmMAC := make(map[string]string)
	vmIP := make(map[string]string)
	vmSSH := make(map[string]bool)

	log.Info("Waiting for VMs to become ready...")

	// Print initial blank status lines so cursor-up overwriting works.
	for range sshVMs {
		fmt.Println()
	}
	fmt.Println()

	if err := waitForReady(stack, network, sshVMs, vmMAC, vmIP, vmSSH); err != nil {
		return err
	}

	fmt.Println()
	log.Ok("All VMs ready")
	time.Sleep(time.Second)

	return launchTmux(session, stack, key, l, vmIP)
}

// waitForReady blocks until all SSH VMs have a MAC, an IP, and accept SSH.
func waitForReady(stack, network string, sshVMs []string,
	vmMAC, vmIP map[string]string, vmSSH map[string]bool,
) error {
	elapsed := 0
	vmCount := len(sshVMs)

	for {
		// Gather MACs.
		allMACs := true
		for _, v := range sshVMs {
			if vmMAC[v] == "" {
				mac := vm.DomainMAC(fmt.Sprintf("%s-%s", stack, v))
				if mac != "" {
					vmMAC[v] = mac
				} else {
					allMACs = false
				}
			}
		}

		if !allMACs {
			if elapsed >= maxWaitSeconds {
				return fmt.Errorf("timeout waiting for VM network interfaces")
			}
			time.Sleep(time.Second)
			elapsed++
			continue
		}

		// Try SSH for each VM.
		for _, v := range sshVMs {
			if vmMAC[v] != "" {
				ip := vm.DHCPLeaseIP(network, vmMAC[v])
				vmIP[v] = ip
			}
			if vmIP[v] != "" && !vmSSH[v] {
				if sshReachable(fmt.Sprintf("keys/%s/id_ed25519", stack), vmIP[v]) {
					vmSSH[v] = true
				}
			}
		}

		// Overwrite previous status lines.
		fmt.Printf("\033[%dF", vmCount+1)
		for _, v := range sshVMs {
			name := stack + "-" + v
			state := vm.DomainState(name)
			sshStatus := "pending"
			if vmSSH[v] {
				sshStatus = "ready"
			}
			ipStr := vmIP[v]
			if ipStr == "" {
				ipStr = "pending"
			}
			fmt.Printf("%-18s | state: %-12s | ip: %-15s | ssh: %-8s\n",
				name, state, ipStr, sshStatus)
		}
		fmt.Println()

		allReady := true
		for _, v := range sshVMs {
			if !vmSSH[v] {
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}

		if elapsed >= maxWaitSeconds {
			return fmt.Errorf("timeout waiting for VM readiness")
		}

		time.Sleep(2 * time.Second)
		elapsed += 2
	}
}

// launchTmux kills any existing session with the same name, creates a new one,
// and sends commands to each pane according to the layout.
func launchTmux(session, stack, key string, l *layout.Layout, vmIP map[string]string) error {
	// Kill previous session if it exists.
	_ = exec.Command("tmux", "has-session", "-t", session).Run()
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()

	if err := exec.Command("tmux", "new-session", "-d", "-s", session).Run(); err != nil {
		return fmt.Errorf("tmux new-session: %w", err)
	}

	// Create additional panes (first is created by new-session).
	for i := 1; i < len(l.Panes); i++ {
		if err := exec.Command("tmux", "split-window", "-t", session).Run(); err != nil {
			return fmt.Errorf("tmux split-window: %w", err)
		}
	}

	// Send commands to each pane.
	for i, pane := range l.Panes {
		if err := exec.Command("tmux", "select-pane", "-t", fmt.Sprintf("%d", i)).Run(); err != nil {
			return fmt.Errorf("tmux select-pane %d: %w", i, err)
		}

		var cmd string
		if pane.Type == "ssh" {
			ip := vmIP[pane.VM]
			cmd = fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no %s@%s", key, sshUser, ip)
		} else {
			cmd = layout.ExpandCommand(pane.Command, stack)
		}

		if err := exec.Command("tmux", "send-keys", "-t", session, cmd, "C-m").Run(); err != nil {
			return fmt.Errorf("tmux send-keys pane %d: %w", i, err)
		}
	}

	if err := exec.Command("tmux", "select-layout", l.Layout).Run(); err != nil {
		return fmt.Errorf("tmux select-layout: %w", err)
	}

	// Attach – this replaces the current process.
	attach := exec.Command("tmux", "attach-session", "-t", session)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	return attach.Run()
}

// sshReachable returns true if ssh can connect with BatchMode=yes.
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
