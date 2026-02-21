package manifest_test

import (
	"strings"
	"testing"

	"github.com/h3ow3d/nlab/internal/manifest"
)

const validManifest = `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: basic
spec:
  networks:
    basic_net:
      xml: |
        <network>
          <name>basic_net</name>
          <bridge name="virbr-basic" stp="on" delay="0"/>
          <forward mode="nat"/>
          <ip address="10.10.10.1" netmask="255.255.255.0">
            <dhcp>
              <range start="10.10.10.100" end="10.10.10.200"/>
            </dhcp>
          </ip>
        </network>
  vms:
    attacker:
      xml: |
        <domain type="kvm">
          <name>attacker</name>
          <memory unit="MiB">4096</memory>
          <vcpu>2</vcpu>
        </domain>
    target:
      xml: |
        <domain type="kvm">
          <name>target</name>
          <memory unit="MiB">2048</memory>
          <vcpu>2</vcpu>
        </domain>
`

func TestLoadBytesValid(t *testing.T) {
	m, err := manifest.LoadBytes([]byte(validManifest), "test")
	if err != nil {
		t.Fatalf("LoadBytes: %v", err)
	}
	if m.APIVersion != "nlab.io/v1alpha1" {
		t.Errorf("APIVersion = %q, want nlab.io/v1alpha1", m.APIVersion)
	}
	if m.Kind != "Stack" {
		t.Errorf("Kind = %q, want Stack", m.Kind)
	}
	if m.Metadata.Name != "basic" {
		t.Errorf("Metadata.Name = %q, want basic", m.Metadata.Name)
	}
	if len(m.Spec.Networks) != 1 {
		t.Errorf("len(Networks) = %d, want 1", len(m.Spec.Networks))
	}
	if _, ok := m.Spec.Networks["basic_net"]; !ok {
		t.Error("expected network basic_net")
	}
	if len(m.Spec.VMs) != 2 {
		t.Errorf("len(VMs) = %d, want 2", len(m.Spec.VMs))
	}
	if _, ok := m.Spec.VMs["attacker"]; !ok {
		t.Error("expected VM attacker")
	}
	if _, ok := m.Spec.VMs["target"]; !ok {
		t.Error("expected VM target")
	}
}

func TestLoadBytesFile(t *testing.T) {
	_, err := manifest.Load("/nonexistent/path/stack.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestValidateMissingAPIVersion(t *testing.T) {
	yaml := strings.ReplaceAll(validManifest, "apiVersion: nlab.io/v1alpha1\n", "")
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing apiVersion, got nil")
	}
	if !strings.Contains(err.Error(), "apiVersion") {
		t.Errorf("error should mention apiVersion, got: %v", err)
	}
}

func TestValidateWrongAPIVersion(t *testing.T) {
	yaml := strings.ReplaceAll(validManifest, "nlab.io/v1alpha1", "nlab.io/v2")
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for wrong apiVersion, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported apiVersion") {
		t.Errorf("error should mention unsupported apiVersion, got: %v", err)
	}
}

func TestValidateMissingKind(t *testing.T) {
	yaml := strings.ReplaceAll(validManifest, "kind: Stack\n", "")
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing kind, got nil")
	}
	if !strings.Contains(err.Error(), "kind") {
		t.Errorf("error should mention kind, got: %v", err)
	}
}

func TestValidateWrongKind(t *testing.T) {
	yaml := strings.ReplaceAll(validManifest, "kind: Stack", "kind: Network")
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for wrong kind, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported kind") {
		t.Errorf("error should mention unsupported kind, got: %v", err)
	}
}

func TestValidateMissingMetadataName(t *testing.T) {
	yaml := strings.ReplaceAll(validManifest, "  name: basic\n", "")
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing metadata.name, got nil")
	}
	if !strings.Contains(err.Error(), "metadata.name") {
		t.Errorf("error should mention metadata.name, got: %v", err)
	}
}

func TestValidateMissingNetworks(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  vms:
    attacker:
      xml: |
        <domain type="kvm"><name>attacker</name></domain>
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing networks, got nil")
	}
	if !strings.Contains(err.Error(), "networks") {
		t.Errorf("error should mention networks, got: %v", err)
	}
}

func TestValidateMissingVMs(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  networks:
    net:
      xml: |
        <network><name>net</name></network>
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing VMs, got nil")
	}
	if !strings.Contains(err.Error(), "vms") {
		t.Errorf("error should mention vms, got: %v", err)
	}
}

func TestValidateMissingNetworkXML(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  networks:
    basic_net: {}
  vms:
    attacker:
      xml: |
        <domain type="kvm"><name>attacker</name></domain>
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing network xml, got nil")
	}
	if !strings.Contains(err.Error(), "basic_net") {
		t.Errorf("error should mention network name, got: %v", err)
	}
}

func TestValidateMissingVMXML(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  networks:
    net:
      xml: |
        <network><name>net</name></network>
  vms:
    attacker: {}
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for missing vm xml, got nil")
	}
	if !strings.Contains(err.Error(), "attacker") {
		t.Errorf("error should mention vm name, got: %v", err)
	}
}

func TestValidateMalformedNetworkXML(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  networks:
    net:
      xml: "<network><name>net</name>"
  vms:
    attacker:
      xml: |
        <domain type="kvm"><name>attacker</name></domain>
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for malformed network xml, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("error should mention malformed, got: %v", err)
	}
}

func TestValidateMalformedVMXML(t *testing.T) {
	yaml := `
apiVersion: nlab.io/v1alpha1
kind: Stack
metadata:
  name: test
spec:
  networks:
    net:
      xml: |
        <network><name>net</name></network>
  vms:
    attacker:
      xml: "<domain type=\"kvm\"><name>attacker</name>"
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for malformed vm xml, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("error should mention malformed, got: %v", err)
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	yaml := `
kind: Stack
metadata:
  name: test
spec:
  networks:
    net:
      xml: "<network><name>net</name>"
  vms:
    attacker: {}
`
	_, err := manifest.LoadBytes([]byte(yaml), "test")
	if err == nil {
		t.Error("expected error for multiple validation issues, got nil")
	}
	// Should mention both apiVersion and xml issues
	if !strings.Contains(err.Error(), "apiVersion") {
		t.Errorf("error should mention apiVersion, got: %v", err)
	}
}
