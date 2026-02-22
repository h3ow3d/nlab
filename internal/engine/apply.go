package engine

import (
	"fmt"
	"sort"

	"github.com/h3ow3d/nlab/internal/types"
)

// apply is the internal implementation of Apply.
func (e *Engine) apply(m *types.StackManifest) error {
	stack := m.Metadata.Name

	// Networks first (sorted for determinism).
	for _, logicalName := range sortedKeys(m.Spec.Networks) {
		spec := m.Spec.Networks[logicalName]
		if err := e.applyNetwork(stack, logicalName, spec.XML); err != nil {
			return fmt.Errorf("apply network %s: %w", logicalName, err)
		}
	}

	// VMs second (sorted for determinism).
	for _, logicalName := range sortedKeys(m.Spec.VMs) {
		spec := m.Spec.VMs[logicalName]
		if err := e.applyVM(stack, logicalName, spec.XML); err != nil {
			return fmt.Errorf("apply vm %s: %w", logicalName, err)
		}
	}

	return nil
}

func (e *Engine) applyNetwork(stack, logicalName, xmlStr string) error {
	netName, err := networkNameFromXML(xmlStr)
	if err != nil {
		return err
	}

	if e.p.NetworkDefined(netName) {
		markers, err := e.p.NetworkMarkers(netName)
		if err != nil {
			return fmt.Errorf("read markers for network %s: %w", netName, err)
		}
		if !markers.Managed {
			return fmt.Errorf("network %s already exists but is not managed by nlab; rename it or remove it manually", netName)
		}
		if markers.Stack != stack {
			return fmt.Errorf("network %s is managed by stack %q, not %q; refusing to overwrite", netName, markers.Stack, stack)
		}
		// Already managed by this stack — idempotent: ensure it is active.
		if !e.p.NetworkActive(netName) {
			return e.p.StartNetwork(netName)
		}
		return nil
	}

	patched, err := InjectMarkers(xmlStr, "network", logicalName, stack)
	if err != nil {
		return fmt.Errorf("inject markers into network XML: %w", err)
	}
	if err := e.p.DefineNetwork(patched); err != nil {
		return err
	}
	if err := e.p.StartNetwork(netName); err != nil {
		return err
	}
	return e.p.AutostartNetwork(netName)
}

func (e *Engine) applyVM(stack, logicalName, xmlStr string) error {
	domName, err := domainNameFromXML(xmlStr)
	if err != nil {
		return err
	}

	if e.p.DomainDefined(domName) {
		markers, err := e.p.DomainMarkers(domName)
		if err != nil {
			return fmt.Errorf("read markers for domain %s: %w", domName, err)
		}
		if !markers.Managed {
			return fmt.Errorf("domain %s already exists but is not managed by nlab; rename it or remove it manually", domName)
		}
		if markers.Stack != stack {
			return fmt.Errorf("domain %s is managed by stack %q, not %q; refusing to overwrite", domName, markers.Stack, stack)
		}
		// Already managed by this stack — idempotent: ensure it is running.
		if !e.p.DomainActive(domName) {
			return e.p.StartDomain(domName)
		}
		return nil
	}

	patched, err := InjectMarkers(xmlStr, "vm", logicalName, stack)
	if err != nil {
		return fmt.Errorf("inject markers into domain XML: %w", err)
	}
	if err := e.p.DefineDomain(patched); err != nil {
		return err
	}
	return e.p.StartDomain(domName)
}

// sortedKeys returns the keys of a map[string]V in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
