package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	aggregateTypeUser event.AggregateType = "User"

	eventUserRegistered     event.Type = "UserRegistered"
	eventRolesUpdated       event.Type = "RolesUpdated"
	eventEmailChanged       event.Type = "EmailChanged"
	eventDisplayNameChanged event.Type = "DisplayNameChanged"
	eventUserDeleted        event.Type = "UserDeleted"
	eventCredentialAdded    event.Type = "CredentialAdded"   //nolint:gosec // event type name, not credential
	eventCredentialRemoved  event.Type = "CredentialRemoved" //nolint:gosec // event type name, not credential

	cmdRegisterUser      command.Type = "RegisterUser"
	cmdUpdateRoles       command.Type = "UpdateRoles"
	cmdChangeEmail       command.Type = "ChangeEmail"
	cmdChangeDisplayName command.Type = "ChangeDisplayName"
	cmdDeleteUser        command.Type = "DeleteUser"
	cmdAddCredential     command.Type = "AddCredential"    //nolint:gosec // command type name, not credential
	cmdRemoveCredential  command.Type = "RemoveCredential" //nolint:gosec // command type name, not credential
)

var allUserEventTypes = []event.Type{
	eventUserRegistered,
	eventRolesUpdated,
	eventEmailChanged,
	eventDisplayNameChanged,
	eventUserDeleted,
	eventCredentialAdded,
	eventCredentialRemoved,
}
