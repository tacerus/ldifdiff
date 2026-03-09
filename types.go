package ldifdiff

import (
	"github.com/go-ldap/ldap/v3"
)

/* Types */

// Entry represents a mapping between a LDAP attribute name and the associated values.
type entry map[string][]string

// Entries represents a mapping between a LDAP DN and associated attribute maps.
type entries map[string]entry

// ActionEntry represents a batch of operations for a given DN.
type actionEntry struct {
	Dn string

	Add []*ldap.AddRequest
	Del []*ldap.DelRequest
	Mod []*ldap.ModifyRequest
}
