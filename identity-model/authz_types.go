package identitymodel

// Action represents an authorization action verb (e.g. "read", "execute").
type Action string

const (
	ActionExecute Action = "execute"
	ActionRead    Action = "read"
	ActionAll     Action = "*"
)

// Valid reports whether a is one of the declared Action constants.
func (a Action) Valid() bool {
	switch a {
	case ActionExecute, ActionRead, ActionAll:
		return true
	}

	return false
}

// Effect represents the outcome of a policy rule: allow or deny.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Valid reports whether e is one of the declared Effect constants.
func (e Effect) Valid() bool {
	switch e {
	case EffectAllow, EffectDeny:
		return true
	}

	return false
}

// Role is a named role used in RBAC group policies.
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleUser       Role = "user"
	RoleViewer     Role = "viewer"
	RoleOwner      Role = "owner"
)

// Valid reports whether r is one of the declared Role constants.
func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleUser, RoleViewer, RoleOwner:
		return true
	}

	return false
}

// AssignableRoles returns the roles that can be granted to a member within a
// tenant — i.e. every role except super_admin, which is global-only.
func AssignableRoles() []Role {
	return []Role{RoleViewer, RoleUser, RoleAdmin, RoleOwner}
}

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
	ModelString string
	Policies    []Policy
	Groups      []GroupPolicy
}
