// Package engine provides the orchestration logic for applying, deleting, and
// inspecting nlab stack resources against a Provider.
//
// The engine is intentionally decoupled from the provider so that it can be
// unit-tested without a running libvirt daemon. Real provider implementations
// live in internal/provider/libvirt.
package engine

import "github.com/h3ow3d/nlab/internal/types"

// DeleteOptions controls the behaviour of the Delete operation.
type DeleteOptions struct {
	// Force bypasses the managed-resource safety check and deletes resources
	// that carry no nlab ownership markers (or markers for a different stack).
	Force bool
	// Purge additionally removes overlay disks and stack-associated volumes.
	Purge bool
}

// ResourceInfo describes the runtime state of a single stack resource.
type ResourceInfo struct {
	Kind    string // "network" or "vm"
	Name    string // logical name from the manifest key
	Domain  string // libvirt resource name (from XML <name>)
	Defined bool   // exists in libvirt
	Active  bool   // currently running / active
	Managed bool   // carries nlab.io/managed=true
	Stack   string // value of nlab.io/stack marker
	State   string // free-form state string (e.g. "running", "shut off")
}

// Provider is the interface the engine uses to interact with the hypervisor.
// A real virsh-backed implementation lives in internal/provider/libvirt; a
// mock can be supplied in tests so the engine is testable in isolation.
type Provider interface {
	// Network operations.
	NetworkDefined(name string) bool
	NetworkActive(name string) bool
	NetworkMarkers(name string) (Markers, error)
	DefineNetwork(xmlStr string) error
	StartNetwork(name string) error
	AutostartNetwork(name string) error
	StopNetwork(name string) error
	UndefineNetwork(name string) error

	// Domain (VM) operations.
	DomainDefined(name string) bool
	DomainActive(name string) bool
	DomainMarkers(name string) (Markers, error)
	DefineDomain(xmlStr string) error
	StartDomain(name string) error
	StopDomain(name string) error
	UndefineDomain(name string, purge bool) error
}

// Engine orchestrates stack resources against a Provider.
type Engine struct {
	p Provider
}

// New returns an Engine backed by the given Provider.
func New(p Provider) *Engine {
	return &Engine{p: p}
}

// Apply implements the apply operation.  It delegates to apply.go.
func (e *Engine) Apply(m *types.StackManifest) error { return e.apply(m) }

// Delete implements the delete operation.  It delegates to delete.go.
func (e *Engine) Delete(m *types.StackManifest, opts DeleteOptions) error {
	return e.delete(m, opts)
}

// Get implements the get/status operation.  It delegates to get.go.
func (e *Engine) Get(m *types.StackManifest) ([]ResourceInfo, error) { return e.get(m) }
