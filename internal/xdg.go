package lab

import (
	"fmt"
	"os"
	"path/filepath"
)

// XDGDirs holds the resolved XDG-compliant directory paths for nlab.
type XDGDirs struct {
	// Config is ~/.config/nlab  (XDG_CONFIG_HOME)
	Config string
	// Data is ~/.local/share/nlab  (XDG_DATA_HOME)
	Data string
	// State is ~/.local/state/nlab  (XDG_STATE_HOME)
	State string
}

// xdgBase returns the XDG base directory, falling back to the given default
// when the environment variable is unset or empty.
func xdgBase(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fallback
	}
	return filepath.Join(home, fallback)
}

// DefaultXDGDirs returns the resolved XDG directory set for nlab using the
// current environment and home directory. All paths are absolute.
func DefaultXDGDirs() XDGDirs {
	return XDGDirs{
		Config: filepath.Join(xdgBase("XDG_CONFIG_HOME", ".config"), "nlab"),
		Data:   filepath.Join(xdgBase("XDG_DATA_HOME", ".local/share"), "nlab"),
		State:  filepath.Join(xdgBase("XDG_STATE_HOME", ".local/state"), "nlab"),
	}
}

// ConfigFile returns the path to the nlab config file.
func (d XDGDirs) ConfigFile() string {
	return filepath.Join(d.Config, "config.yaml")
}

// ImagesDir returns the base-image cache directory.
func (d XDGDirs) ImagesDir() string {
	return filepath.Join(d.Data, "images")
}

// StacksDir returns the optional stacks library directory.
func (d XDGDirs) StacksDir() string {
	return filepath.Join(d.Data, "stacks")
}

// CloudInitDir returns the generated cloud-init seeds directory.
func (d XDGDirs) CloudInitDir() string {
	return filepath.Join(d.Data, "cloudinit")
}

// LogsDir returns the logs directory.
func (d XDGDirs) LogsDir() string {
	return filepath.Join(d.State, "logs")
}

// PcapDir returns the packet-capture directory.
func (d XDGDirs) PcapDir() string {
	return filepath.Join(d.State, "pcap")
}

// EnsureDirs creates all nlab XDG directories that do not yet exist.
// Directories are created with mode 0700 so that only the owning user can
// read them (private data / state / config).
func (d XDGDirs) EnsureDirs() error {
	dirs := []string{
		d.Config,
		d.ImagesDir(),
		d.StacksDir(),
		d.CloudInitDir(),
		d.LogsDir(),
		d.PcapDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return nil
}
