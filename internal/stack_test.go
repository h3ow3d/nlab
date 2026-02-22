package lab_test

import (
	"os"
	"path/filepath"
	"testing"

	lab "github.com/h3ow3d/nlab/internal"
)

func setupStack(t *testing.T, name, content string) {
	t.Helper()
	dir := t.TempDir()
	stackDir := filepath.Join(dir, "stacks", name)
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stackDir, "stack.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestLoadStackBasic(t *testing.T) {
	setupStack(t, "basic", `
network: basic_net
vms:
  - name: attacker
    memory: 4096
    vcpus: 2
  - name: target
    memory: 2048
    vcpus: 2
`)
	cfg, err := lab.LoadStack("basic")
	if err != nil {
		t.Fatalf("LoadStack: %v", err)
	}
	if cfg.Network != "basic_net" {
		t.Errorf("Network = %q, want basic_net", cfg.Network)
	}
	if len(cfg.VMs) != 2 {
		t.Fatalf("len(VMs) = %d, want 2", len(cfg.VMs))
	}
	if cfg.VMs[0].Name != "attacker" || cfg.VMs[0].Memory != 4096 || cfg.VMs[0].VCPUs != 2 {
		t.Errorf("VMs[0] = %+v", cfg.VMs[0])
	}
	if cfg.VMs[1].Name != "target" || cfg.VMs[1].Memory != 2048 {
		t.Errorf("VMs[1] = %+v", cfg.VMs[1])
	}
}

func TestLoadStackMissingNetwork(t *testing.T) {
	setupStack(t, "bad", `
vms:
  - name: x
    memory: 1024
    vcpus: 1
`)
	_, err := lab.LoadStack("bad")
	if err == nil {
		t.Error("expected error for missing network field, got nil")
	}
}

func TestLoadStackMissingVMs(t *testing.T) {
	setupStack(t, "empty", `network: foo_net`)
	_, err := lab.LoadStack("empty")
	if err == nil {
		t.Error("expected error for empty vms list, got nil")
	}
}

func TestLoadStackNotFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	_ = os.Chdir(dir)

	_, err := lab.LoadStack("nosuchstack")
	if err == nil {
		t.Error("expected error for non-existent stack.yaml, got nil")
	}
}

func TestLoadStackV1alpha1(t *testing.T) {
	setupStack(t, "v1", `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: v1
spec:
  networks:
    mynet:
      xml: |
        <network><name>mynet</name></network>
  vms:
    attacker:
      xml: |
        <domain type="kvm">
          <memory unit="MiB">4096</memory>
          <vcpu>2</vcpu>
        </domain>
    target:
      xml: |
        <domain type="kvm">
          <memory unit="MiB">2048</memory>
          <vcpu>1</vcpu>
        </domain>
`)
	cfg, err := lab.LoadStack("v1")
	if err != nil {
		t.Fatalf("LoadStack v1alpha1: %v", err)
	}
	if cfg.Network != "mynet" {
		t.Errorf("Network = %q, want mynet", cfg.Network)
	}
	if len(cfg.VMs) != 2 {
		t.Fatalf("len(VMs) = %d, want 2", len(cfg.VMs))
	}
	// Find attacker VM (map iteration order is non-deterministic)
	var attacker *lab.VMSpec
	for i := range cfg.VMs {
		if cfg.VMs[i].Name == "attacker" {
			attacker = &cfg.VMs[i]
			break
		}
	}
	if attacker == nil {
		t.Fatal("VM 'attacker' not found")
	}
	if attacker.Memory != 4096 {
		t.Errorf("attacker.Memory = %d, want 4096", attacker.Memory)
	}
	if attacker.VCPUs != 2 {
		t.Errorf("attacker.VCPUs = %d, want 2", attacker.VCPUs)
	}
}
