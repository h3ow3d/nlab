package lab_test

import (
	"path/filepath"
	"testing"

	lab "github.com/h3ow3d/nlab/internal"
)

func TestRunDoctorChecks_ReturnsResults(t *testing.T) {
	tmp := t.TempDir()
	dirs := lab.XDGDirs{
		Config: filepath.Join(tmp, "config", "nlab"),
		Data:   filepath.Join(tmp, "data", "nlab"),
		State:  filepath.Join(tmp, "state", "nlab"),
	}

	results := lab.RunDoctorChecks(dirs)
	if len(results) == 0 {
		t.Fatal("RunDoctorChecks returned no results")
	}

	// Every result must have a non-empty Name and Message.
	for _, r := range results {
		if r.Name == "" {
			t.Errorf("CheckResult has empty Name: %+v", r)
		}
		if r.Message == "" {
			t.Errorf("CheckResult %q has empty Message", r.Name)
		}
		// When a check fails it MUST include a HowToFix hint.
		if !r.OK && r.HowToFix == "" {
			t.Errorf("failed check %q is missing HowToFix hint", r.Name)
		}
	}
}

func TestRunDoctorChecks_XDGCheckPasses(t *testing.T) {
	tmp := t.TempDir()
	dirs := lab.XDGDirs{
		Config: filepath.Join(tmp, "config", "nlab"),
		Data:   filepath.Join(tmp, "data", "nlab"),
		State:  filepath.Join(tmp, "state", "nlab"),
	}

	results := lab.RunDoctorChecks(dirs)

	var xdgResult *lab.CheckResult
	for i := range results {
		if results[i].Name == "XDG directory access" {
			xdgResult = &results[i]
			break
		}
	}
	if xdgResult == nil {
		t.Fatal("XDG directory access check not found")
	}
	if !xdgResult.OK {
		t.Errorf("XDG directory access check failed: %s", xdgResult.Message)
	}
}

func TestRunDoctorChecks_XDGCheckFailsOnReadOnly(t *testing.T) {
	// Point all dirs at a path that cannot be created under a read-only root.
	dirs := lab.XDGDirs{
		Config: "/proc/nlab/config",
		Data:   "/proc/nlab/data",
		State:  "/proc/nlab/state",
	}

	results := lab.RunDoctorChecks(dirs)

	var xdgResult *lab.CheckResult
	for i := range results {
		if results[i].Name == "XDG directory access" {
			xdgResult = &results[i]
			break
		}
	}
	if xdgResult == nil {
		t.Fatal("XDG directory access check not found")
	}
	if xdgResult.OK {
		t.Error("expected XDG directory access check to fail for unwritable path")
	}
	if xdgResult.HowToFix == "" {
		t.Error("failed XDG check must provide a HowToFix hint")
	}
}
