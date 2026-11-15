// Package gflusermgmt rebuilds the usermgmt decider tests as GFL-authored
// executable specs: usermgmt.gfl describes the scenarios in prose,
// gfl.bindings.json maps each step to real identity-model commands and
// systemadapter read-model queries, and the gfl CLI compiles them into
// usermgmt_test.go — which runs against the SAME deciders the hand-written
// scenario tests in ../usermgmt/es_scenario*_test.go exercise.
//
// This file is the only hand-written Go: fixed streams plus one helper per
// bound step. Everything else is generated or pinned by events_sync_test.go.
package gflusermgmt

import (
	"context"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// Fixed streams, one per aggregate, mirroring the counter example's pattern:
// every scenario boots a fresh in-memory system, so names need no uniqueness.
var (
	userStream       = mustStream("user-gfl")
	tenantStream     = mustStream("tenant-gfl")
	botStream        = mustStream("bot-gfl")
	memberActor      = identitymodel.NewActorID(identitymodel.ActorUser, "01JXGFLUSER0000000000000A1")
	memberTenant     = identitymodel.NewTenantID("01JXGFLTENANT0000000000000B2")
	botOwner         = identitymodel.NewUserID("01JXGFLOWNER0000000000000C3")
	botTokenHash     = []byte{0xDE, 0xAD, 0xBE, 0xEF}
	membershipStream = identitymodel.DeriveMembershipID(memberActor, memberTenant)
)

func mustStream(name string) id.StreamID {
	stream, err := id.ParseStreamID(name)
	if err != nil {
		panic("gfl-usermgmt: parse stream id: " + err.Error())
	}
	return stream
}

func userRef() id.StreamRef {
	return id.NewStreamRef(id.StreamType(identitymodel.AggregateTypeUser), userStream)
}

func tenantRef() id.StreamRef {
	return id.NewStreamRef(id.StreamType(identitymodel.AggregateTypeTenant), tenantStream)
}

func botRef() id.StreamRef {
	return id.NewStreamRef(id.StreamType(identitymodel.AggregateTypeBot), botStream)
}

func membershipRef() id.StreamRef {
	return id.NewStreamRef(id.StreamType(identitymodel.AggregateTypeMembership), membershipStream)
}

// parseRoles splits a prose role list ("admin, writer") into identity-model roles.
func parseRoles(roles string) []identitymodel.Role {
	parts := strings.Split(roles, ",")
	parsed := make([]identitymodel.Role, 0, len(parts))
	for _, part := range parts {
		parsed = append(parsed, identitymodel.Role(strings.TrimSpace(part)))
	}
	return parsed
}

// --- Given helpers: mint seed events, mirroring the deciders' own emissions ---

// RegisteredUser seeds UserRegistered on the user stream.
func RegisteredUser(sc *systemscenario.Scenario) []event.Event {
	return []event.Event{sc.Event(identitymodel.EventUserRegistered, userRef(), identitymodel.UserRegisteredPayload{
		SchemaVersion: identitymodel.CurrentSchemaVersion,
		Email:         "user@test.com",
		DisplayName:   "Test User",
		Roles:         []identitymodel.Role{"admin"},
	})}
}

// VerifiedUser seeds UserRegistered plus EmailVerified.
func VerifiedUser(sc *systemscenario.Scenario) []event.Event {
	return append(
		RegisteredUser(sc),
		sc.Event(identitymodel.EventEmailVerified, userRef(), identitymodel.EmailVerifiedPayload{
			SchemaVersion: identitymodel.CurrentSchemaVersion,
			Email:         "user@test.com",
		}),
	)
}

// CreatedTenant seeds TenantCreated with the given name.
func CreatedTenant(sc *systemscenario.Scenario, name string) []event.Event {
	return []event.Event{sc.Event(identitymodel.EventTenantCreated, tenantRef(), identitymodel.TenantCreatedPayload{
		SchemaVersion: identitymodel.CurrentSchemaVersion,
		Name:          name,
		DisplayName:   name,
	})}
}

// SuspendedTenant seeds TenantCreated plus TenantSuspended.
func SuspendedTenant(sc *systemscenario.Scenario, name string) []event.Event {
	return append(
		CreatedTenant(sc, name),
		sc.Event(identitymodel.EventTenantSuspended, tenantRef(), identitymodel.TenantSuspendedPayload{
			SchemaVersion: identitymodel.CurrentSchemaVersion,
			Reason:        "seeded",
		}),
	)
}

// MemberWithRoles seeds MemberAdded on the derived membership stream.
func MemberWithRoles(sc *systemscenario.Scenario, roles string) []event.Event {
	return []event.Event{sc.Event(identitymodel.EventMemberAdded, membershipRef(), identitymodel.MemberAddedPayload{
		SchemaVersion: identitymodel.CurrentSchemaVersion,
		ActorKind:     memberActor.Kind().String(),
		ActorID:       memberActor.String(),
		TenantID:      memberTenant.Get(),
		Roles:         parseRoles(roles),
	})}
}

// RegisteredBot seeds BotRegistered owned by the fixed bot owner.
func RegisteredBot(sc *systemscenario.Scenario) []event.Event {
	return []event.Event{sc.Event(identitymodel.EventBotRegistered, botRef(), identitymodel.BotRegisteredPayload{
		SchemaVersion: identitymodel.CurrentSchemaVersion,
		Name:          "deploy-bot",
		OwnerID:       botOwner,
		TokenHash:     botTokenHash,
		Scopes:        []string{"deploy"},
	})}
}

// --- When helpers: build real identity-model commands on the fixed streams ---

// RegisterUser builds the RegisterUser command (happy path: fresh stream).
func RegisterUser() *identitymodel.RegisterUserCmd {
	return identitymodel.NewRegisterUserCmd(userStream, "user@test.com", "Test User", []identitymodel.Role{"admin"})
}

// ChangeEmail builds the ChangeEmail command.
func ChangeEmail(email string) *identitymodel.ChangeEmailCmd {
	return identitymodel.NewChangeEmailCmd(userStream, email)
}

// VerifyEmail builds the VerifyEmail command.
func VerifyEmail() *identitymodel.VerifyEmailCmd {
	return identitymodel.NewVerifyEmailCmd(userStream)
}

// ChangeDisplayName builds the ChangeDisplayName command.
func ChangeDisplayName(displayName string) *identitymodel.ChangeDisplayNameCmd {
	return identitymodel.NewChangeDisplayNameCmd(userStream, displayName)
}

// CreateTenant builds the CreateTenant command.
func CreateTenant(name string) *identitymodel.CreateTenantCmd {
	return identitymodel.NewCreateTenantCmd(tenantStream, name, name)
}

// SuspendTenant builds the SuspendTenant command.
func SuspendTenant(reason string) *identitymodel.SuspendTenantCmd {
	return identitymodel.NewSuspendTenantCmd(tenantStream, reason)
}

// ReactivateTenant builds the ReactivateTenant command.
func ReactivateTenant() *identitymodel.ReactivateTenantCmd {
	return identitymodel.NewReactivateTenantCmd(tenantStream)
}

// DeleteTenant builds the DeleteTenant command.
func DeleteTenant(reason string) *identitymodel.DeleteTenantCmd {
	return identitymodel.NewDeleteTenantCmd(tenantStream, reason)
}

// AddMember builds the AddMember command on the derived membership stream.
func AddMember(roles string) *identitymodel.AddMemberCmd {
	return identitymodel.NewAddMemberCmd(memberActor, memberTenant, parseRoles(roles))
}

// UpdateMemberRoles builds the UpdateMemberRoles command.
func UpdateMemberRoles(roles string) *identitymodel.UpdateMemberRolesCmd {
	return identitymodel.NewUpdateMemberRolesCmd(memberActor, memberTenant, parseRoles(roles))
}

// RemoveMember builds the RemoveMember command.
func RemoveMember() *identitymodel.RemoveMemberCmd {
	return identitymodel.NewRemoveMemberCmd(memberActor, memberTenant)
}

// RegisterBot builds the RegisterBot command.
func RegisterBot(name string) *identitymodel.RegisterBotCmd {
	return identitymodel.NewRegisterBotCmd(botStream, name, botOwner, botTokenHash, []string{"deploy"})
}

// DeleteBot builds the DeleteBot command.
func DeleteBot(reason string) *identitymodel.DeleteBotCmd {
	return identitymodel.NewDeleteBotCmd(botStream, reason)
}

// --- Then helpers: read the systemadapter declarative read models via the
// exported finders (FindUserByID / FindTenantByID). ---

// UserEmailVerified reads the user_by_id projection's EmailVerified flag.
func UserEmailVerified(ctx context.Context, sc *systemscenario.Scenario) (bool, error) {
	view, err := systemadapter.FindUserByID(ctx, sc.System(), userStream.String())
	if err != nil {
		return false, err
	}
	return view.EmailVerified, nil
}

// TenantSuspended reads the tenant_by_id projection's Suspended flag.
func TenantSuspended(ctx context.Context, sc *systemscenario.Scenario) (bool, error) {
	view, err := systemadapter.FindTenantByID(ctx, sc.System(), tenantStream.String())
	if err != nil {
		return false, err
	}
	return view.Suspended, nil
}

// TenantName reads the tenant_by_id projection's Name.
func TenantName(ctx context.Context, sc *systemscenario.Scenario) (string, error) {
	view, err := systemadapter.FindTenantByID(ctx, sc.System(), tenantStream.String())
	if err != nil {
		return "", err
	}
	return view.Name, nil
}
