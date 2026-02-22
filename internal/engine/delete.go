package engine

import (
	"fmt"
	"strings"

	"github.com/h3ow3d/nlab/internal/types"
)

// delete is the internal implementation of Delete.
//
// Safety rules (in priority order):
//  1. A resource owned by a *different* stack is never deleted (no override).
//  2. An unmanaged resource is refused unless opts.Force is true.
//  3. opts.Purge removes overlay disks / volumes when undefining domains.
//
// VMs are deleted before networks so that networks are not in use when removed.
func (e *Engine) delete(m *types.StackManifest, opts DeleteOptions) error {
	stack := m.Metadata.Name
	var errs []string

	for _, logicalName := range sortedKeys(m.Spec.VMs) {
		spec := m.Spec.VMs[logicalName]
		if err := e.deleteVM(stack, logicalName, spec.XML, opts); err != nil {
			errs = append(errs, fmt.Sprintf("vm %s: %v", logicalName, err))
		}
	}

	for _, logicalName := range sortedKeys(m.Spec.Networks) {
		spec := m.Spec.Networks[logicalName]
		if err := e.deleteNetwork(stack, logicalName, spec.XML, opts); err != nil {
			errs = append(errs, fmt.Sprintf("network %s: %v", logicalName, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("delete encountered errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (e *Engine) deleteNetwork(stack, _ /* logicalName */, xmlStr string, opts DeleteOptions) error {
	netName, err := networkNameFromXML(xmlStr)
	if err != nil {
		return err
	}

	if !e.p.NetworkDefined(netName) {
		return nil // already gone
	}

	markers, err := e.p.NetworkMarkers(netName)
	if err != nil {
		return fmt.Errorf("read markers for network %s: %w", netName, err)
	}

	if markers.Managed && markers.Stack != stack {
		return fmt.Errorf("network %s is owned by stack %q, not %q; refusing to delete", netName, markers.Stack, stack)
	}
	if !markers.Managed && !opts.Force {
		return fmt.Errorf("network %s has no nlab ownership markers; use --force to delete anyway", netName)
	}

	if e.p.NetworkActive(netName) {
		if err := e.p.StopNetwork(netName); err != nil {
			return fmt.Errorf("stop network %s: %w", netName, err)
		}
	}
	return e.p.UndefineNetwork(netName)
}

func (e *Engine) deleteVM(stack, _ /* logicalName */, xmlStr string, opts DeleteOptions) error {
	domName, err := domainNameFromXML(xmlStr)
	if err != nil {
		return err
	}

	if !e.p.DomainDefined(domName) {
		return nil // already gone
	}

	markers, err := e.p.DomainMarkers(domName)
	if err != nil {
		return fmt.Errorf("read markers for domain %s: %w", domName, err)
	}

	if markers.Managed && markers.Stack != stack {
		return fmt.Errorf("domain %s is owned by stack %q, not %q; refusing to delete", domName, markers.Stack, stack)
	}
	if !markers.Managed && !opts.Force {
		return fmt.Errorf("domain %s has no nlab ownership markers; use --force to delete anyway", domName)
	}

	if e.p.DomainActive(domName) {
		if err := e.p.StopDomain(domName); err != nil {
			return fmt.Errorf("stop domain %s: %w", domName, err)
		}
	}
	return e.p.UndefineDomain(domName, opts.Purge)
}
