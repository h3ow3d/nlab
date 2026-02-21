package lab

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// StackConfig is the top-level structure of a stacks/<name>/stack.yaml file.
type StackConfig struct {
	Network string   `yaml:"network"`
	VMs     []VMSpec `yaml:"vms"`
}

// VMSpec describes one VM within a stack.
type VMSpec struct {
	Name   string `yaml:"name"`
	Memory int    `yaml:"memory"` // MiB
	VCPUs  int    `yaml:"vcpus"`
}

// LoadStack reads stacks/<name>/stack.yaml and returns the parsed StackConfig.
func LoadStack(stackName string) (*StackConfig, error) {
	path := fmt.Sprintf("stacks/%s/stack.yaml", stackName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stack config %s: %w", path, err)
	}
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
