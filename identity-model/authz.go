package identitymodel

import (
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Authz wraps a Casbin enforcer and provides domain-aware RBAC authorization.
// It is the single authorization engine for the identity model.
type Authz struct {
	enforcer *casbin.Enforcer
}

// NewAuthz creates an Authz with the given optional config. When no config is
// provided, the default model and a single admin wildcard policy are used.
func NewAuthz(config ...EnforcerConfig) (*Authz, error) {
	config := EnforcerConfig{
		ModelString: DefaultRBACModel,
		Policies:    DefaultPolicies(),
	}
	if len(config) > 0 {
		config = config[0]
	}

	modelStr := config.ModelString
	if modelStr == "" {
		modelStr = DefaultRBACModel
	}

	m, err := model.NewModelFromString(modelStr)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "parse casbin model")
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "casbin_error", "create enforcer")
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

	for _, h := range DefaultRoleHierarchy() {
		if _, err := e.AddNamedGroupingPolicy("g2", string(h.From), string(h.To)); err != nil {
			return nil, errorfamily.Wrapf(
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
		return false, errorfamily.Wrapf(
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
		return false, errorfamily.WrapTransient(err, "casbin_error", "enforce any")
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
		return nil, errorfamily.Wrapf(
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
		return errorfamily.Wrapf(
			err, event.Transient, "casbin_error",
			"authorize %s/%s/%s/%s", sub, dom, obj, act,
		)
	}

	if !ok {
		return errorfamily.Wrapf(
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

// wrapCasbinError wraps an underlying casbin error with operation context.
func wrapCasbinError(err error, op string, args ...any) error {
	return errorfamily.Wrapf(err, event.Transient, "casbin_error", op, args...)
}

// wrapPolicyError wraps an error from a Policy operation.
func wrapPolicyError(err error, op string, p Policy) error {
	return wrapCasbinError(err, op+" policy {%s, %s, %s, %s, %s}",
		p.Subject, p.Domain, p.Object, p.Action, p.Effect)
}

// wrapGroupError wraps an error from a GroupPolicy operation.
func wrapGroupError(err error, op string, g GroupPolicy) error {
	return wrapCasbinError(err, op+" group {%s, %s, %s}",
		g.Subject, string(g.Role), g.Domain)
}

// convertRoles converts a slice of strings to typed Roles.
func convertRoles(roles []string) []Role {
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}

	return result
}
