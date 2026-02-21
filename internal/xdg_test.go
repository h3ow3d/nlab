package lab_test

import (
	"os"
	"path/filepath"
	"testing"

	lab "github.com/h3ow3d/nlab/internal"
)

func TestDefaultXDGDirs_Structure(t *testing.T) {
	dirs := lab.DefaultXDGDirs()

	if dirs.Config == "" {
		t.Error("Config must not be empty")
	}
	if dirs.Data == "" {
		t.Error("Data must not be empty")
	}
	if dirs.State == "" {
		t.Error("State must not be empty")
	}
}

func TestXDGDirs_SubPaths(t *testing.T) {
	dirs := lab.XDGDirs{
		Config: "/tmp/cfg/nlab",
		Data:   "/tmp/data/nlab",
		State:  "/tmp/state/nlab",
	}

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"ConfigFile", dirs.ConfigFile(), "/tmp/cfg/nlab/config.yaml"},
		{"ImagesDir", dirs.ImagesDir(), "/tmp/data/nlab/images"},
		{"StacksDir", dirs.StacksDir(), "/tmp/data/nlab/stacks"},
		{"CloudInitDir", dirs.CloudInitDir(), "/tmp/data/nlab/cloudinit"},
		{"LogsDir", dirs.LogsDir(), "/tmp/state/nlab/logs"},
		{"PcapDir", dirs.PcapDir(), "/tmp/state/nlab/pcap"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestXDGDirs_XDGEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	dirs := lab.DefaultXDGDirs()

	if dirs.Config != filepath.Join(tmp, "config", "nlab") {
		t.Errorf("Config = %q, want %q", dirs.Config, filepath.Join(tmp, "config", "nlab"))
	}
	if dirs.Data != filepath.Join(tmp, "data", "nlab") {
		t.Errorf("Data = %q, want %q", dirs.Data, filepath.Join(tmp, "data", "nlab"))
	}
	if dirs.State != filepath.Join(tmp, "state", "nlab") {
		t.Errorf("State = %q, want %q", dirs.State, filepath.Join(tmp, "state", "nlab"))
	}
}

func TestEnsureDirs_CreatesAll(t *testing.T) {
	tmp := t.TempDir()
	dirs := lab.XDGDirs{
		Config: filepath.Join(tmp, "config", "nlab"),
		Data:   filepath.Join(tmp, "data", "nlab"),
		State:  filepath.Join(tmp, "state", "nlab"),
	}

	if err := dirs.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}

	expected := []string{
		dirs.Config,
		dirs.ImagesDir(),
		dirs.StacksDir(),
		dirs.CloudInitDir(),
		dirs.LogsDir(),
		dirs.PcapDir(),
	}
	for _, d := range expected {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}
}

func TestEnsureDirs_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	dirs := lab.XDGDirs{
		Config: filepath.Join(tmp, "config", "nlab"),
		Data:   filepath.Join(tmp, "data", "nlab"),
		State:  filepath.Join(tmp, "state", "nlab"),
	}

	if err := dirs.EnsureDirs(); err != nil {
		t.Fatalf("first EnsureDirs: %v", err)
	}
	if err := dirs.EnsureDirs(); err != nil {
		t.Fatalf("second EnsureDirs (idempotency): %v", err)
	}
}
