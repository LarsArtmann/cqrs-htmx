package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type RegisterUserCmd struct {
	aggregateID  id.AggregateID
	email        string
	displayName  string
	passwordHash string
	roles        []Role
}

func NewRegisterUserCmd(
	aggID id.AggregateID, email, displayName, passwordHash string, roles []Role,
) *RegisterUserCmd {
	return &RegisterUserCmd{
		aggregateID:  aggID,
		email:        email,
		displayName:  displayName,
		passwordHash: passwordHash,
		roles:        roles,
	}
}

func (c *RegisterUserCmd) Type() command.Type          { return cmdRegisterUser }
func (c *RegisterUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangePasswordCmd struct {
	aggregateID  id.AggregateID
	passwordHash string
}

func NewChangePasswordCmd(aggID id.AggregateID, passwordHash string) *ChangePasswordCmd {
	return &ChangePasswordCmd{aggregateID: aggID, passwordHash: passwordHash}
}

func (c *ChangePasswordCmd) Type() command.Type          { return cmdChangePassword }
func (c *ChangePasswordCmd) AggregateID() id.AggregateID { return c.aggregateID }

type UpdateRolesCmd struct {
	aggregateID id.AggregateID
	roles       []Role
	domain      string
}

func NewUpdateRolesCmd(aggID id.AggregateID, roles []Role, domain string) *UpdateRolesCmd {
	return &UpdateRolesCmd{aggregateID: aggID, roles: roles, domain: domain}
}

func (c *UpdateRolesCmd) Type() command.Type          { return cmdUpdateRoles }
func (c *UpdateRolesCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeEmailCmd struct {
	aggregateID id.AggregateID
	email       string
}

func NewChangeEmailCmd(aggID id.AggregateID, email string) *ChangeEmailCmd {
	return &ChangeEmailCmd{aggregateID: aggID, email: email}
}

func (c *ChangeEmailCmd) Type() command.Type          { return cmdChangeEmail }
func (c *ChangeEmailCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeDisplayNameCmd struct {
	aggregateID id.AggregateID
	displayName string
}

func NewChangeDisplayNameCmd(aggID id.AggregateID, displayName string) *ChangeDisplayNameCmd {
	return &ChangeDisplayNameCmd{aggregateID: aggID, displayName: displayName}
}

func (c *ChangeDisplayNameCmd) Type() command.Type          { return cmdChangeDisplayName }
func (c *ChangeDisplayNameCmd) AggregateID() id.AggregateID { return c.aggregateID }

type DeleteUserCmd struct {
	aggregateID id.AggregateID
	reason      string
}

func NewDeleteUserCmd(aggID id.AggregateID, reason string) *DeleteUserCmd {
	return &DeleteUserCmd{aggregateID: aggID, reason: reason}
}

func (c *DeleteUserCmd) Type() command.Type          { return cmdDeleteUser }
func (c *DeleteUserCmd) AggregateID() id.AggregateID { return c.aggregateID }
