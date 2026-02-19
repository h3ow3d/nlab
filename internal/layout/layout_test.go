package layout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/h3ow3d/nlab/internal/layout"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	yaml := `layout: even-horizontal
panes:
  - name: attacker
    type: ssh
    vm: attacker
  - name: target
    type: ssh
    vm: target
  - name: monitor
    type: command
    command: "sudo tcpdump -i virbr-{stack} -nn"
`
	path := writeFile(t, dir, "layout.yaml", yaml)

	l, err := layout.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if l.Layout != "even-horizontal" {
		t.Errorf("Layout = %q, want even-horizontal", l.Layout)
	}
	if len(l.Panes) != 3 {
		t.Fatalf("len(Panes) = %d, want 3", len(l.Panes))
	}
	if l.Panes[0].Type != "ssh" || l.Panes[0].VM != "attacker" {
		t.Errorf("Pane[0] = %+v", l.Panes[0])
	}
	if l.Panes[2].Type != "command" {
		t.Errorf("Pane[2].Type = %q, want command", l.Panes[2].Type)
	}
}

func TestLoadDefaultLayout(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "layout.yaml", "panes:\n  - type: command\n    command: echo hi\n")

	l, err := layout.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if l.Layout != "tiled" {
		t.Errorf("default Layout = %q, want tiled", l.Layout)
	}
}

func TestSSHVMs(t *testing.T) {
	dir := t.TempDir()
	yaml := `layout: tiled
panes:
  - type: ssh
    vm: a
  - type: ssh
    vm: b
  - type: ssh
    vm: a
  - type: command
    command: echo
`
	path := writeFile(t, dir, "layout.yaml", yaml)

	l, err := layout.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	vms := l.SSHVMs()
	if len(vms) != 2 {
		t.Fatalf("SSHVMs() = %v, want [a b]", vms)
	}
	if vms[0] != "a" || vms[1] != "b" {
		t.Errorf("SSHVMs() = %v, want [a b]", vms)
	}
}

func TestExpandCommand(t *testing.T) {
	tests := []struct {
		cmd, stack, want string
	}{
		{"sudo tcpdump -i virbr-{stack} -nn", "basic", "sudo tcpdump -i virbr-basic -nn"},
		{"echo hello", "mystack", "echo hello"},
		{"{stack}/{stack}", "x", "x/x"},
	}
	for _, tc := range tests {
		got := layout.ExpandCommand(tc.cmd, tc.stack)
		if got != tc.want {
			t.Errorf("ExpandCommand(%q, %q) = %q, want %q", tc.cmd, tc.stack, got, tc.want)
		}
	}
}
