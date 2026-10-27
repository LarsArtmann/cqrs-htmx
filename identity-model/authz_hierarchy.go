package identitymodel

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// EnsureRoleHierarchy mirrors DefaultRoleHierarchy into the domain-scoped `g`
// role manager for one domain.
//
// Casbin keeps one role manager per role definition. A grant written to `g`
// (subject, role, domain) and an inheritance edge written to `g2` (role, role)
// therefore never compose: before this existed, a user granted `user` was not
// allowed what `viewer` is allowed, even though the model documents
// super_admin > admin > user > viewer. The g2 seed still ships (it documents
// the hierarchy and serves consumers who bring their own model), but the edges
// that actually fire are the ones mirrored into `g` for the grant's domain.
//
// Idempotent. Grants replay through AddGroupPolicy, so rebuilding the Casbin
// projection re-seeds every domain it touches; call this directly only when
// loading policies from an external store that predates the mirror.
func (a *Authz) EnsureRoleHierarchy(domain TenantID) error {
	return a.ensureRoleHierarchy(domain.Get())
}

func (a *Authz) ensureRoleHierarchy(domain string) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}

	for _, h := range DefaultRoleHierarchy() {
		if _, err := a.enforcer.AddGroupingPolicy(string(h.From), string(h.To), domain); err != nil {
			return errorfamily.Wrapf(
				err, event.Transient, "casbin_error",
				"seed hierarchy %s -> %s in domain %s", h.From, h.To, domain,
			)
		}
	}

	return nil
}
