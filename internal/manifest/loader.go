// Package manifest provides loading and validation for nlab v1alpha1 stack manifests.
package manifest

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/h3ow3d/nlab/internal/types"
)

const (
	supportedAPIVersion = "nlab.io/v1alpha1"
	supportedKind       = "Stack"
)

// Load reads a manifest file from path, parses it, and validates it.
// It returns the parsed StackManifest or an error with actionable guidance.
func Load(path string) (*types.StackManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read manifest %q: %w", path, err)
	}
	return LoadBytes(data, path)
}

// LoadBytes parses and validates a manifest from raw YAML bytes.
// The source parameter is used only for error messages.
func LoadBytes(data []byte, source string) (*types.StackManifest, error) {
	var m types.StackManifest
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("manifest %q: YAML parse error: %w", source, err)
	}
	if err := Validate(&m, source); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks a parsed StackManifest for correctness and returns a
// descriptive error if any issue is found.
func Validate(m *types.StackManifest, source string) error {
	var errs []string

	// Schema / version checks.
	if m.APIVersion == "" {
		errs = append(errs, "missing required field: apiVersion (expected \"nlab.io/v1alpha1\")")
	} else if m.APIVersion != supportedAPIVersion {
		errs = append(errs, fmt.Sprintf("unsupported apiVersion %q: only %q is supported", m.APIVersion, supportedAPIVersion))
	}

	if m.Kind == "" {
		errs = append(errs, "missing required field: kind (expected \"Stack\")")
	} else if m.Kind != supportedKind {
		errs = append(errs, fmt.Sprintf("unsupported kind %q: only %q is supported", m.Kind, supportedKind))
	}

	// Metadata checks.
	if m.Metadata.Name == "" {
		errs = append(errs, "missing required field: metadata.name")
	}

	// spec.networks checks.
	if len(m.Spec.Networks) == 0 {
		errs = append(errs, "spec.networks: at least one network is required")
	}
	for name, net := range m.Spec.Networks {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "spec.networks: network name must not be empty or whitespace-only")
			continue
		}
		if net.XML == "" {
			errs = append(errs, fmt.Sprintf("spec.networks.%s: xml field is required", name))
		} else if xmlErr := validateXML(net.XML); xmlErr != nil {
			errs = append(errs, fmt.Sprintf("spec.networks.%s: xml is malformed: %v", name, xmlErr))
		}
	}

	// spec.vms checks.
	if len(m.Spec.VMs) == 0 {
		errs = append(errs, "spec.vms: at least one VM is required")
	}
	for name, vm := range m.Spec.VMs {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "spec.vms: VM name must not be empty or whitespace-only")
			continue
		}
		if vm.XML == "" {
			errs = append(errs, fmt.Sprintf("spec.vms.%s: xml field is required", name))
		} else if xmlErr := validateXML(vm.XML); xmlErr != nil {
			errs = append(errs, fmt.Sprintf("spec.vms.%s: xml is malformed: %v", name, xmlErr))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("manifest %q is invalid:\n  - %s", source, strings.Join(errs, "\n  - "))
	}
	return nil
}

// validateXML checks that s is well-formed XML.
func validateXML(s string) error {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
