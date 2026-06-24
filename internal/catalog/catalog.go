// Package catalog models the Home Assistant service catalog returned by the
// REST /api/services endpoint, used to describe and validate service calls.
package catalog

import "encoding/json"

// Field describes a single service field as advertised by Home Assistant.
type Field struct {
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Example     json.RawMessage `json:"example,omitempty"`
	Selector    json.RawMessage `json:"selector,omitempty"`
}

// Service describes one service within a domain.
type Service struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Fields      map[string]Field `json:"fields,omitempty"`
	Target      json.RawMessage  `json:"target,omitempty"`
}

// Domain groups services under a domain name.
type Domain struct {
	Domain   string             `json:"domain"`
	Services map[string]Service `json:"services"`
}

// Catalog is the full list of domains and their services.
type Catalog struct {
	Domains []Domain
}

// Parse decodes the /api/services payload into a Catalog.
func Parse(raw json.RawMessage) (*Catalog, error) {
	var domains []Domain
	if err := json.Unmarshal(raw, &domains); err != nil {
		return nil, err
	}
	return &Catalog{Domains: domains}, nil
}

// Lookup finds a service by domain and service name; ok is false if absent.
func (c *Catalog) Lookup(domain, service string) (Service, bool) {
	for _, d := range c.Domains {
		if d.Domain != domain {
			continue
		}
		svc, ok := d.Services[service]
		if ok && svc.Name == "" {
			svc.Name = service
		}
		return svc, ok
	}
	return Service{}, false
}
