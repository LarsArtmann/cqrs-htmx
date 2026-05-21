package usermgmt

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/cockroachdb/errors"
)

// Action represents an authorization action verb (e.g. "read", "execute").
type Action string

const (
	// ActionExecute grants permission to perform a write or state-changing operation.
	ActionExecute Action = "execute"
	// ActionRead grants permission to read or view a resource.
	ActionRead Action = "read"
	// ActionAll is a wildcard that matches any action.
	ActionAll Action = "*"
)

// Effect represents the outcome of a policy rule: allow or deny.
type Effect string

const (
	// EffectAllow permits the action.
	EffectAllow Effect = "allow"
	// EffectDeny blocks the action, taking priority over any allow.
	EffectDeny Effect = "deny"
)

// Role is a named role used in RBAC group policies.
type Role string

const (
	// RoleAdmin has unrestricted access (default policy: wildcard allow).
	RoleAdmin Role = "admin"
	// RoleUser is the standard role assigned on registration.
	RoleUser Role = "user"
	// RoleViewer can only read resources.
	RoleViewer Role = "viewer"
	// RoleOwner grants ownership-level permissions within a domain.
	RoleOwner Role = "owner"
)

const defaultModel = `[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act, eft

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (p.sub == "*" || g(r.sub, p.sub, r.dom)) && (p.dom == "*" || r.dom == p.dom) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)
`

// Policy defines a single RBAC rule: who can do what, with an allow/deny effect.
type Policy struct {
	Subject Role
	Domain  string
	Object  string
	Action  Action
	Effect  Effect
}

// GroupPolicy assigns a Role to a Subject (typically a user ID) within a Domain.
type GroupPolicy struct {
	Subject string
	Role    Role
	Domain  string
}

// EnforceResult contains the outcome of an EnforceEx call, including matched rules.
type EnforceResult struct {
	Allowed      bool
	MatchedRules []string
	Subject      string
	Domain       string
	Object       string
	Action       Action
}

// PolicyUpdate is a batch of policy and group changes to apply atomically.
type PolicyUpdate struct {
	AddGroups      []GroupPolicy
	RemoveGroups   []GroupPolicy
	AddPolicies    []Policy
	RemovePolicies []Policy
}

// EnforcerConfig controls the initial model, policies, and groups for NewAuthz.
type EnforcerConfig struct {
	// ModelString is the Casbin PERM model. Defaults to the built-in RBAC-with-domains model.
	ModelString string
	// Policies are seed policies applied on creation.
	Policies []Policy
	// Groups are seed group policies applied on creation.
	Groups []GroupPolicy
}

// Authz wraps a Casbin enforcer and provides domain-aware RBAC authorization.
type Authz struct {
	enforcer *casbin.Enforcer
}

// NewAuthz creates an Authz with the given optional config. When no config is
// provided, the default model and a single admin wildcard policy are used.
func NewAuthz(cfg ...EnforcerConfig) (*Authz, error) {
	config := EnforcerConfig{
		ModelString: defaultModel,
		Policies:    defaultPolicies(),
	}
	if len(cfg) > 0 {
		config = cfg[0]
	}

	modelStr := config.ModelString
	if modelStr == "" {
		modelStr = defaultModel
	}

	m, err := model.NewModelFromString(modelStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse casbin model")
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, errors.Wrapf(err, "create enforcer")
	}

	for _, p := range config.Policies {
		if _, err := e.AddPolicy(policyArgs(p)...); err != nil {
			return nil, errors.Wrapf(err, "add policy {%s, %s, %s, %s, %s}",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect)
		}
	}

	for _, g := range config.Groups {
		if _, err := e.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
			return nil, errors.Wrapf(err, "add group {%s, %s, %s}", g.Subject, g.Role, g.Domain)
		}
	}

	return &Authz{enforcer: e}, nil
}

// Enforce checks whether the subject is allowed to perform the action on the
// object within the given domain.
func (a *Authz) Enforce(sub, dom, obj string, act Action) (bool, error) {
	return a.enforcer.Enforce(sub, dom, obj, string(act))
}

// EnforceAny passes arbitrary values directly to the Casbin enforcer.
func (a *Authz) EnforceAny(rvals ...any) (bool, error) {
	return a.enforcer.Enforce(rvals...)
}

type enforcerAdapter struct {
	authz *Authz
}

func (e *enforcerAdapter) Enforce(rvals ...any) (bool, error) {
	return e.authz.EnforceAny(rvals...)
}

// AsEnforcer returns a value that satisfies the cqrshtmx.Enforcer interface.
// Use this to bridge usermgmt authorization with cqrs-htmx handlers:
//
//	app := cqrshtmx.New(cqrshtmx.Config{Enforcer: authz.AsEnforcer()})
func (a *Authz) AsEnforcer() interface{ Enforce(...any) (bool, error) } {
	return &enforcerAdapter{authz: a}
}

// EnforceEx is like Enforce but also returns the matched policy rules.
func (a *Authz) EnforceEx(sub, dom, obj string, act Action) (*EnforceResult, error) {
	allowed, matched, err := a.enforcer.EnforceEx(sub, dom, obj, string(act))
	if err != nil {
		return nil, fmt.Errorf("sub=%s dom=%s obj=%s: %w", sub, dom, obj, err)
	}
	return &EnforceResult{
		Allowed:      allowed,
		MatchedRules: matched,
		Subject:      sub,
		Domain:       dom,
		Object:       obj,
		Action:       act,
	}, nil
}

// Authorize is like Enforce but returns ErrForbidden with context on denial.
func (a *Authz) Authorize(sub, dom, obj string, act Action) error {
	ok, err := a.Enforce(sub, dom, obj, act)
	if err != nil {
		return errors.Wrapf(err, "authorize %s/%s/%s/%s", sub, dom, obj, act)
	}
	if !ok {
		return errors.WithMessagef(ErrForbidden, "%s cannot %s %s in domain %s", sub, act, obj, dom)
	}
	return nil
}

// policyArgs converts a Policy to the variadic args expected by casbin
// policy methods (AddPolicy, RemovePolicy, etc.).
func policyArgs(p Policy) []any {
	return []any{string(p.Subject), p.Domain, p.Object, string(p.Action), string(p.Effect)}
}

func policyWrapErr(msg string, p Policy) string {
	return fmt.Sprintf(
		"%s {%s, %s, %s, %s, %s}",
		msg,
		p.Subject,
		p.Domain,
		p.Object,
		p.Action,
		p.Effect,
	)
}

// Apply atomically applies a batch of group and policy additions/removals.
func (a *Authz) Apply(update PolicyUpdate) error {
	for _, g := range update.RemoveGroups {
		if _, err := a.enforcer.RemoveGroupingPolicy(
			g.Subject,
			string(g.Role),
			g.Domain,
		); err != nil {
			return errors.Wrapf(err, "remove group {%s, %s, %s}", g.Subject, g.Role, g.Domain)
		}
	}
	for _, p := range update.RemovePolicies {
		if _, err := a.enforcer.RemovePolicy(policyArgs(p)...); err != nil {
			return errors.Wrapf(err, policyWrapErr("remove policy", p))
		}
	}
	for _, g := range update.AddGroups {
		if _, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
			return errors.Wrapf(err, "add group {%s, %s, %s}", g.Subject, g.Role, g.Domain)
		}
	}
	for _, p := range update.AddPolicies {
		if _, err := a.enforcer.AddPolicy(policyArgs(p)...); err != nil {
			return errors.Wrapf(err, policyWrapErr("add policy", p))
		}
	}
	return nil
}

// AddPolicy adds a single RBAC policy rule.
func (a *Authz) AddPolicy(p Policy) error {
	_, err := a.enforcer.AddPolicy(policyArgs(p)...)
	return err
}

// RemovePolicy removes a single RBAC policy rule.
func (a *Authz) RemovePolicy(p Policy) error {
	_, err := a.enforcer.RemovePolicy(policyArgs(p)...)
	return err
}

// AddGroupPolicy assigns a role to a subject in a domain.
func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	return err
}

// RemoveGroupPolicy removes a role assignment from a subject in a domain.
func (a *Authz) RemoveGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.RemoveGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	return err
}

// Policies returns all stored policy rules.
func (a *Authz) Policies() ([][]string, error) {
	return a.enforcer.GetPolicy()
}

// GroupPolicies returns all stored group (role) policies.
func (a *Authz) GroupPolicies() ([][]string, error) {
	return a.enforcer.GetGroupingPolicy()
}

func convertRoles(roles []string) []Role {
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}
	return result
}

// RolesForUser returns the directly assigned roles for a user in the given domain.
func (a *Authz) RolesForUser(userID UserID, domain string) ([]Role, error) {
	roles, err := a.enforcer.GetRolesForUser(userID.Get(), domain)
	if err != nil {
		return nil, fmt.Errorf("domain=%s: %w", domain, err)
	}
	return convertRoles(roles), nil
}

// ImplicitRolesForUser returns all roles inherited (transitively) by the user in the domain.
func (a *Authz) ImplicitRolesForUser(userID UserID, domain string) ([]Role, error) {
	roles, err := a.enforcer.GetImplicitRolesForUser(userID.Get(), domain)
	if err != nil {
		return nil, fmt.Errorf("domain=%s: %w", domain, err)
	}
	return convertRoles(roles), nil
}

// ImplicitPermissionsForUser returns all permissions the user has in the domain,
// including those inherited through role hierarchy.
func (a *Authz) ImplicitPermissionsForUser(userID UserID, domain string) ([][]string, error) {
	return a.enforcer.GetImplicitPermissionsForUser(userID.Get(), domain)
}

// DomainsForUser returns all domains the user has roles in.
func (a *Authz) DomainsForUser(userID UserID) ([]string, error) {
	return a.enforcer.GetDomainsForUser(userID.Get())
}

// UsersForRole returns all user IDs that have the given role in the domain.
func (a *Authz) UsersForRole(role Role, domain string) ([]string, error) {
	return a.enforcer.GetUsersForRole(string(role), domain)
}

func defaultPolicies() []Policy {
	return []Policy{
		{RoleAdmin, "*", "*", ActionAll, EffectAllow},
	}
}
