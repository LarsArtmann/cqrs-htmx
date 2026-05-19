package usermgmt

import (
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleViewer = "viewer"
)

const rbacModel = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
`

const defaultPolicy = `p, admin, *, *
p, user, users, read
p, user, users, update
p, user, profile, read
p, user, profile, update
p, viewer, users, read
p, viewer, profile, read
`

// DefaultPolicies returns the built-in role/permission policies.
// Each entry is [sub, obj, act].
func DefaultPolicies() [][]string {
	return [][]string{
		{RoleAdmin, "*", "*"},
		{RoleUser, "users", "read"},
		{RoleUser, "users", "update"},
		{RoleUser, "profile", "read"},
		{RoleUser, "profile", "update"},
		{RoleViewer, "users", "read"},
		{RoleViewer, "profile", "read"},
	}
}

// NewEnforcer creates a Casbin enforcer with the built-in RBAC model
// seeded with default policies. Uses an in-memory adapter.
func NewEnforcer() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return nil, err
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}

	if err := seedPolicies(e); err != nil {
		return nil, err
	}

	return e, nil
}

// NewEnforcerWithModel creates a Casbin enforcer with a custom RBAC model string.
// The model must follow Casbin's INI format.
func NewEnforcerWithModel(modelStr string) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelStr)
	if err != nil {
		return nil, err
	}

	return casbin.NewEnforcer(m)
}

func seedPolicies(e *casbin.Enforcer) error {
	policies := DefaultPolicies()
	for _, p := range policies {
		_, err := e.AddPolicy(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// AssignRole assigns a role to a user.
func AssignRole(e *casbin.Enforcer, userID, role string) error {
	_, err := e.AddGroupingPolicy(userID, role)
	return err
}

// RevokeRole removes a role from a user.
func RevokeRole(e *casbin.Enforcer, userID, role string) error {
	_, err := e.RemoveGroupingPolicy(userID, role)
	return err
}

// RolesForUser returns all roles assigned to a user.
func RolesForUser(e *casbin.Enforcer, userID string) ([]string, error) {
	return e.GetRolesForUser(userID)
}

// CheckPermission checks if a user has permission to perform an action.
func CheckPermission(e *casbin.Enforcer, userID, object, action string) (bool, error) {
	return e.Enforce(userID, object, action)
}

// AddPolicy adds a custom policy (sub, obj, act).
func AddPolicy(e *casbin.Enforcer, sub, obj, act string) error {
	_, err := e.AddPolicy(sub, obj, act)
	return err
}

// RemovePolicy removes a policy (sub, obj, act).
func RemovePolicy(e *casbin.Enforcer, sub, obj, act string) error {
	_, err := e.RemovePolicy(sub, obj, act)
	return err
}
