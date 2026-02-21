package lab

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const maxWaitSeconds = 180

// LaunchTmux waits for all SSH VMs defined in layout.yaml to become reachable,
// then opens a tmux session with the configured pane layout.
func LaunchTmux(stack, network string) error {
	layoutFile := fmt.Sprintf("stacks/%s/layout.yaml", stack)
	if _, err := os.Stat(layoutFile); err != nil {
		return fmt.Errorf("no layout.yaml found at %s", layoutFile)
	}

	l, err := LoadLayout(layoutFile)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("keys/%s/id_ed25519", stack)
	session := fmt.Sprintf("red-team-%s", stack)
	sshVMs := l.SSHVMs()

	vmMAC := make(map[string]string)
	vmIP := make(map[string]string)
	vmSSHReady := make(map[string]bool)

	// Reserve N+1 lines for the readiness table (section header + column header + one per VM).
	fmt.Println(dashSectionHeader("Waiting for VMs")[0])
	fmt.Printf(dc(dDim+dBold, "  %-22s  %-12s  %-15s  %s\n"), "VM", "STATE", "IP", "SSH")
	for range sshVMs {
		fmt.Println()
	}

	if err := waitForVMsReady(stack, network, key, sshVMs, vmMAC, vmIP, vmSSHReady); err != nil {
		return err
	}

	fmt.Println()
	Ok("All VMs ready — launching tmux session")
	time.Sleep(500 * time.Millisecond)

	return launchTmuxSession(session, stack, key, l, vmIP)
}

func waitForVMsReady(stack, network, key string, sshVMs []string,
	vmMAC, vmIP map[string]string, vmSSHReady map[string]bool,
) error {
	elapsed := 0
	// redrawLines is the number of VM rows to overwrite on each refresh tick.
	redrawLines := len(sshVMs)

	for {
		allMACs := true
		for _, v := range sshVMs {
			if vmMAC[v] == "" {
				mac := DomainMAC(fmt.Sprintf("%s-%s", stack, v))
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

		for _, v := range sshVMs {
			if vmMAC[v] != "" {
				vmIP[v] = DHCPLeaseIP(network, vmMAC[v])
			}
			if vmIP[v] != "" && !vmSSHReady[v] {
				if sshReachable(key, vmIP[v]) {
					vmSSHReady[v] = true
				}
			}
		}

		// Move cursor up to overwrite the VM rows.
		fmt.Printf("\033[%dF", redrawLines)
		for _, v := range sshVMs {
			name := stack + "-" + v
			state := DomainState(name)
			ipStr := vmIP[v]
			if ipStr == "" {
				ipStr = dc(dDim, "pending")
			}
			fmt.Printf("  %-22s  %-12s  %-15s  %s\033[K\n",
				dc(dWhite, name),
				stateBadge(state),
				ipStr,
				sshBadge(vmSSHReady[v]),
			)
		}

		allReady := true
		for _, v := range sshVMs {
			if !vmSSHReady[v] {
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

func launchTmuxSession(session, stack, key string, l *Layout, vmIP map[string]string) error {
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()

	if err := exec.Command("tmux", "new-session", "-d", "-s", session).Run(); err != nil {
		return fmt.Errorf("tmux new-session: %w", err)
	}

	for i := 1; i < len(l.Panes); i++ {
		if err := exec.Command("tmux", "split-window", "-t", session).Run(); err != nil {
			return fmt.Errorf("tmux split-window: %w", err)
		}
	}

	for i, pane := range l.Panes {
		if err := exec.Command("tmux", "select-pane", "-t", fmt.Sprintf("%d", i)).Run(); err != nil {
			return fmt.Errorf("tmux select-pane %d: %w", i, err)
		}
		var cmd string
		if pane.Type == "ssh" {
			ip := vmIP[pane.VM]
			cmd = fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no %s@%s", key, sshUser, ip)
		} else {
			cmd = ExpandCommand(pane.Command, stack)
		}
		if err := exec.Command("tmux", "send-keys", "-t", session, cmd, "C-m").Run(); err != nil {
			return fmt.Errorf("tmux send-keys pane %d: %w", i, err)
		}
	}

	if err := exec.Command("tmux", "select-layout", l.Layout).Run(); err != nil {
		return fmt.Errorf("tmux select-layout: %w", err)
	}

	attach := exec.Command("tmux", "attach-session", "-t", session)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	return attach.Run()
}
