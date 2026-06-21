package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// RolesForUser returns the directly assigned roles for a user in the given domain.
func (a *Authz) RolesForUser(userID UserID, domain string) ([]Role, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	roles, err := a.enforcer.GetRolesForUser(userID.Get(), domain)
	if err != nil {
		return nil, event.WrapTransient(err, "casbin_error", "domain="+domain)
	}
	return convertRoles(roles), nil
}

// ImplicitRolesForUser returns all roles inherited (transitively) by the user in the domain.
func (a *Authz) ImplicitRolesForUser(userID UserID, domain string) ([]Role, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	roles, err := a.enforcer.GetImplicitRolesForUser(userID.Get(), domain)
	if err != nil {
		return nil, event.WrapTransient(err, "casbin_error", "domain="+domain)
	}
	return convertRoles(roles), nil
}

// ImplicitPermissionsForUser returns all permissions the user has in the domain,
// including those inherited through role hierarchy.
func (a *Authz) ImplicitPermissionsForUser(userID UserID, domain string) ([][]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	p, err := a.enforcer.GetImplicitPermissionsForUser(userID.Get(), domain)
	if err != nil {
		return nil, event.WrapTransient(
			err, "casbin_error",
			"implicit permissions domain="+domain,
		)
	}
	return p, nil
}

// DomainsForUser returns all domains the user has roles in.
func (a *Authz) DomainsForUser(userID UserID) ([]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	d, err := a.enforcer.GetDomainsForUser(userID.Get())
	if err != nil {
		return nil, event.WrapTransient(err, "casbin_error", "domains for user")
	}
	filtered := d[:0]
	for _, dom := range d {
		if dom != "" {
			filtered = append(filtered, dom)
		}
	}
	return filtered, nil
}

// UsersForRole returns all user IDs that have the given role in the domain.
func (a *Authz) UsersForRole(role Role, domain string) ([]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	u, err := a.enforcer.GetUsersForRole(string(role), domain)
	if err != nil {
		return nil, event.Wrapf(
			err, event.Transient, "casbin_error",
			"users for role %s domain=%s", role, domain,
		)
	}
	return u, nil
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
