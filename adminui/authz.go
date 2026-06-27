package adminui

import (
	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// defaultAuthorizer builds the access check used when [Config.Authorizer] is nil.
//
//   - Super Admin mode: the user must hold the super_admin or admin role in the
//     global ("*") domain.
//   - Tenant Admin mode: the user must hold the admin or owner role within
//     [Config.TenantID] (via the membership/Authz system).
//
// These defaults match the role vocabulary seeded by usermgmt. If your
// application assigns roles differently, pass your own [Config.Authorizer]
// (see [RequireAnyRole] and [RequireAuthenticated]).
func defaultAuthorizer(cfg Config) func(*usermgmt.User) error {
	if cfg.Mode == ModeTenantAdmin {
		return RequireAnyRole(cfg.Service, cfg.TenantID.Get(),
			usermgmt.RoleAdmin, usermgmt.RoleOwner)
	}
	return RequireAnyRole(cfg.Service, "*",
		usermgmt.RoleSuperAdmin, usermgmt.RoleAdmin)
}

// RequireAnyRole returns an authorizer that grants access when the user holds
// any of the given roles in domain (use "*" for a global check, or a tenant ID
// for a scoped check). A nil or unauthenticated user is always denied.
func RequireAnyRole(service *usermgmt.Service, domain string, roles ...usermgmt.Role) func(*usermgmt.User) error {
	want := make(map[usermgmt.Role]struct{}, len(roles))
	for _, r := range roles {
		want[r] = struct{}{}
	}
	return func(user *usermgmt.User) error {
		if user == nil {
			return errForbidden
		}
		held, err := service.Authz().ImplicitRolesForUser(user.ID, domain)
		if err != nil {
			return errForbidden
		}
		for _, r := range held {
			if _, ok := want[r]; ok {
				return nil
			}
		}
		return errForbidden
	}
}

// RequireAuthenticated returns an authorizer that grants access to any
// authenticated user, regardless of role. Use for low-trust panels or as a
// building block combined with additional checks.
func RequireAuthenticated() func(*usermgmt.User) error {
	return func(user *usermgmt.User) error {
		if user == nil {
			return errForbidden
		}
		return nil
	}
}
