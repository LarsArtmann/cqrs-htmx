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
