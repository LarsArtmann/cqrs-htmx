package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	aggregateTypeUser event.AggregateType = "User"

	eventUserRegistered     event.Type = "UserRegistered"
	eventPasswordChanged    event.Type = "PasswordChanged"
	eventRolesUpdated       event.Type = "RolesUpdated"
	eventEmailChanged       event.Type = "EmailChanged"
	eventDisplayNameChanged event.Type = "DisplayNameChanged"
	eventUserDeleted        event.Type = "UserDeleted"

	cmdRegisterUser      command.Type = "RegisterUser"
	cmdChangePassword    command.Type = "ChangePassword"
	cmdUpdateRoles       command.Type = "UpdateRoles"
	cmdChangeEmail       command.Type = "ChangeEmail"
	cmdChangeDisplayName command.Type = "ChangeDisplayName"
	cmdDeleteUser        command.Type = "DeleteUser"
)

var allUserEventTypes = []event.Type{
	eventUserRegistered,
	eventPasswordChanged,
	eventRolesUpdated,
	eventEmailChanged,
	eventDisplayNameChanged,
	eventUserDeleted,
}
