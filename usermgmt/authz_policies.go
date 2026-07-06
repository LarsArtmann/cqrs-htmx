package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// wrapCasbinError wraps an underlying casbin error with operation context.
// Centralizes the transient classification and "casbin_error" sentinel used
// across all casbin operations.
func wrapCasbinError(err error, op string, args ...any) error {
	return errorfamily.Wrapf(err, event.Transient, "casbin_error", op, args...)
}

// wrapPolicyError wraps an error from a Policy operation with the standard
// 5-field format "{subject, domain, object, action, effect}".
func wrapPolicyError(err error, op string, p Policy) error {
	return wrapCasbinError(err, op+" policy {%s, %s, %s, %s, %s}",
		p.Subject, p.Domain, p.Object, p.Action, p.Effect)
}

// wrapGroupError wraps an error from a GroupPolicy operation with the standard
// 3-field format "{subject, role, domain}".
func wrapGroupError(err error, op string, g GroupPolicy) error {
	return wrapCasbinError(err, op+" group {%s, %s, %s}",
		g.Subject, string(g.Role), g.Domain)
}

// Apply applies a batch of group and policy additions/removals sequentially.
// Operations are applied in order: add groups, add policies, remove groups, remove policies.
// Add-first ordering ensures that if remove fails mid-way, the user retains access
// rather than losing all permissions. If any operation fails mid-way, the policy
// state is partially updated — callers should treat this as a best-effort operation.
func (a *Authz) Apply(update PolicyUpdate) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	for _, g := range update.AddGroups {
		if _, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
			return wrapGroupError(err, "add", g)
		}
	}
	for _, p := range update.AddPolicies {
		if _, err := a.enforcer.AddPolicy(policyArgs(p)...); err != nil {
			return wrapPolicyError(err, "add", p)
		}
	}
	for _, g := range update.RemoveGroups {
		if _, err := a.enforcer.RemoveGroupingPolicy(
			g.Subject,
			string(g.Role),
			g.Domain,
		); err != nil {
			return wrapGroupError(err, "remove", g)
		}
	}
	for _, p := range update.RemovePolicies {
		if _, err := a.enforcer.RemovePolicy(policyArgs(p)...); err != nil {
			return wrapPolicyError(err, "remove", p)
		}
	}
	return nil
}

// AddPolicy adds a single RBAC policy rule.
func (a *Authz) AddPolicy(p Policy) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	_, err := a.enforcer.AddPolicy(policyArgs(p)...)
	if err != nil {
		return errorfamily.WrapTransient(err, "casbin_error", "add policy")
	}
	return nil
}

// RemovePolicy removes a single RBAC policy rule.
func (a *Authz) RemovePolicy(p Policy) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	_, err := a.enforcer.RemovePolicy(policyArgs(p)...)
	if err != nil {
		return errorfamily.WrapTransient(err, "casbin_error", "remove policy")
	}
	return nil
}

// AddGroupPolicy assigns a role to a subject in a domain.
func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	_, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	if err != nil {
		return wrapGroupError(err, "add", g)
	}
	return nil
}

// RemoveGroupPolicy removes a role assignment from a subject in a domain.
func (a *Authz) RemoveGroupPolicy(g GroupPolicy) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	_, err := a.enforcer.RemoveGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	if err != nil {
		return wrapGroupError(err, "remove", g)
	}
	return nil
}

// RemoveAllRolesForUser removes all group policies (role assignments) for the
// given subject across all domains. Used when a user is deleted.
//
// The subject parameter is a raw string, not UserID, because Casbin subjects
// are polymorphic: they can be user IDs, bot IDs, or prefixed actor IDs
// ("user:01JX...", "bot:01JX..."). Typed wrappers would limit this method to
// one actor kind. The string form is whatever Casbin stored as the subject
// when the group policy was added.
func (a *Authz) RemoveAllRolesForUser(subject string) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	domains, err := a.enforcer.GetDomainsForUser(subject)
	if err != nil {
		return errorfamily.NewTransient("casbin_error",
			"get domains for user "+subject).WithCause(err)
	}
	for _, domain := range domains {
		roles, err := a.enforcer.GetRolesForUser(subject, domain)
		if err != nil {
			return wrapCasbinError(err, "get roles for %s in domain %s", subject, domain)
		}
		for _, role := range roles {
			if _, err := a.enforcer.RemoveGroupingPolicy(subject, role, domain); err != nil {
				return wrapCasbinError(err, "remove group {%s, %s, %s}", subject, role, domain)
			}
		}
	}
	return nil
}

// RemoveAllRolesInDomain removes all group policies (role assignments) for the
// given subject within a specific domain. Used when a member's roles change
// or a member is removed from a tenant.
//
// The subject parameter is a raw string for the same reason as
// RemoveAllRolesForUser: Casbin subjects are polymorphic (users, bots,
// prefixed actors). The domain is typed as TenantID because all domains in
// this system are tenant-scoped.
func (a *Authz) RemoveAllRolesInDomain(subject string, domain TenantID) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}
	d := domain.Get()
	roles, err := a.enforcer.GetRolesForUser(subject, d)
	if err != nil {
		return wrapCasbinError(err, "get roles for %s in domain %s", subject, d)
	}
	for _, role := range roles {
		if _, err := a.enforcer.RemoveGroupingPolicy(subject, role, d); err != nil {
			return wrapCasbinError(err, "remove group {%s, %s, %s}", subject, role, d)
		}
	}
	return nil
}

// Policies returns all stored policy rules.
func (a *Authz) Policies() ([][]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	p, err := a.enforcer.GetPolicy()
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "get policies")
	}
	return p, nil
}

// GroupPolicies returns all stored group (role) policies.
func (a *Authz) GroupPolicies() ([][]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	g, err := a.enforcer.GetGroupingPolicy()
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "get group policies")
	}
	return g, nil
}

func convertRoles(roles []string) []Role {
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}
	return result
}
