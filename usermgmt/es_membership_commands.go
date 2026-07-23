package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	AddMemberCmd         = identitymodel.AddMemberCmd
	UpdateMemberRolesCmd = identitymodel.UpdateMemberRolesCmd
	RemoveMemberCmd      = identitymodel.RemoveMemberCmd
)

func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return identitymodel.NewAddMemberCmd(actorID, tenantID, roles)
}

func NewUpdateMemberRolesCmd(actorID ActorID, tenantID TenantID, roles []Role) *UpdateMemberRolesCmd {
	return identitymodel.NewUpdateMemberRolesCmd(actorID, tenantID, roles)
}

func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return identitymodel.NewRemoveMemberCmd(actorID, tenantID)
}

func deriveMembershipID(actorID ActorID, tenantID TenantID) id.AggregateID {
	return identitymodel.DeriveMembershipID(actorID, tenantID)
}
