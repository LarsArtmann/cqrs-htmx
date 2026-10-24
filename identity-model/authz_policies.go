package identitymodel

import (
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// policyMutator is the slice of *casbin.Enforcer that Apply needs. It exists so
// the rollback path can be exercised with a stub that fails on demand; a
// failing policy store is otherwise impossible to provoke in a unit test.
type policyMutator interface {
	AddGroupingPolicy(params ...any) (bool, error)
	RemoveGroupingPolicy(params ...any) (bool, error)
	AddPolicy(params ...any) (bool, error)
	RemovePolicy(params ...any) (bool, error)
}

type undoKind int

const (
	undoRemoveGroup undoKind = iota
	undoAddGroup
	undoRemovePolicy
	undoAddPolicy
)

type undoOp struct {
	kind undoKind
	args []any
}

func (u undoOp) apply(m policyMutator) error {
	var err error
	switch u.kind {
	case undoRemoveGroup:
		_, err = m.RemoveGroupingPolicy(u.args...)
	case undoAddGroup:
		_, err = m.AddGroupingPolicy(u.args...)
	case undoRemovePolicy:
		_, err = m.RemovePolicy(u.args...)
	case undoAddPolicy:
		_, err = m.AddPolicy(u.args...)
	}
	if err != nil {
		return errorfamily.WrapTransient(err, "casbin_error", "undo policy mutation")
	}

	return nil
}

// Apply applies a batch of group and policy changes atomically: if one
// operation fails, every already-applied operation is rolled back, so a caller
// never observes half a batch. Rollback is best-effort — a store that fails on
// the way in can fail on the way out too, and that is reported on the returned
// error rather than hidden.
//
// The role hierarchy is seeded for every domain the batch grants into, so an
// inheritance edge exists before the grant that relies on it.
func (a *Authz) Apply(update PolicyUpdate) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}

	for _, g := range update.AddGroups {
		if err := a.ensureRoleHierarchy(g.Domain); err != nil {
			return err
		}
	}

	return applyPolicyUpdate(a.enforcer, update)
}

func applyPolicyUpdate(m policyMutator, update PolicyUpdate) error {
	var applied []undoOp

	for _, g := range update.AddGroups {
		if err := applyAddGroup(m, &applied, g); err != nil {
			return rollback(m, applied, err)
		}
	}
	for _, p := range update.AddPolicies {
		if err := applyAddPolicy(m, &applied, p); err != nil {
			return rollback(m, applied, err)
		}
	}
	for _, g := range update.RemoveGroups {
		if err := applyRemoveGroup(m, &applied, g); err != nil {
			return rollback(m, applied, err)
		}
	}
	for _, p := range update.RemovePolicies {
		if err := applyRemovePolicy(m, &applied, p); err != nil {
			return rollback(m, applied, err)
		}
	}

	return nil
}

func applyAddGroup(m policyMutator, applied *[]undoOp, g GroupPolicy) error {
	args := []any{g.Subject, string(g.Role), g.Domain}
	changed, err := m.AddGroupingPolicy(args...)
	if err != nil {
		return wrapGroupError(err, "add", g)
	}
	if changed {
		*applied = append(*applied, undoOp{kind: undoRemoveGroup, args: args})
	}

	return nil
}

func applyAddPolicy(m policyMutator, applied *[]undoOp, p Policy) error {
	args := policyArgs(p)
	changed, err := m.AddPolicy(args...)
	if err != nil {
		return wrapPolicyError(err, "add", p)
	}
	if changed {
		*applied = append(*applied, undoOp{kind: undoRemovePolicy, args: args})
	}

	return nil
}

func applyRemoveGroup(m policyMutator, applied *[]undoOp, g GroupPolicy) error {
	args := []any{g.Subject, string(g.Role), g.Domain}
	changed, err := m.RemoveGroupingPolicy(args...)
	if err != nil {
		return wrapGroupError(err, "remove", g)
	}
	if changed {
		*applied = append(*applied, undoOp{kind: undoAddGroup, args: args})
	}

	return nil
}

func applyRemovePolicy(m policyMutator, applied *[]undoOp, p Policy) error {
	args := policyArgs(p)
	changed, err := m.RemovePolicy(args...)
	if err != nil {
		return wrapPolicyError(err, "remove", p)
	}
	if changed {
		*applied = append(*applied, undoOp{kind: undoAddPolicy, args: args})
	}

	return nil
}

// rollback reverses applied operations in reverse order and returns cause,
// annotated when the rollback itself could not complete.
func rollback(m policyMutator, applied []undoOp, cause error) error {
	var failed []string
	for i := len(applied) - 1; i >= 0; i-- {
		if err := applied[i].apply(m); err != nil {
			failed = append(failed, err.Error())
		}
	}

	if len(failed) > 0 {
		return errorfamily.WrapTransient(cause, "casbin_error",
			"apply rolled back incompletely: "+strings.Join(failed, "; "))
	}

	return cause
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

// AddGroupPolicy assigns a role to a subject in a domain, seeding the role
// hierarchy for that domain first so inherited permissions resolve.
func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	if a.enforcer == nil {
		return ErrEnforcerNotInitialized
	}

	if err := a.ensureRoleHierarchy(g.Domain); err != nil {
		return err
	}

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
