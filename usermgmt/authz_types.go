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
	Action         = identitymodel.Action
	Effect         = identitymodel.Effect
	Role           = identitymodel.Role
	Policy         = identitymodel.Policy
	GroupPolicy    = identitymodel.GroupPolicy
	EnforceResult  = identitymodel.EnforceResult
	PolicyUpdate   = identitymodel.PolicyUpdate
	EnforcerConfig = identitymodel.EnforcerConfig
	Authz          = identitymodel.Authz
)

const (
	ActionExecute  = identitymodel.ActionExecute
	ActionRead     = identitymodel.ActionRead
	ActionAll      = identitymodel.ActionAll
	EffectAllow    = identitymodel.EffectAllow
	EffectDeny     = identitymodel.EffectDeny
	RoleSuperAdmin = identitymodel.RoleSuperAdmin
	RoleAdmin      = identitymodel.RoleAdmin
	RoleUser       = identitymodel.RoleUser
	RoleViewer     = identitymodel.RoleViewer
	RoleOwner      = identitymodel.RoleOwner
)

// NewAuthz creates an Authz with the given optional config.
func NewAuthz(cfg ...EnforcerConfig) (*Authz, error) {
	return identitymodel.NewAuthz(cfg...)
}

// AssignableRoles returns roles grantable within a tenant (excludes super_admin).
func AssignableRoles() []Role {
	return identitymodel.AssignableRoles()
}
