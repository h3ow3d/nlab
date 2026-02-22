// nlab – Red-Team Lab Framework CLI
//
// Command tree:
//
//	nlab version                     – print the nlab version
//	nlab doctor                      – check host prerequisites
//	nlab validate [<stack>|-f <file>] – validate a v1alpha1 stack manifest
//	nlab image download              – download the Ubuntu 22.04 base cloud image
//	nlab key generate <stack>        – generate a per-stack ed25519 SSH key pair
//	nlab network create <stack>      – define and start the libvirt network
//	nlab network destroy <stack>     – stop and undefine the libvirt network
//	nlab vm create <stack> <role>    – provision a single VM
//	nlab vm destroy <stack> <role>   – destroy a single VM
//	nlab session <stack>             – wait for SSH readiness then open tmux
//	nlab dashboard <stack>           – show the live creation dashboard
//	nlab up <stack>                  – full stack bring-up (key+net+vms+session)
//	nlab down <stack>                – full stack tear-down
//	nlab list                        – list all libvirt domains
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"

	lab "github.com/h3ow3d/nlab/internal"
	"github.com/h3ow3d/nlab/internal/manifest"
)

// Version is the nlab release string. Override at build time with:
//
//	go build -ldflags "-X main.Version=v1.2.3" ./cmd/nlab
var Version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "nlab",
		Short: "Red-Team Lab Framework",
		Long: `nlab – spin up isolated KVM/libvirt lab stacks for offensive-security practice.

Each stack provisions attacker and target VMs on a private NAT network,
then opens a tmux session so you can start working immediately.

Quick start:
  nlab image download   # one-time: fetch Ubuntu 22.04 base image
  nlab up basic         # bring up the basic stack
  nlab down basic       # tear it all down`,
	}

	root.AddCommand(
		versionCmd(),
		doctorCmd(),
		validateCmd(),
		imageCmd(),
		keyCmd(),
		networkCmd(),
		vmCmd(),
		sessionCmd(),
		dashboardCmd(),
		upCmd(),
		downCmd(),
		listCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── version ───────────────────────────────────────────────────────────────────

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the nlab version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("nlab", Version)
		},
	}
}

// ── doctor ────────────────────────────────────────────────────────────────────

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Check host prerequisites for nlab",
		SilenceUsage: true,
		Long: `Verifies that all required tools and system features are present:

  • virsh / libvirt connectivity (local qemu:///system)
  • qemu/kvm availability (/dev/kvm)
  • tmux
  • tcpdump
  • write access to XDG config / data / state directories

Exits with a non-zero status if any critical prerequisite is missing.`,
		Example: "  nlab doctor",
		RunE: func(_ *cobra.Command, _ []string) error {
			dirs := lab.DefaultXDGDirs()
			results := lab.RunDoctorChecks(dirs)

			allOK := true
			for _, r := range results {
				if r.OK {
					lab.Ok(r.Name + ": " + r.Message)
				} else {
					allOK = false
					lab.Error(r.Name + ": " + r.Message)
					if r.HowToFix != "" {
						fmt.Fprintf(os.Stderr, "     Fix: %s\n", r.HowToFix)
					}
				}
			}

			if !allOK {
				return fmt.Errorf("one or more prerequisites are missing; see above for details")
			}
			return nil
		},
	}
}

// ── validate ─────────────────────────────────────────────────────────────────────────

func validateCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:          "validate [<stack> | -f <file>]",
		Short:        "Validate a v1alpha1 stack manifest",
		SilenceUsage: true,
		Long: `Parses and validates a stack manifest against the nlab.io/v1alpha1 schema.

Supply either a stack name or an explicit file path:

  nlab validate basic              reads stacks/basic/stack.yaml
  nlab validate -f /my/stack.yaml  reads the given file directly

Checks performed:
  • apiVersion and kind are present and correct
  • metadata.name is set
  • spec.networks and spec.vms are non-empty
  • Each network and VM has a non-empty xml field
  • All xml fields are well-formed XML`,
		Example: "  nlab validate basic\n  nlab validate -f stacks/basic/stack.yaml",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := file
			if path == "" {
				if len(args) == 0 {
					return fmt.Errorf("provide a stack name (e.g. nlab validate basic) or use -f <file>")
				}
				path = fmt.Sprintf("stacks/%s/stack.yaml", args[0])
			}
			if _, err := manifest.Load(path); err != nil {
				return err
			}
			fmt.Printf("manifest %q is valid\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to the stack manifest YAML file (overrides stack name)")
	return cmd
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

Replaces: ./images/download_base.sh`,
		Example: "  nlab image download",
		RunE: func(_ *cobra.Command, _ []string) error {
			return lab.Download()
		},
	})
	return cmd
}

// ── key ───────────────────────────────────────────────────────────────────────

func keyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage per-stack SSH key pairs",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "generate <stack>",
		Short: "Generate an ed25519 SSH key pair for a stack",
		Long: `Generates keys/<stack>/id_ed25519 and keys/<stack>/id_ed25519.pub if they
do not already exist.

Replaces: ./scripts/generate-key.sh <stack>`,
		Example: "  nlab key generate basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return lab.EnsureKey(args[0])
		},
	})
	return cmd
}

// ── network ───────────────────────────────────────────────────────────────────

func networkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage libvirt networks",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create <stack>",
		Short: "Define and start the libvirt network for a stack",
		Long: `Reads stacks/<stack>/network.xml and stacks/<stack>/stack.yaml, then
defines, starts, and sets the network to autostart in libvirt.

Replaces: ./scripts/create-network.sh stacks/<stack>/network.xml <network> <stack>`,
		Example: "  nlab network create basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			stackName := args[0]
			cfg, err := lab.LoadStack(stackName)
			if err != nil {
				return err
			}
			return lab.CreateNetwork(fmt.Sprintf("stacks/%s/network.xml", stackName), cfg.Network)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "destroy <stack>",
		Short: "Stop and undefine the libvirt network for a stack",
		Long: `Reads stacks/<stack>/stack.yaml for the network name, then stops and
undefines the libvirt network.

Replaces: ./scripts/destroy-network.sh <network>`,
		Example: "  nlab network destroy basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := lab.LoadStack(args[0])
			if err != nil {
				return err
			}
			return lab.DestroyNetwork(cfg.Network)
		},
	})

	return cmd
}

// ── vm ────────────────────────────────────────────────────────────────────────

func vmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage individual virtual machines",
	}

	var memory int
	var vcpus int

	createCmd := &cobra.Command{
		Use:   "create <stack> <role>",
		Short: "Provision a single VM within a stack",
		Long: `Creates a VM named <stack>-<role> using virt-install and cloud-init.
VM specs (memory, vcpus) are read from stacks/<stack>/stack.yaml unless
overridden with --memory / --vcpus flags.

Replaces: ./scripts/create-vm.sh <stack> <role> <memory-mb> <vcpus> <network>`,
		Example: `  nlab vm create basic attacker
  nlab vm create basic attacker --memory 8192 --vcpus 4`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			stackName, role := args[0], args[1]
			cfg, err := lab.LoadStack(stackName)
			if err != nil {
				return err
			}
			mem, cpus := memory, vcpus
			if mem == 0 || cpus == 0 {
				for _, v := range cfg.VMs {
					if v.Name == role {
						if mem == 0 {
							mem = v.Memory
						}
						if cpus == 0 {
							cpus = v.VCPUs
						}
						break
					}
				}
			}
			if mem == 0 {
				return fmt.Errorf("no memory spec found for role %q in stack.yaml; use --memory", role)
			}
			if cpus == 0 {
				return fmt.Errorf("no vcpus spec found for role %q in stack.yaml; use --vcpus", role)
			}
			return lab.CreateVM(lab.VMConfig{
				Stack:   stackName,
				Role:    role,
				Memory:  mem,
				VCPUs:   cpus,
				Network: cfg.Network,
			})
		},
	}
	createCmd.Flags().IntVar(&memory, "memory", 0, "RAM in MiB (overrides stack.yaml)")
	createCmd.Flags().IntVar(&vcpus, "vcpus", 0, "vCPU count (overrides stack.yaml)")
	cmd.AddCommand(createCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "destroy <stack> <role>",
		Short: "Destroy a single VM and remove its storage",
		Long: `Stops and undefines the VM named <stack>-<role>, removing all storage.

Replaces: ./scripts/destroy-vm.sh <stack> <role>`,
		Example: "  nlab vm destroy basic attacker",
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return lab.DestroyVM(args[0], args[1])
		},
	})

	return cmd
}

// ── session ───────────────────────────────────────────────────────────────────

func sessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session <stack>",
		Short: "Wait for SSH readiness then open a tmux session",
		Long: `Polls each SSH VM defined in stacks/<stack>/layout.yaml until it is
reachable, then opens a tmux session with the configured pane layout.

Replaces: ./scripts/launch-tmux.sh <stack> <network>`,
		Example: "  nlab session basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := lab.LoadStack(args[0])
			if err != nil {
				return err
			}
			return lab.LaunchTmux(args[0], cfg.Network)
		},
	}
}

// ── dashboard ─────────────────────────────────────────────────────────────────

func dashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard <stack>",
		Short: "Show the live creation dashboard for a stack",
		Long: `Renders a continuously-refreshed in-place dashboard showing keys,
network, VM states, artifacts, and recent event log entries.
Press Ctrl-C to exit.

Replaces: ./scripts/create-dashboard.sh <stack> <network>`,
		Example: "  nlab dashboard basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := lab.LoadStack(args[0])
			if err != nil {
				return err
			}
			done := make(chan struct{})
			lab.RunDashboard(args[0], cfg.Network, done)
			return nil
		},
	}
}

// ── up ────────────────────────────────────────────────────────────────────────

func upCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up <stack>",
		Short: "Stand up a complete lab stack",
		Long: `Brings up the named stack end-to-end:
  1. nlab key generate  <stack>
  2. nlab network create <stack>
  3. nlab vm create <stack> <role>  (all VMs in parallel)
  4. nlab dashboard <stack>          (live progress display)
  5. nlab session <stack>            (tmux when all VMs are SSH-ready)

Stack configuration is read from stacks/<stack>/stack.yaml.

Replaces: make <stack>`,
		Example: "  nlab up basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUp(args[0])
		},
	}
}

func runUp(stackName string) error {
	cfg, err := lab.LoadStack(stackName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll("logs", 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	if err := lab.EnsureKey(stackName); err != nil {
		return err
	}

	if err := lab.CreateNetwork(fmt.Sprintf("stacks/%s/network.xml", stackName), cfg.Network); err != nil {
		return err
	}

	// Start dashboard in background. Use a WaitGroup so we can be sure it has
	// fully exited (and released the terminal) before LaunchTmux draws anything.
	done := make(chan struct{})
	var dashWg sync.WaitGroup
	dashWg.Add(1)
	go func() {
		defer dashWg.Done()
		lab.RunDashboard(stackName, cfg.Network, done)
	}()

	var vmWg sync.WaitGroup
	errs := make(chan error, len(cfg.VMs))

	for _, v := range cfg.VMs {
		v := v
		vmWg.Add(1)
		go func() {
			defer vmWg.Done()
			logPath := fmt.Sprintf("logs/%s.log", v.Name)
			logFile, err := os.Create(logPath)
			if err != nil {
				errs <- fmt.Errorf("open log %s: %w", logPath, err)
				return
			}
			defer logFile.Close()

			if err := lab.CreateVM(lab.VMConfig{
				Stack:   stackName,
				Role:    v.Name,
				Memory:  v.Memory,
				VCPUs:   v.VCPUs,
				Network: cfg.Network,
				Out:     logFile, // redirect virt-install / cloud-localds away from stdout
			}); err != nil {
				errs <- fmt.Errorf("create VM %s: %w", v.Name, err)
			}
		}()
	}

	vmWg.Wait()
	close(done)   // signal dashboard to stop
	dashWg.Wait() // wait until dashboard goroutine has fully exited

	close(errs)
	for e := range errs {
		if e != nil {
			return e
		}
	}

	return lab.LaunchTmux(stackName, cfg.Network)
}

// ── down ──────────────────────────────────────────────────────────────────────

func downCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <stack>",
		Short: "Tear down a complete lab stack",
		Long: `Destroys every VM in the stack (and their storage), then removes the
libvirt network.

Equivalent to running:
  nlab vm destroy <stack> <role>  (for each VM)
  nlab network destroy <stack>

Stack configuration is read from stacks/<stack>/stack.yaml.

Replaces: make <stack>-destroy`,
		Example: "  nlab down basic",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDown(args[0])
		},
	}
}

func runDown(stackName string) error {
	cfg, err := lab.LoadStack(stackName)
	if err != nil {
		return err
	}

	for _, v := range cfg.VMs {
		if err := lab.DestroyVM(stackName, v.Name); err != nil {
			lab.Error(err.Error())
		}
	}

	return lab.DestroyNetwork(cfg.Network)
}

// ── list ──────────────────────────────────────────────────────────────────────

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all libvirt domains",
		Long: `Runs 'virsh list --all' to show every defined domain regardless of state.

Replaces: make list`,
		Example: "  nlab list",
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("virsh", "--connect", "qemu:///system", "list", "--all")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}
