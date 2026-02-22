package lab

import (
	"encoding/xml"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// StackConfig is the top-level structure of a stacks/<name>/stack.yaml file.
type StackConfig struct {
	Network    string   `yaml:"network"`
	NetworkXML string   `yaml:"-"` // populated from v1alpha1 spec.networks.<name>.xml
	VMs        []VMSpec `yaml:"vms"`
}

// VMSpec describes one VM within a stack.
type VMSpec struct {
	Name   string `yaml:"name"`
	Memory int    `yaml:"memory"` // MiB
	VCPUs  int    `yaml:"vcpus"`
}

// LoadStack reads stacks/<name>/stack.yaml and returns the parsed StackConfig.
// It supports both the legacy flat format and the v1alpha1 manifest format.
func LoadStack(stackName string) (*StackConfig, error) {
	path := fmt.Sprintf("stacks/%s/stack.yaml", stackName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stack config %s: %w", path, err)
	}

	// Detect v1alpha1 format by checking for top-level apiVersion and kind keys.
	var meta struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &meta); err == nil && meta.APIVersion != "" && meta.Kind != "" {
		return loadStackV1alpha1(data, path)
	}

	// Legacy flat format.
	var cfg StackConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse stack config %s: %w", path, err)
	}
	if cfg.Network == "" {
		return nil, fmt.Errorf("stack config %s: network field is required", path)
	}
	if len(cfg.VMs) == 0 {
		return nil, fmt.Errorf("stack config %s: at least one vm is required", path)
	}
	return &cfg, nil
}

// v1alpha1Raw is used to unmarshal only the fields LoadStack needs from a
// v1alpha1 manifest, without importing the manifest package (avoiding cycles).
type v1alpha1Raw struct {
	Spec struct {
		Networks map[string]struct {
			XML string `yaml:"xml"`
		} `yaml:"networks"`
		VMs map[string]struct {
			XML string `yaml:"xml"`
		} `yaml:"vms"`
	} `yaml:"spec"`
}

// domainMemVCPU is a minimal representation used to extract memory and vcpu
// from a libvirt domain XML fragment.
type domainMemVCPU struct {
	Memory struct {
		Unit  string `xml:"unit,attr"`
		Value int    `xml:",chardata"`
	} `xml:"memory"`
	VCPU int `xml:"vcpu"`
}

func loadStackV1alpha1(data []byte, path string) (*StackConfig, error) {
	var raw v1alpha1Raw
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse stack config %s: %w", path, err)
	}

	if len(raw.Spec.Networks) == 0 {
		return nil, fmt.Errorf("stack config %s: spec.networks is required", path)
	}
	if len(raw.Spec.VMs) == 0 {
		return nil, fmt.Errorf("stack config %s: spec.vms is required", path)
	}

	// Use the first network (sorted alphabetically for deterministic selection).
	// v1alpha1 manifests typically define a single network; if multiple are
	// present the alphabetically first name is chosen.
	var networkNames []string
	for name := range raw.Spec.Networks {
		networkNames = append(networkNames, name)
	}
	sort.Strings(networkNames)
	networkName := networkNames[0]
	networkXML := strings.TrimSpace(raw.Spec.Networks[networkName].XML)

	if networkXML == "" {
		return nil, fmt.Errorf("stack config %s: spec.networks.%s.xml is required and must be non-empty", path, networkName)
	}

	cfg := &StackConfig{Network: networkName, NetworkXML: networkXML}

	for name, vm := range raw.Spec.VMs {
		spec := VMSpec{Name: name}
		if vm.XML != "" {
			var d domainMemVCPU
			if err := xml.Unmarshal([]byte(vm.XML), &d); err != nil {
				return nil, fmt.Errorf("stack config %s: parse VM %s domain XML: %w", path, name, err)
			}
			mem := d.Memory.Value
			// Normalise to MiB — libvirt default unit is KiB.
			switch strings.ToLower(d.Memory.Unit) {
			case "kib", "k", "":
				mem /= 1024
			case "mib", "m":
				// already MiB
			case "gib", "g":
				mem *= 1024
			}
			spec.Memory = mem
			spec.VCPUs = d.VCPU
		}
		cfg.VMs = append(cfg.VMs, spec)
	}

	return cfg, nil
}
