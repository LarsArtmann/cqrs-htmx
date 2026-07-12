package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// RolesForUser returns the directly assigned roles for a user in the given domain.
func (a *Authz) RolesForUser(userID UserID, domain TenantID) ([]Role, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	roles, err := a.enforcer.GetRolesForUser(userID.Get().String(), domain.Get())
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "domain="+domain.Get())
	}
	return convertRoles(roles), nil
}

// ImplicitRolesForUser returns all roles inherited (transitively) by the user in the domain.
func (a *Authz) ImplicitRolesForUser(userID UserID, domain TenantID) ([]Role, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	roles, err := a.enforcer.GetImplicitRolesForUser(userID.Get().String(), domain.Get())
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "domain="+domain.Get())
	}
	return convertRoles(roles), nil
}

// ImplicitPermissionsForUser returns all permissions the user has in the domain,
// including those inherited through role hierarchy.
func (a *Authz) ImplicitPermissionsForUser(userID UserID, domain TenantID) ([][]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	p, err := a.enforcer.GetImplicitPermissionsForUser(userID.Get().String(), domain.Get())
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err, "casbin_error",
			"implicit permissions domain="+domain.Get(),
		)
	}
	return p, nil
}

// DomainsForUser returns all domains the user has roles in.
func (a *Authz) DomainsForUser(userID UserID) ([]TenantID, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	d, err := a.enforcer.GetDomainsForUser(userID.Get().String())
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "domains for user")
	}
	filtered := make([]TenantID, 0, len(d))
	for _, dom := range d {
		if dom != "" {
			filtered = append(filtered, NewTenantID(dom))
		}
	}
	return filtered, nil
}

// UsersForRole returns all user IDs that have the given role in the domain.
func (a *Authz) UsersForRole(role Role, domain TenantID) ([]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	u, err := a.enforcer.GetUsersForRole(string(role), domain.Get())
	if err != nil {
		return nil, errorfamily.Wrapf(
			err, event.Transient, "casbin_error",
			"users for role %s domain=%s", role, domain.Get(),
		)
	}
	return u, nil
}

// RolesForActor returns the direct roles assigned to an actor in a tenant.
// Uses the ActorID's raw string value as the Casbin subject.
func (a *Authz) RolesForActor(actorID ActorID, tenantID TenantID) ([]Role, error) {
	return a.RolesForUser(NewUserID(actorID.String()), tenantID)
}

// ImplicitRolesForActor returns all roles assigned to an actor in a tenant,
// including those inherited via the g2 role hierarchy.
func (a *Authz) ImplicitRolesForActor(actorID ActorID, tenantID TenantID) ([]Role, error) {
	return a.ImplicitRolesForUser(NewUserID(actorID.String()), tenantID)
}

func defaultPolicies() []Policy {
	return []Policy{
		{RoleAdmin, "*", "*", ActionAll, EffectAllow},
		{RoleSuperAdmin, "*", "*", ActionAll, EffectAllow},
	}
}

// defaultRoleHierarchy returns the g2 role inheritance policies that enable
// the hierarchy: super_admin > admin > user > viewer.
// These are global (not domain-scoped) and applied once at Authz creation.
func defaultRoleHierarchy() []struct {
	From Role
	To   Role
} {
	return []struct {
		From Role
		To   Role
	}{
		{RoleSuperAdmin, RoleAdmin},
		{RoleAdmin, RoleUser},
		{RoleUser, RoleViewer},
	}
}
