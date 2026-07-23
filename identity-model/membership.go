package identitymodel

import (
	"slices"
	"time"
)

// Membership links an Actor to a Tenant with scoped roles.
type Membership struct {
	ActorID  ActorID   `json:"actor_id"`
	TenantID TenantID  `json:"tenant_id"`
	Roles    []Role    `json:"roles"`
	AddedAt  time.Time `json:"added_at"`
}

// HasRole reports whether the membership grants the given role.
func (m Membership) HasRole(role Role) bool {
	return slices.Contains(m.Roles, role)
}

// HasAnyRole reports whether the membership grants any of the given roles.
func (m Membership) HasAnyRole(roles ...Role) bool {
	for _, target := range roles {
		if m.HasRole(target) {
			return true
		}
	}
	return false
}
