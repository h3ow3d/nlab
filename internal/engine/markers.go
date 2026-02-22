package engine

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Marker key constants embedded in libvirt <description> elements.
const (
	keyManaged  = "nlab.io/managed"
	keyStack    = "nlab.io/stack"
	keyResource = "nlab.io/resource"
	keyName     = "nlab.io/name"
)

// Markers holds the nlab ownership annotations for a libvirt resource.
type Markers struct {
	Managed  bool
	Stack    string
	Resource string // "network" or "vm"
	Name     string // logical name from the manifest key
}

// IsOwnedBy returns true when the markers indicate ownership by the named stack.
func (m Markers) IsOwnedBy(stack string) bool {
	return m.Managed && m.Stack == stack
}

// markerString serialises markers as a single-line string for embedding in a
// libvirt <description> element.
func markerString(resource, name, stack string) string {
	return fmt.Sprintf("%s=true %s=%s %s=%s %s=%s",
		keyManaged, keyStack, stack, keyResource, resource, keyName, name)
}

// ParseMarkersFromDesc parses nlab ownership markers from the text content of
// a libvirt <description> element.
func ParseMarkersFromDesc(desc string) Markers {
	var m Markers
	for _, part := range strings.Fields(desc) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case keyManaged:
			m.Managed = kv[1] == "true"
		case keyStack:
			m.Stack = kv[1]
		case keyResource:
			m.Resource = kv[1]
		case keyName:
			m.Name = kv[1]
		}
	}
	return m
}

// ParseMarkersFromXML returns markers embedded in the XML returned by
// virsh dumpxml / virsh net-dumpxml.
func ParseMarkersFromXML(xmlStr string) Markers {
	type doc struct {
		Description string `xml:"description"`
	}
	var d doc
	if err := xml.Unmarshal([]byte(xmlStr), &d); err != nil {
		return Markers{}
	}
	return ParseMarkersFromDesc(d.Description)
}

// InjectMarkers returns xmlStr with nlab ownership markers set in the
// <description> child of the root element.  Any existing <description> is
// replaced; all other content is preserved verbatim.
func InjectMarkers(xmlStr, resource, name, stack string) (string, error) {
	return setDescription(xmlStr, markerString(resource, name, stack))
}

// setDescription replaces or inserts a <description>text</description> inside
// xmlStr.  It operates with simple string manipulation to preserve formatting
// and avoid lossy XML round-tripping.
func setDescription(xmlStr, text string) (string, error) {
	const openTag = "<description>"
	const closeTag = "</description>"

	lo := strings.Index(xmlStr, openTag)
	hi := strings.Index(xmlStr, closeTag)
	if lo != -1 && hi != -1 && hi > lo {
		// Replace existing description content.
		return xmlStr[:lo+len(openTag)] + text + xmlStr[hi:], nil
	}

	// No <description>: inject immediately after the first root-element >.
	idx := strings.Index(xmlStr, ">")
	if idx == -1 {
		return "", fmt.Errorf("malformed XML: no root-element closing bracket found")
	}
	return xmlStr[:idx+1] + "\n  <description>" + text + "</description>" + xmlStr[idx+1:], nil
}

// networkNameFromXML extracts the <name> child from a libvirt network XML.
func networkNameFromXML(xmlStr string) (string, error) {
	return nameFromXML(xmlStr, "network")
}

// domainNameFromXML extracts the <name> child from a libvirt domain XML.
func domainNameFromXML(xmlStr string) (string, error) {
	return nameFromXML(xmlStr, "domain")
}

// nameFromXML extracts the <name> child of the root element of xmlStr.
// The root parameter is used only in error messages.
func nameFromXML(xmlStr, root string) (string, error) {
	type named struct {
		Name string `xml:"name"`
	}
	var n named
	if err := xml.Unmarshal([]byte(xmlStr), &n); err != nil {
		return "", fmt.Errorf("parse %s XML: %w", root, err)
	}
	if n.Name == "" {
		return "", fmt.Errorf("%s XML is missing a <name> element", root)
	}
	return n.Name, nil
}
