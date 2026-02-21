package lab

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Layout is the top-level structure of a layout.yaml file.
type Layout struct {
	Layout string `yaml:"layout"`
	Panes  []Pane `yaml:"panes"`
}

// Pane describes one tmux pane.
type Pane struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`    // "ssh" | "command"
	VM      string `yaml:"vm"`      // set when Type == "ssh"
	Command string `yaml:"command"` // set when Type == "command"
}

// LoadLayout reads and parses a layout.yaml file.
func LoadLayout(path string) (*Layout, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read layout file: %w", err)
	}
	var l Layout
	if err := yaml.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse layout file: %w", err)
	}
	if l.Layout == "" {
		l.Layout = "tiled"
	}
	return &l, nil
}

// SSHVMs returns the deduplicated list of VM names whose panes have type "ssh".
func (l *Layout) SSHVMs() []string {
	seen := make(map[string]bool)
	var out []string
	for _, p := range l.Panes {
		if p.Type == "ssh" && !seen[p.VM] {
			seen[p.VM] = true
			out = append(out, p.VM)
		}
	}
	return out
}

// ExpandCommand substitutes {stack} in a pane command with the given stack name.
func ExpandCommand(cmd, stack string) string {
	return strings.ReplaceAll(cmd, "{stack}", stack)
}
