package usermgmt

import (
	"context"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// GrantRoles gives actorID exactly the given roles inside tenantID: it creates
// the membership when it does not exist yet and replaces the role set when it
// does, so calling it twice converges on the same state instead of failing.
//
// This is the durable grant path. It dispatches the Membership commands
// (AddMember, then UpdateMemberRoles when the membership already exists), so the
// grant is an event in the journal and [CasbinProjection] rebuilds it on every
// restart, projection rebuild, and second instance. Prefer it over
// [identitymodel.Authz.AddGroupPolicy], which writes into the in-memory enforcer
// only and is lost the moment the process stops.
//
// Pass identitymodel.WildcardDomain ("*") as tenantID for a global grant — the
// domain the admin panel checks for super_admin/admin. Roles are validated
// before dispatch, so a typo is rejected instead of silently producing an
// unusable policy row.
func (s *Service) GrantRoles(ctx context.Context, actorID ActorID, tenantID TenantID, roles []Role) error {
	if err := validateGrant(actorID, tenantID, roles); err != nil {
		return err
	}

	err := s.AddMember(ctx, actorID, tenantID, roles)
	if err == nil {
		return nil
	}

	if !isConflict(err) {
		return err
	}

	return s.UpdateMemberRoles(ctx, actorID, tenantID, roles)
}

// RevokeRoles removes actorID's membership in tenantID, so the actor holds no
// roles there at all. It reuses RemoveMember, which means the removal is an
// event too — the projection drops the actor's policy rows on replay.
func (s *Service) RevokeRoles(ctx context.Context, actorID ActorID, tenantID TenantID) error {
	if actorID.IsZero() {
		return errorfamily.NewRejection("usermgmt.grant.actor_required", "actor ID is required")
	}

	if tenantID.IsZero() {
		return errorfamily.NewRejection("usermgmt.grant.tenant_required", "tenant ID is required")
	}

	return s.RemoveMember(ctx, actorID, tenantID)
}

// HasAdminGrant reports whether any subject holds one of the admin-tier roles
// (super_admin or admin) in domain. Use identitymodel.WildcardDomain to ask
// "can anybody reach the admin panel at all?".
//
// It reads the policy state the service has materialized so far, so call it
// after the projections have drained: an in-flight replay can legitimately
// report false until the grants it is restoring have been applied.
func (s *Service) HasAdminGrant(domain TenantID) (bool, error) {
	groups, err := s.authz.GroupPolicies()
	if err != nil {
		return false, errorfamily.WrapInfrastructure(
			err,
			"usermgmt.admin_grant.enumerate",
			"enumerate casbin group policies",
		)
	}

	admins := make(map[Role]struct{}, len(identitymodel.AdminRoles()))
	for _, r := range identitymodel.AdminRoles() {
		admins[r] = struct{}{}
	}

	for _, row := range groups {
		if len(row) != 3 || row[2] != domain.Get() {
			continue
		}

		// The Casbin `g` section carries two different relations in this
		// codebase: real grant rows (subject, role, domain) and the mirrored
		// role-hierarchy edges (parentRole, childRole, domain) that
		// ensureRoleHierarchy writes so inheritance composes. A hierarchy
		// row's SUBJECT is a role name, never an actor ID, so the role
		// vocabulary is what tells them apart. Without this filter, seeding a
		// domain's hierarchy alone would report "an admin exists".
		if Role(row[0]).Valid() {
			continue
		}

		if _, ok := admins[Role(row[1])]; ok {
			return true, nil
		}
	}

	return false, nil
}

// validateGrant rejects a grant that could never take effect: a zero actor, a
// zero tenant, or a role string outside the declared vocabulary. Catching it
// here keeps invalid values out of the journal — the projection would happily
// write a policy row for a role no check ever asks about.
func validateGrant(actorID ActorID, tenantID TenantID, roles []Role) error {
	if actorID.IsZero() {
		return errorfamily.NewRejection("usermgmt.grant.actor_required", "actor ID is required")
	}

	if tenantID.IsZero() {
		return errorfamily.NewRejection("usermgmt.grant.tenant_required", "tenant ID is required")
	}

	for _, r := range roles {
		if r.Valid() {
			continue
		}

		return errorfamily.NewRejection(
			"usermgmt.grant.unknown_role",
			"unknown role: "+string(r),
		).WithContext("valid_roles", "super_admin, admin, owner, user, viewer")
	}

	return nil
}
