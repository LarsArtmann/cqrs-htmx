package usermgmt

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

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
			return event.NewTransient("casbin_error", fmt.Sprintf("add group {%s, %s, %s}", g.Subject, g.Role, g.Domain)).
				WithCause(err)
		}
	}
	for _, p := range update.AddPolicies {
		if _, err := a.enforcer.AddPolicy(policyArgs(p)...); err != nil {
			return event.NewTransient(
				"casbin_error",
				policyWrapErr("add policy", p),
			).WithCause(err)
		}
	}
	for _, g := range update.RemoveGroups {
		if _, err := a.enforcer.RemoveGroupingPolicy(
			g.Subject,
			string(g.Role),
			g.Domain,
		); err != nil {
			return event.NewTransient("casbin_error", fmt.Sprintf("remove group {%s, %s, %s}", g.Subject, g.Role, g.Domain)).
				WithCause(err)
		}
	}
	for _, p := range update.RemovePolicies {
		if _, err := a.enforcer.RemovePolicy(policyArgs(p)...); err != nil {
			return event.NewTransient(
				"casbin_error",
				policyWrapErr("remove policy", p),
			).WithCause(err)
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
		return event.NewTransient("casbin_error", "add policy").WithCause(err)
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
		return event.NewTransient("casbin_error", "remove policy").WithCause(err)
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
		return event.NewTransient("casbin_error", fmt.Sprintf("add group %s/%s/%s", g.Subject, g.Role, g.Domain)).
			WithCause(err)
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
		return event.NewTransient("casbin_error", fmt.Sprintf("remove group %s/%s/%s", g.Subject, g.Role, g.Domain)).
			WithCause(err)
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
		return nil, event.NewTransient("casbin_error", "get policies").WithCause(err)
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
		return nil, event.NewTransient("casbin_error", "get group policies").WithCause(err)
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
