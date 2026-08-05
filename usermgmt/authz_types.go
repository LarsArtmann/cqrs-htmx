package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// All authorization types, the Casbin-backed Authz engine, and the default RBAC
// model/policies/hierarchy now live in identity-model. These aliases re-export
// them from usermgmt for backward compatibility. Methods on *Authz (Enforce,
// Authorize, RolesForUser, AddPolicy, Apply, etc.) are inherited automatically
// through the type alias.

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Action = identitymodel.Action
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Effect = identitymodel.Effect
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Role = identitymodel.Role
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Policy = identitymodel.Policy
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	GroupPolicy = identitymodel.GroupPolicy
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EnforceResult = identitymodel.EnforceResult
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	PolicyUpdate = identitymodel.PolicyUpdate
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EnforcerConfig = identitymodel.EnforcerConfig
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	Authz = identitymodel.Authz
)

const (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActionExecute = identitymodel.ActionExecute
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActionRead = identitymodel.ActionRead
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	ActionAll = identitymodel.ActionAll
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EffectAllow = identitymodel.EffectAllow
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	EffectDeny = identitymodel.EffectDeny
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RoleSuperAdmin = identitymodel.RoleSuperAdmin
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RoleAdmin = identitymodel.RoleAdmin
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RoleUser = identitymodel.RoleUser
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RoleViewer = identitymodel.RoleViewer
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RoleOwner = identitymodel.RoleOwner
)

// NewAuthz creates an Authz with the given optional config.
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewAuthz(config ...EnforcerConfig) (*Authz, error) {
	return identitymodel.NewAuthz(config...)
}

// AssignableRoles returns roles grantable within a tenant (excludes super_admin).
// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func AssignableRoles() []Role {
	return identitymodel.AssignableRoles()
}
