package usermgmt

import (
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
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
	// RoleSuperAdmin has unrestricted global access across all tenants.
	// Inherits all other roles via the g2 role hierarchy.
	RoleSuperAdmin Role = "super_admin"
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
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (p.sub == "*" || g(r.sub, p.sub, r.dom) || g2(r.sub, p.sub)) && (p.dom == "*" || r.dom == p.dom) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)
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
		return nil, event.WrapTransient(err, "casbin_error", "parse casbin model")
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, event.WrapTransient(err, "casbin_error", "create enforcer")
	}

	for _, p := range config.Policies {
		if _, err := e.AddPolicy(policyArgs(p)...); err != nil {
			return nil, wrapPolicyError(err, "add", p)
		}
	}

	for _, g := range config.Groups {
		if _, err := e.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
			return nil, wrapGroupError(err, "add", g)
		}
	}

	for _, h := range defaultRoleHierarchy() {
		if _, err := e.AddNamedGroupingPolicy("g2", string(h.From), string(h.To)); err != nil {
			return nil, event.Wrapf(
				err, event.Transient, "casbin_error",
				"seed role hierarchy g2(%s, %s)", h.From, h.To,
			)
		}
	}

	return &Authz{enforcer: e}, nil
}

// Enforce checks whether the subject is allowed to perform the action on the
// object within the given domain.
func (a *Authz) Enforce(sub, dom, obj string, act Action) (bool, error) {
	if a.enforcer == nil {
		return false, ErrEnforcerNotInitialized
	}
	ok, err := a.enforcer.Enforce(sub, dom, obj, string(act))
	if err != nil {
		return false, event.Wrapf(
			err, event.Transient, "casbin_error",
			"enforce %s/%s/%s/%s", sub, dom, obj, act,
		)
	}
	return ok, nil
}

// EnforceAny passes arbitrary values directly to the Casbin enforcer.
func (a *Authz) EnforceAny(rvals ...any) (bool, error) {
	if a.enforcer == nil {
		return false, ErrEnforcerNotInitialized
	}
	ok, err := a.enforcer.Enforce(rvals...)
	if err != nil {
		return false, event.WrapTransient(err, "casbin_error", "enforce any")
	}
	return ok, nil
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
	if a.enforcer == nil {
		return nil, ErrEnforcerNotInitialized
	}
	allowed, matched, err := a.enforcer.EnforceEx(sub, dom, obj, string(act))
	if err != nil {
		return nil, event.Wrapf(
			err, event.Transient, "casbin_error",
			"sub=%s dom=%s obj=%s", sub, dom, obj,
		)
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
		return event.Wrapf(
			err, event.Transient, "casbin_error",
			"authorize %s/%s/%s/%s", sub, dom, obj, act,
		)
	}
	if !ok {
		return event.Wrapf(
			ErrForbidden, event.Rejection, "forbidden",
			"%s cannot %s %s in domain %s", sub, act, obj, dom,
		)
	}
	return nil
}

// policyArgs converts a Policy to the variadic args expected by casbin
// policy methods (AddPolicy, RemovePolicy, etc.).
func policyArgs(p Policy) []any {
	return []any{string(p.Subject), p.Domain, p.Object, string(p.Action), string(p.Effect)}
}
