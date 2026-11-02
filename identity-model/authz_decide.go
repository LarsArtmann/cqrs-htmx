package identitymodel

// Check reports whether subject may perform action on resource within domain,
// together with the policy rows that decided it.
//
// It is this engine's implementation of the authorization seam the cqrs-htmx
// root package declares (PermissionChecker): the root defines the interface in
// primitives, this package satisfies it structurally, and neither imports the
// other. A compile-time assertion lives in usermgmt, the module that can see
// both.
//
// An unknown or unused action is not an error: it simply matches no policy, and
// the answer is a denial with no matched rules — the fail-closed outcome.
func (a *Authz) Check(subject, domain, resource, action string) (bool, []string, error) {
	res, err := a.EnforceEx(subject, domain, resource, Action(action))
	if err != nil {
		return false, nil, err
	}

	return res.Allowed, res.MatchedRules, nil
}
