// nlab – Red-Team Lab Framework CLI
//
// Usage:
//
//	nlab image download          – download the Ubuntu 22.04 base cloud image
//	nlab up   <stack>            – stand up a lab stack
//	nlab down <stack>            – tear down a lab stack
//	nlab list                    – list all libvirt domains
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"

	"github.com/h3ow3d/nlab/internal/dashboard"
	"github.com/h3ow3d/nlab/internal/image"
	"github.com/h3ow3d/nlab/internal/keys"
	"github.com/h3ow3d/nlab/internal/log"
	"github.com/h3ow3d/nlab/internal/network"
	"github.com/h3ow3d/nlab/internal/stack"
	"github.com/h3ow3d/nlab/internal/tmux"
	"github.com/h3ow3d/nlab/internal/vm"
)

func main() {
	root := &cobra.Command{
		Use:   "nlab",
		Short: "Red-Team Lab Framework",
		Long: `nlab – spin up isolated KVM/libvirt lab stacks for offensive-security practice.

Each stack provisions attacker and target VMs on a private NAT network,
then opens a tmux session so you can start working immediately.`,
	}

	root.AddCommand(imageCmd(), upCmd(), downCmd(), listCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── image ─────────────────────────────────────────────────────────────────────

func imageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage base images",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "download",
		Short: "Download the Ubuntu 22.04 base cloud image",
		Long: `Downloads jammy-server-cloudimg-amd64.img, verifies its SHA-256 checksum,
and installs it to /var/lib/libvirt/images/ubuntu-base.qcow2.

Equivalent to: ./images/download_base.sh`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return image.Download()
		},
	})
	return cmd
}

// ── up ────────────────────────────────────────────────────────────────────────

func upCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up <stack>",
		Short: "Stand up a lab stack",
		Long: `Brings up the named stack:
  1. generates a per-stack ed25519 SSH key
  2. defines and starts the libvirt network
  3. provisions all VMs in parallel (cloud-init / virt-install)
  4. shows a live dashboard while VMs boot
  5. opens a tmux session when every VM is SSH-reachable

Stack configuration is read from stacks/<stack>/stack.yaml.

Equivalent to: make <stack>`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUp(args[0])
		},
	}
}

func runUp(stackName string) error {
	cfg, err := stack.Load(stackName)
	if err != nil {
		return err
	}

	networkXML := fmt.Sprintf("stacks/%s/network.xml", stackName)

	if err := os.MkdirAll("logs", 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	// 1. Generate SSH key.
	if err := keys.EnsureKey(stackName); err != nil {
		return err
	}

	// 2. Create network.
	if err := network.Create(networkXML, cfg.Network); err != nil {
		return err
	}

	// 3. Provision VMs in parallel with a live dashboard.
	done := make(chan struct{})
	go dashboard.Run(stackName, cfg.Network, done)

	var wg sync.WaitGroup
	errs := make(chan error, len(cfg.VMs))

	for _, v := range cfg.VMs {
		v := v // capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			logPath := fmt.Sprintf("logs/%s.log", v.Name)
			logFile, err := os.Create(logPath)
			if err != nil {
				errs <- fmt.Errorf("open log %s: %w", logPath, err)
				return
			}
			defer logFile.Close()

			vmCfg := vm.Config{
				Stack:   stackName,
				Role:    v.Name,
				Memory:  v.Memory,
				VCPUs:   v.VCPUs,
				Network: cfg.Network,
			}
			if err := vm.Create(vmCfg); err != nil {
				errs <- fmt.Errorf("create VM %s: %w", v.Name, err)
			}
		}()
	}

	wg.Wait()
	close(done)

	close(errs)
	for e := range errs {
		if e != nil {
			return e
		}
	}

	// 4. Launch tmux session.
	return tmux.Launch(stackName, cfg.Network)
}

// ── down ──────────────────────────────────────────────────────────────────────

func downCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <stack>",
		Short: "Tear down a lab stack",
		Long: `Destroys all VMs in the named stack (and their storage), then removes the
libvirt network.

Stack configuration is read from stacks/<stack>/stack.yaml.

Equivalent to: make <stack>-destroy`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDown(args[0])
		},
	}
}

func runDown(stackName string) error {
	cfg, err := stack.Load(stackName)
	if err != nil {
		return err
	}

	for _, v := range cfg.VMs {
		if err := vm.Destroy(stackName, v.Name); err != nil {
			log.Error(err.Error())
		}
	}

	return network.Destroy(cfg.Network)
}

// ── list ──────────────────────────────────────────────────────────────────────

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all libvirt domains",
		Long: `Runs 'virsh list --all' to show every defined domain regardless of state.

Equivalent to: make list`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("virsh", "--connect", "qemu:///system", "list", "--all")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}
