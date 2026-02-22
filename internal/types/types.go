// Package types defines the typed model for nlab stack manifests (v1alpha1).
package types

// StackManifest is the top-level structure of a v1alpha1 stack manifest.
type StackManifest struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
	Spec       StackSpec  `yaml:"spec"`
}

// ObjectMeta holds identity metadata for a stack manifest.
type ObjectMeta struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// StackSpec is the spec section of a StackManifest.
type StackSpec struct {
	Networks map[string]NetworkSpec         `yaml:"networks"`
	VMs      map[string]VMSpec              `yaml:"vms"`
	Storage  map[string]interface{}         `yaml:"storage"`
	Tmux     map[string]interface{}         `yaml:"tmux"`
	Defaults map[string]interface{}         `yaml:"defaults"`
}

// NetworkSpec describes a single libvirt network resource.
type NetworkSpec struct {
	XML string `yaml:"xml"`
}

// VMSpec describes a single libvirt domain (VM) resource.
type VMSpec struct {
	XML string `yaml:"xml"`
}
