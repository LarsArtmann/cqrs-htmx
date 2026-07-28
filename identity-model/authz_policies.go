package identitymodel

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Apply applies a batch of group and policy additions/removals sequentially.
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
			g.Subject, string(g.Role), g.Domain,
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
	return a.mutatePolicy(p, "add", a.enforcer.AddPolicy)
}

// RemovePolicy removes a single RBAC policy rule.
func (a *Authz) RemovePolicy(p Policy) error {
	return a.mutatePolicy(p, "remove", a.enforcer.RemovePolicy)
}

// mutatePolicy delegates to the given casbin policy method (AddPolicy or
// RemovePolicy) and wraps any error as a Transient failure with a
// human-readable action prefix.
func (a *Authz) mutatePolicy(p Policy, action string, fn func(...any) (bool, error)) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}

	if _, err := fn(policyArgs(p)...); err != nil {
		return errorfamily.WrapTransient(err, "casbin_error", action+" policy")
	}

	return nil
}

// AddGroupPolicy assigns a role to a subject in a domain.
func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	return a.mutateGroupPolicy(g, "add", a.enforcer.AddGroupingPolicy)
}

// RemoveGroupPolicy removes a role assignment from a subject in a domain.
func (a *Authz) RemoveGroupPolicy(g GroupPolicy) error {
	return a.mutateGroupPolicy(g, "remove", a.enforcer.RemoveGroupingPolicy)
}

// mutateGroupPolicy delegates to the given casbin grouping method
// (AddGroupingPolicy or RemoveGroupingPolicy) and wraps any error via
// wrapGroupError with the supplied action verb.
func (a *Authz) mutateGroupPolicy(g GroupPolicy, action string, fn func(...any) (bool, error)) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}

	if _, err := fn(g.Subject, string(g.Role), g.Domain); err != nil {
		return wrapGroupError(err, action, g)
	}

	return nil
}

// RemoveAllRolesForUser removes all group policies (role assignments) for the
// given subject across all domains.
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

// RemoveAllRolesInDomain removes all group policies for the given subject
// within a specific domain.
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
	return a.getPolicies(a.enforcer.GetPolicy, "get policies")
}

// GroupPolicies returns all stored group (role) policies.
func (a *Authz) GroupPolicies() ([][]string, error) {
	return a.getPolicies(a.enforcer.GetGroupingPolicy, "get group policies")
}

// getPolicies delegates to the given casbin getter (GetPolicy or
// GetGroupingPolicy) and wraps any error as a Transient failure.
func (a *Authz) getPolicies(getter func() ([][]string, error), errMsg string) ([][]string, error) {
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}

	p, err := getter()
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", errMsg)
	}

	return p, nil
}
