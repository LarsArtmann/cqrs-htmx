package identitymodel

// DefaultRBACModel is the Casbin PERM model for RBAC-with-domains authorization.
// It defines request and policy structures, role definitions (g for domain-scoped,
// g2 for global hierarchy), and matchers that check subject/domain/object/action.
const DefaultRBACModel = `[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act, eft

[role_definition]
g = _, _, _
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (p.sub == "*" || g(r.sub, p.sub, r.dom) || g2(r.sub, p.sub)) && (p.dom == "*" || r.dom == p.dom) && (p.obj == "*" || r.obj == p.obj) && (p.act == "*" || r.act == p.act)`

// DefaultPolicies returns the seed policies applied on enforcer creation.
func DefaultPolicies() []Policy {
	return []Policy{
		{RoleAdmin, "*", "*", ActionAll, EffectAllow},
		{RoleSuperAdmin, "*", "*", ActionAll, EffectAllow},
	}
}

// RoleHierarchyEntry represents a single role inheritance rule.
type RoleHierarchyEntry struct {
	From Role
	To   Role
}

// DefaultRoleHierarchy returns the g2 role inheritance policies that enable
// the hierarchy: super_admin > admin > user > viewer.
func DefaultRoleHierarchy() []RoleHierarchyEntry {
	return []RoleHierarchyEntry{
		{RoleSuperAdmin, RoleAdmin},
		{RoleAdmin, RoleUser},
		{RoleUser, RoleViewer},
	}
}
