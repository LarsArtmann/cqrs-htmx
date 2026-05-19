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

const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleViewer = "viewer"
	RoleOwner  = "owner"
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
	Subject string
	Domain  string
	Object  string
	Action  Action
	Effect  Effect
}

type GroupPolicy struct {
	User   string
	Role   string
	Domain string
}

type EnforceResult struct {
	Allowed     bool
	MatchedRule []string
	Subject     string
	Domain      string
	Object      string
	Action      Action
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
		if _, err := e.AddPolicy(p.Subject, p.Domain, p.Object, string(p.Action), string(p.Effect)); err != nil {
			return nil, fmt.Errorf("add policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}

	for _, g := range config.Groups {
		if _, err := e.AddGroupingPolicy(g.User, g.Role, g.Domain); err != nil {
			return nil, fmt.Errorf("add group {%s, %s, %s}: %w", g.User, g.Role, g.Domain, err)
		}
	}

	return &Authz{enforcer: e}, nil
}

func (a *Authz) Enforce(sub, dom, obj string, act Action) (bool, error) {
	return a.enforcer.Enforce(sub, dom, obj, string(act))
}

func (a *Authz) EnforceEx(sub, dom, obj string, act Action) (*EnforceResult, error) {
	allowed, matched, err := a.enforcer.EnforceEx(sub, dom, obj, string(act))
	if err != nil {
		return nil, err
	}
	return &EnforceResult{
		Allowed:     allowed,
		MatchedRule: matched,
		Subject:     sub,
		Domain:      dom,
		Object:      obj,
		Action:      act,
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

func (a *Authz) Apply(update PolicyUpdate) error {
	for _, g := range update.RemoveGroups {
		if _, err := a.enforcer.RemoveGroupingPolicy(g.User, g.Role, g.Domain); err != nil {
			return fmt.Errorf("remove group {%s, %s, %s}: %w", g.User, g.Role, g.Domain, err)
		}
	}
	for _, p := range update.RemovePolicies {
		if _, err := a.enforcer.RemovePolicy(p.Subject, p.Domain, p.Object, string(p.Action), string(p.Effect)); err != nil {
			return fmt.Errorf("remove policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}
	for _, g := range update.AddGroups {
		if _, err := a.enforcer.AddGroupingPolicy(g.User, g.Role, g.Domain); err != nil {
			return fmt.Errorf("add group {%s, %s, %s}: %w", g.User, g.Role, g.Domain, err)
		}
	}
	for _, p := range update.AddPolicies {
		if _, err := a.enforcer.AddPolicy(p.Subject, p.Domain, p.Object, string(p.Action), string(p.Effect)); err != nil {
			return fmt.Errorf("add policy {%s, %s, %s, %s, %s}: %w",
				p.Subject, p.Domain, p.Object, p.Action, p.Effect, err)
		}
	}
	return nil
}

func (a *Authz) AddPolicy(p Policy) error {
	_, err := a.enforcer.AddPolicy(p.Subject, p.Domain, p.Object, string(p.Action), string(p.Effect))
	return err
}

func (a *Authz) RemovePolicy(p Policy) error {
	_, err := a.enforcer.RemovePolicy(p.Subject, p.Domain, p.Object, string(p.Action), string(p.Effect))
	return err
}

func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.AddGroupingPolicy(g.User, g.Role, g.Domain)
	return err
}

func (a *Authz) RemoveGroupPolicy(g GroupPolicy) error {
	_, err := a.enforcer.RemoveGroupingPolicy(g.User, g.Role, g.Domain)
	return err
}

func (a *Authz) Policies() ([][]string, error) {
	return a.enforcer.GetPolicy()
}

func (a *Authz) GroupPolicies() ([][]string, error) {
	return a.enforcer.GetGroupingPolicy()
}

func (a *Authz) RolesForUser(userID, domain string) ([]string, error) {
	return a.enforcer.GetRolesForUser(userID, domain)
}

func (a *Authz) ImplicitRolesForUser(userID, domain string) ([]string, error) {
	return a.enforcer.GetImplicitRolesForUser(userID, domain)
}

func (a *Authz) ImplicitPermissionsForUser(userID, domain string) ([][]string, error) {
	return a.enforcer.GetImplicitPermissionsForUser(userID, domain)
}

func (a *Authz) DomainsForUser(userID string) ([]string, error) {
	return a.enforcer.GetDomainsForUser(userID)
}

func (a *Authz) UsersForRole(role, domain string) ([]string, error) {
	return a.enforcer.GetUsersForRole(role, domain)
}

func (a *Authz) RawEnforcer() *casbin.Enforcer {
	return a.enforcer
}

func defaultPolicies() []Policy {
	return []Policy{
		{RoleAdmin, "*", "*", ActionAll, EffectAllow},
	}
}
