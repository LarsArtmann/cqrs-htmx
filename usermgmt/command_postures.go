package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// CommandPostures declares what every command this module dispatches requires of
// its caller, so a consumer can gate the dispatch boundary without re-deriving
// the classification from the service's method list.
//
// NewService owns the dispatcher, so installing the gate is the consumer's act:
//
//	gate, err := cqrshtmx.NewDispatchGate(svc.Authz())
//	if err != nil {
//	    return err
//	}
//
//	for _, posture := range usermgmt.CommandPostures() {
//	    if err := gate.Declare(posture); err != nil {
//	        return err
//	    }
//	}
//
//	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
//	    // FIRST, so the decision happens before any consumer retry middleware.
//	    // The audit chain above it is observation, which should see denials.
//	    CommandMiddleware: []command.Middleware{gate.CommandMiddleware()},
//	})
//
// The classification follows the caller, not the aggregate:
//
//   - public — reachable with no session at all. Registration is open by design
//     (closing it is a separate policy), and RecordCredentialUse runs inside the
//     sign-in ceremony, before a session exists. Gating either one on a session
//     would make passwordless login impossible.
//   - authenticated — a signed-in caller acting on their own account.
//   - role-gated — an administrative act. A consumer that wants self-service
//     account deletion grants `command/DeleteUser` to its user role; the safe
//     default is that only an administrator may.
func CommandPostures() []cqrshtmx.DispatchPosture {
	return []cqrshtmx.DispatchPosture{
		commandPosture(identitymodel.CmdRegisterUser, cqrshtmx.PosturePublic,
			"registration must be reachable by an unauthenticated visitor; closing it is a separate policy"),
		commandPosture(identitymodel.CmdRecordCredentialUse, cqrshtmx.PosturePublic,
			"runs inside the sign-in ceremony, before a session exists"),
		commandPosture(identitymodel.CmdChangeEmail, cqrshtmx.PostureAuthenticated,
			"a signed-in caller acting on their own account"),
		commandPosture(identitymodel.CmdChangeDisplayName, cqrshtmx.PostureAuthenticated,
			"a signed-in caller acting on their own account"),
		commandPosture(identitymodel.CmdAddCredential, cqrshtmx.PostureAuthenticated,
			"enrolling a credential requires an authenticated owner"),
		commandPosture(identitymodel.CmdRemoveCredential, cqrshtmx.PostureAuthenticated,
			"removing a credential requires an authenticated owner"),
		commandPosture(identitymodel.CmdVerifyEmail, cqrshtmx.PostureAuthenticated,
			"verification confirms an address on the caller's own account"),
		commandPosture(identitymodel.CmdEnableTOTP, cqrshtmx.PostureAuthenticated,
			"enabling a second factor requires an authenticated owner"),
		commandPosture(identitymodel.CmdDisableTOTP, cqrshtmx.PostureAuthenticated,
			"disabling a second factor requires an authenticated owner"),
		commandPosture(identitymodel.CmdLinkExternalAccount, cqrshtmx.PostureAuthenticated,
			"linking an identity provider account requires an authenticated owner"),
		commandPosture(identitymodel.CmdUnlinkExternalAccount, cqrshtmx.PostureAuthenticated,
			"unlinking an identity provider account requires an authenticated owner"),
		commandPosture(identitymodel.CmdDeleteUser, cqrshtmx.PostureRoleGated,
			"destructive and administrative; self-service deletion grants command/DeleteUser to the user role"),
		commandPosture(identitymodel.CmdAddMember, cqrshtmx.PostureRoleGated,
			"granting a role is an administrative act"),
		commandPosture(identitymodel.CmdUpdateMemberRoles, cqrshtmx.PostureRoleGated,
			"changing a role is an administrative act"),
		commandPosture(identitymodel.CmdRemoveMember, cqrshtmx.PostureRoleGated,
			"revoking a role is an administrative act"),
		commandPosture(identitymodel.CmdCreateTenant, cqrshtmx.PostureRoleGated,
			"tenants are created by an administrator"),
		commandPosture(identitymodel.CmdSuspendTenant, cqrshtmx.PostureRoleGated,
			"suspending a tenant is an administrative act"),
		commandPosture(identitymodel.CmdReactivateTenant, cqrshtmx.PostureRoleGated,
			"reactivating a tenant is an administrative act"),
		commandPosture(identitymodel.CmdDeleteTenant, cqrshtmx.PostureRoleGated,
			"destroying a tenant is an administrative act"),
		commandPosture(identitymodel.CmdRegisterBot, cqrshtmx.PostureRoleGated,
			"minting an API token is an administrative act"),
		commandPosture(identitymodel.CmdDeleteBot, cqrshtmx.PostureRoleGated,
			"revoking a bot is an administrative act"),
	}
}

// commandPosture builds one declaration; the kind is always the command
// dispatcher, because usermgmt registers no queries.
func commandPosture(cmdType command.Type, posture cqrshtmx.Posture, why string) cqrshtmx.DispatchPosture {
	return cqrshtmx.DispatchPosture{
		Kind:    cqrshtmx.DispatchCommandKind,
		Name:    string(cmdType),
		Posture: posture,
		Why:     why,
	}
}
