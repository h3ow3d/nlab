package engine

import (
	"fmt"

	"github.com/h3ow3d/nlab/internal/types"
)

// get is the internal implementation of Get.
// It is a read-only diagnostic operation; it never modifies provider state.
func (e *Engine) get(m *types.StackManifest) ([]ResourceInfo, error) {
	var results []ResourceInfo

	for _, logicalName := range sortedKeys(m.Spec.Networks) {
		spec := m.Spec.Networks[logicalName]
		info, err := e.getNetwork(logicalName, spec.XML)
		if err != nil {
			return nil, err
		}
		results = append(results, info)
	}

	for _, logicalName := range sortedKeys(m.Spec.VMs) {
		spec := m.Spec.VMs[logicalName]
		info, err := e.getVM(logicalName, spec.XML)
		if err != nil {
			return nil, err
		}
		results = append(results, info)
	}

	return results, nil
}

func (e *Engine) getNetwork(logicalName, xmlStr string) (ResourceInfo, error) {
	info := ResourceInfo{Kind: "network", Name: logicalName}

	netName, err := networkNameFromXML(xmlStr)
	if err != nil {
		return info, err
	}
	info.Domain = netName

	if !e.p.NetworkDefined(netName) {
		return info, nil
	}
	info.Defined = true
	info.Active = e.p.NetworkActive(netName)
	if info.Active {
		info.State = "active"
	} else {
		info.State = "inactive"
	}

	markers, err := e.p.NetworkMarkers(netName)
	if err != nil {
		return info, fmt.Errorf("read markers for network %s: %w", netName, err)
	}
	info.Managed = markers.Managed
	info.Stack = markers.Stack
	return info, nil
}

func (e *Engine) getVM(logicalName, xmlStr string) (ResourceInfo, error) {
	info := ResourceInfo{Kind: "vm", Name: logicalName}

	domName, err := domainNameFromXML(xmlStr)
	if err != nil {
		return info, err
	}
	info.Domain = domName

	if !e.p.DomainDefined(domName) {
		return info, nil
	}
	info.Defined = true
	info.Active = e.p.DomainActive(domName)
	if info.Active {
		info.State = "running"
	} else {
		info.State = "shut off"
	}

	markers, err := e.p.DomainMarkers(domName)
	if err != nil {
		return info, fmt.Errorf("read markers for domain %s: %w", domName, err)
	}
	info.Managed = markers.Managed
	info.Stack = markers.Stack
	return info, nil
}
