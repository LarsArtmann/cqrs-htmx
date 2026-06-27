package usermgmt

import (
	"context"
)

// AddMember grants the given roles to an actor within a tenant by dispatching
// an AddMember command. Re-adding the same actor replaces its role set. It is
// the write-side counterpart of [Service.TenantMembers].
func (s *Service) AddMember(ctx context.Context, actorID ActorID, tenantID TenantID, roles []Role) error {
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx, NewAddMemberCmd(actorID, tenantID, roles),
	)
}

// UpdateMemberRoles replaces the roles an actor holds in a tenant.
func (s *Service) UpdateMemberRoles(ctx context.Context, actorID ActorID, tenantID TenantID, roles []Role) error {
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx, NewUpdateMemberRolesCmd(actorID, tenantID, roles),
	)
}

// RemoveMember revokes an actor's membership in a tenant.
func (s *Service) RemoveMember(ctx context.Context, actorID ActorID, tenantID TenantID) error {
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx, NewRemoveMemberCmd(actorID, tenantID),
	)
}
