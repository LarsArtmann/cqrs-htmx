// Package systemadapter bridges cqrs-htmx's identity-model domain types with
// go-cqrs-lite's system.New() composition root and metaengine projection planner.
//
// This module lets consumers use system.New() as their infrastructure backbone
// while still getting all 20 identity-model commands and 21 event types wired
// automatically. The consumer provides a DeploymentConfig (engines, buses) and
// calls system.New() with the DomainConfig from this package.
//
// Example:
//
//	domain := systemadapter.DomainConfig()
//	deployment := system.DeploymentConfig{
//		Engines: map[string]system.EngineConfig{
//			"primary": {Driver: "memory"},
//		},
//		Instances: []system.InstanceConfig{
//			{Role: system.RoleSourceOfTruth, Engines: []string{"primary"}},
//			{Role: system.RoleProjections, Engines: []string{"primary"}},
//		},
//	}
//	sys, err := system.New(ctx, domain, deployment)
package systemadapter

import (
	"context"
	"fmt"

	"github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// DomainConfig returns a system.DomainConfig pre-wired with all four identity-model
// deciders (User, Membership, Tenant, Bot), all 20 commands, and a TypeDecoder
// that maps all 21 event types to their payload structs.
//
// The consumer passes this to system.New() alongside a DeploymentConfig that
// specifies engines and buses. The system handles repository creation, event
// store wiring, bus fan-out, and projection host lifecycle automatically.
//
// Additional projections (read models, Casbin authz) can be registered on the
// system's projection host between system.New() and sys.Start():
//
//	sys, _ := system.New(ctx, systemadapter.DomainConfig(), deployment)
//	systemadapter.RegisterProjections(sys.ProjectionHost(), sys.EventStore())
//	sys.Start(ctx)
func DomainConfig(opts ...DomainConfigOption) system.DomainConfig {
	cfg := defaultDomainConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg.domainConfig
}

// DomainConfigOption customizes the DomainConfig.
type DomainConfigOption func(*domainConfigBuilder)

type domainConfigBuilder struct {
	domainConfig system.DomainConfig
}

func defaultDomainConfig() domainConfigBuilder {
	return domainConfigBuilder{
		domainConfig: system.DomainConfig{
			Commands:              registerAllCommands,
			ProjectionTypeDecoder: EventTypeDecoder(),
		},
	}
}

// registerAllCommands is the Commands closure passed to system.DomainConfig.
// It registers all four deciders and all 20 commands on the system.
func registerAllCommands(sys *system.System) {
	registerUserDecider(sys)
	registerMembershipDecider(sys)
	registerTenantDecider(sys)
	registerBotDecider(sys)

	registerUserCommands(sys)
	registerMembershipCommands(sys)
	registerTenantCommands(sys)
	registerBotCommands(sys)
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("systemadapter: registration failed: %v", err))
	}
}

func registerUserDecider(sys *system.System) {
	must(system.RegisterDecider[usermgmt.UserState](
		sys,
		string(identitymodel.AggregateTypeUser),
		usermgmt.UserDecider(),
	))
}

func registerMembershipDecider(sys *system.System) {
	must(system.RegisterDecider[usermgmt.MembershipState](
		sys,
		string(identitymodel.AggregateTypeMembership),
		usermgmt.MembershipDecider(),
	))
}

func registerTenantDecider(sys *system.System) {
	must(system.RegisterDecider[usermgmt.TenantState](
		sys,
		string(identitymodel.AggregateTypeTenant),
		usermgmt.TenantDecider(),
	))
}

func registerBotDecider(sys *system.System) {
	must(system.RegisterDecider[usermgmt.BotState](
		sys,
		string(identitymodel.AggregateTypeBot),
		usermgmt.BotDecider(),
	))
}

// --- User Commands ---

func registerUserCommands(sys *system.System) {
	must(system.RegisterCommand[*identitymodel.RegisterUserCmd, usermgmt.UserState](
		sys, identitymodel.CmdRegisterUser,
		func(_ context.Context, c *identitymodel.RegisterUserCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideRegisterUser(c.StreamID(), c.Email(), c.DisplayName(), c.Roles()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.ChangeEmailCmd, usermgmt.UserState](
		sys, identitymodel.CmdChangeEmail,
		func(_ context.Context, c *identitymodel.ChangeEmailCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideChangeEmail(c.StreamID(), c.Email()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.ChangeDisplayNameCmd, usermgmt.UserState](
		sys, identitymodel.CmdChangeDisplayName,
		func(_ context.Context, c *identitymodel.ChangeDisplayNameCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideChangeDisplayName(c.StreamID(), c.DisplayName()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.DeleteUserCmd, usermgmt.UserState](
		sys, identitymodel.CmdDeleteUser,
		func(_ context.Context, c *identitymodel.DeleteUserCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideDeleteUser(c.StreamID(), c.Reason()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.AddCredentialCmd, usermgmt.UserState](
		sys, identitymodel.CmdAddCredential,
		func(_ context.Context, c *identitymodel.AddCredentialCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideAddCredential(c.StreamID(), c.Credential()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.RemoveCredentialCmd, usermgmt.UserState](
		sys, identitymodel.CmdRemoveCredential,
		func(_ context.Context, c *identitymodel.RemoveCredentialCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideRemoveCredential(c.StreamID(), c.CredentialID()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.VerifyEmailCmd, usermgmt.UserState](
		sys, identitymodel.CmdVerifyEmail,
		func(_ context.Context, c *identitymodel.VerifyEmailCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideVerifyEmail(c.StreamID()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.EnableTOTPCmd, usermgmt.UserState](
		sys, identitymodel.CmdEnableTOTP,
		func(_ context.Context, c *identitymodel.EnableTOTPCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideEnableTOTP(c.StreamID(), c.Secret()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.DisableTOTPCmd, usermgmt.UserState](
		sys, identitymodel.CmdDisableTOTP,
		func(_ context.Context, c *identitymodel.DisableTOTPCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideDisableTOTP(c.StreamID()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.LinkExternalAccountCmd, usermgmt.UserState](
		sys, identitymodel.CmdLinkExternalAccount,
		func(_ context.Context, c *identitymodel.LinkExternalAccountCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideLinkExternalAccount(
					c.StreamID(), c.Provider(), c.Subject(), c.Email(), c.DisplayName(),
				),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.UnlinkExternalAccountCmd, usermgmt.UserState](
		sys, identitymodel.CmdUnlinkExternalAccount,
		func(_ context.Context, c *identitymodel.UnlinkExternalAccountCmd) system.Op[usermgmt.UserState] {
			return system.Execute[usermgmt.UserState](
				nil, c.StreamID(), identitymodel.AggregateTypeUser,
				usermgmt.DecideUnlinkExternalAccount(c.StreamID(), c.Provider(), c.Subject()),
			)
		},
	))
}

// --- Membership Commands ---

func registerMembershipCommands(sys *system.System) {
	must(system.RegisterCommand[*identitymodel.AddMemberCmd, usermgmt.MembershipState](
		sys, identitymodel.CmdAddMember,
		func(_ context.Context, c *identitymodel.AddMemberCmd) system.Op[usermgmt.MembershipState] {
			return system.Execute[usermgmt.MembershipState](
				nil, c.StreamID(), identitymodel.AggregateTypeMembership,
				usermgmt.DecideAddMember(c.StreamID(), c.ActorID(), c.TenantID(), c.Roles()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.UpdateMemberRolesCmd, usermgmt.MembershipState](
		sys, identitymodel.CmdUpdateMemberRoles,
		func(_ context.Context, c *identitymodel.UpdateMemberRolesCmd) system.Op[usermgmt.MembershipState] {
			return system.Execute[usermgmt.MembershipState](
				nil, c.StreamID(), identitymodel.AggregateTypeMembership,
				usermgmt.DecideUpdateMemberRoles(c.StreamID(), c.Roles()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.RemoveMemberCmd, usermgmt.MembershipState](
		sys, identitymodel.CmdRemoveMember,
		func(_ context.Context, c *identitymodel.RemoveMemberCmd) system.Op[usermgmt.MembershipState] {
			return system.Execute[usermgmt.MembershipState](
				nil, c.StreamID(), identitymodel.AggregateTypeMembership,
				usermgmt.DecideRemoveMember(c.StreamID()),
			)
		},
	))
}

// --- Tenant Commands ---

func registerTenantCommands(sys *system.System) {
	must(system.RegisterCommand[*identitymodel.CreateTenantCmd, usermgmt.TenantState](
		sys, identitymodel.CmdCreateTenant,
		func(_ context.Context, c *identitymodel.CreateTenantCmd) system.Op[usermgmt.TenantState] {
			return system.Execute[usermgmt.TenantState](
				nil, c.StreamID(), identitymodel.AggregateTypeTenant,
				usermgmt.DecideCreateTenant(c.StreamID(), c.Name(), c.DisplayName()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.SuspendTenantCmd, usermgmt.TenantState](
		sys, identitymodel.CmdSuspendTenant,
		func(_ context.Context, c *identitymodel.SuspendTenantCmd) system.Op[usermgmt.TenantState] {
			return system.Execute[usermgmt.TenantState](
				nil, c.StreamID(), identitymodel.AggregateTypeTenant,
				usermgmt.DecideSuspendTenant(c.StreamID(), c.Reason()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.ReactivateTenantCmd, usermgmt.TenantState](
		sys, identitymodel.CmdReactivateTenant,
		func(_ context.Context, c *identitymodel.ReactivateTenantCmd) system.Op[usermgmt.TenantState] {
			return system.Execute[usermgmt.TenantState](
				nil, c.StreamID(), identitymodel.AggregateTypeTenant,
				usermgmt.DecideReactivateTenant(c.StreamID()),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.DeleteTenantCmd, usermgmt.TenantState](
		sys, identitymodel.CmdDeleteTenant,
		func(_ context.Context, c *identitymodel.DeleteTenantCmd) system.Op[usermgmt.TenantState] {
			return system.Execute[usermgmt.TenantState](
				nil, c.StreamID(), identitymodel.AggregateTypeTenant,
				usermgmt.DecideDeleteTenant(c.StreamID(), c.Reason()),
			)
		},
	))
}

// --- Bot Commands ---

func registerBotCommands(sys *system.System) {
	must(system.RegisterCommand[*identitymodel.RegisterBotCmd, usermgmt.BotState](
		sys, identitymodel.CmdRegisterBot,
		func(_ context.Context, c *identitymodel.RegisterBotCmd) system.Op[usermgmt.BotState] {
			return system.Execute[usermgmt.BotState](
				nil, c.StreamID(), identitymodel.AggregateTypeBot,
				usermgmt.DecideRegisterBot(
					c.StreamID(), c.Name(), c.OwnerID(), c.TokenHash(), c.Scopes(),
				),
			)
		},
	))

	must(system.RegisterCommand[*identitymodel.DeleteBotCmd, usermgmt.BotState](
		sys, identitymodel.CmdDeleteBot,
		func(_ context.Context, c *identitymodel.DeleteBotCmd) system.Op[usermgmt.BotState] {
			return system.Execute[usermgmt.BotState](
				nil, c.StreamID(), identitymodel.AggregateTypeBot,
				usermgmt.DecideDeleteBot(c.StreamID(), c.Reason()),
			)
		},
	))
}
