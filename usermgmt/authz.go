package usermgmt

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

type Action string

const (
	ActionExecute Action = "execute"
	ActionRead    Action = "read"
	ActionAll     Action = "*"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
	RoleOwner  Role = "owner"
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

type Policy struct {
	Subject Role
	Domain  string
	Object  string
	Action  Action
	Effect  Effect
}

type GroupPolicy struct {
	Subject string
	Role    Role
	Domain  string
}

type EnforceResult struct {
	Allowed      bool
	MatchedRules []string
	Subject      string
	Domain       string
	Object       string
	Action       Action
}

type PolicyUpdate struct {
	AddGroups      []GroupPolicy
	RemoveGroups   []GroupPolicy
	AddPolicies    []Policy
	RemovePolicies []Policy
}

type EnforcerConfig struct {
	ModelString string
	Policies    []Policy
	Groups      []GroupPolicy
}

type Authz struct {
	enforcer *casbin.Enforcer
}

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
		return nil, fmt.Errorf("parse casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("create enforcer: %w", err)
	}

	for _, p := range config.Policies {
		if _, err := e.AddPolicy(policyArgs(p)...); err != nil {
			return nil, fmt.Errorf("add policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}

	for _, g := range config.Groups {
		if _, err := e.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
			return nil, fmt.Errorf("add group {%s, %s, %s}: %w", g.Subject, g.Role, g.Domain, err)
		}
	}

	return &Authz{enforcer: e}, nil
}

func (a *Authz) Enforce(sub, dom, obj string, act Action) (bool, error) {
	return a.enforcer.Enforce(sub, dom, obj, string(act))
}

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

func (a *Authz) EnforceEx(sub, dom, obj string, act Action) (*EnforceResult, error) {
	allowed, matched, err := a.enforcer.EnforceEx(sub, dom, obj, string(act))
	if err != nil {
		return nil, err
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

func (a *Authz) Authorize(sub, dom, obj string, act Action) error {
	ok, err := a.Enforce(sub, dom, obj, act)
	if err != nil {
		return fmt.Errorf("authorize %s/%s/%s/%s: %w", sub, dom, obj, act, err)
	}
	if !ok {
		return fmt.Errorf("%w: %s cannot %s %s in domain %s", ErrForbidden, sub, act, obj, dom)
	}
	return nil
}

// policyArgs converts a Policy to the variadic args expected by casbin
// policy methods (AddPolicy, RemovePolicy, etc.).
func policyArgs(p Policy) []any {
	return []any{string(p.Subject), p.Domain, p.Object, string(p.Action), string(p.Effect)}
}

func (a *Authz) Apply(update PolicyUpdate) error {
	for _, g := range update.RemoveGroups {
		if _, err := a.enforcer.RemoveGroupingPolicy(
			g.Subject,
			string(g.Role),
			g.Domain,
		); err != nil {
		}
	}
	for _, p := range update.RemovePolicies {
		if _, err := a.enforcer.RemovePolicy(policyArgs(p)...); err != nil {
			return fmt.Errorf("remove policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}
	for _, g := range update.AddGroups {
		if _, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain); err != nil {
		}
	}
	for _, p := range update.AddPolicies {
		if _, err := a.enforcer.AddPolicy(policyArgs(p)...); err != nil {
			return fmt.Errorf("add policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}
	return nil
}

func (a *Authz) AddPolicy(p Policy) error {
	_, err := a.enforcer.AddPolicy(policyArgs(p)...)
	return err
}

func (a *Authz) RemovePolicy(p Policy) error {
	_, err := a.enforcer.RemovePolicy(policyArgs(p)...)
	return err
}

func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.AddGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	return err
}

func (a *Authz) RemoveGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.RemoveGroupingPolicy(g.Subject, string(g.Role), g.Domain)
	return err
}

func (a *Authz) Policies() ([][]string, error) {
	return a.enforcer.GetPolicy()
}

func (a *Authz) GroupPolicies() ([][]string, error) {
	return a.enforcer.GetGroupingPolicy()
}

func (a *Authz) RolesForUser(userID UserID, domain string) ([]Role, error) {
	roles, err := a.enforcer.GetRolesForUser(userID.String(), domain)
	if err != nil {
		return nil, err
	}
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}
	return result, nil
}

func (a *Authz) ImplicitRolesForUser(userID UserID, domain string) ([]Role, error) {
	roles, err := a.enforcer.GetImplicitRolesForUser(userID.String(), domain)
	if err != nil {
		return nil, err
	}
	result := make([]Role, len(roles))
	for i, r := range roles {
		result[i] = Role(r)
	}
	return result, nil
}

func (a *Authz) ImplicitPermissionsForUser(userID UserID, domain string) ([][]string, error) {
	return a.enforcer.GetImplicitPermissionsForUser(userID.String(), domain)
}

func (a *Authz) DomainsForUser(userID UserID) ([]string, error) {
	return a.enforcer.GetDomainsForUser(userID.String())
}

func (a *Authz) UsersForRole(role Role, domain string) ([]string, error) {
	return a.enforcer.GetUsersForRole(string(role), domain)
}

func defaultPolicies() []Policy {
	return []Policy{
		{RoleAdmin, "*", "*", ActionAll, EffectAllow},
	}
}
