package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type (
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	AddMemberCmd = identitymodel.AddMemberCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	UpdateMemberRolesCmd = identitymodel.UpdateMemberRolesCmd
	// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
	RemoveMemberCmd = identitymodel.RemoveMemberCmd
)

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return identitymodel.NewAddMemberCmd(actorID, tenantID, roles)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewUpdateMemberRolesCmd(actorID ActorID, tenantID TenantID, roles []Role) *UpdateMemberRolesCmd {
	return identitymodel.NewUpdateMemberRolesCmd(actorID, tenantID, roles)
}

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return identitymodel.NewRemoveMemberCmd(actorID, tenantID)
}

func deriveMembershipID(actorID ActorID, tenantID TenantID) id.StreamID {
	return identitymodel.DeriveMembershipID(actorID, tenantID)
}
