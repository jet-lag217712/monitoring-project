// Package vendors assembles the supported vendor profile registry.
package vendors

import (
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors/arista"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors/cisco"
)

// NewRegistry returns the immutable registry of supported vendor profiles.
func NewRegistry() *profile.Registry {
	return profile.NewRegistry(cisco.New(), arista.New())
}
