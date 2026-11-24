package identitymodel

import (
	"slices"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// --- User commands ---

type RegisterUserCmd struct {
	*command.BasicCommand
	email       string
	displayName string
	roles       []Role
}

func NewRegisterUserCmd(
	aggID id.StreamID, email, displayName string, roles []Role,
) *RegisterUserCmd {
	return &RegisterUserCmd{
		BasicCommand: mustCommand(CmdRegisterUser, aggID),
		email:        email,
		displayName:  displayName,
		roles:        roles,
	}
}

func (c *RegisterUserCmd) Email() string       { return c.email }
func (c *RegisterUserCmd) DisplayName() string { return c.displayName }

// Roles returns a copy (S38f): mutating the returned slice must not write
// through to the command — commands are immutable once constructed.
func (c *RegisterUserCmd) Roles() []Role { return slices.Clone(c.roles) }

type ChangeEmailCmd struct {
	*command.BasicCommand
	email string
}

func NewChangeEmailCmd(aggID id.StreamID, email string) *ChangeEmailCmd {
	return &ChangeEmailCmd{
		BasicCommand: mustCommand(CmdChangeEmail, aggID),
		email:        email,
	}
}

func (c *ChangeEmailCmd) Email() string { return c.email }

type ChangeDisplayNameCmd struct {
	*command.BasicCommand
	displayName string
}

func NewChangeDisplayNameCmd(aggID id.StreamID, displayName string) *ChangeDisplayNameCmd {
	return &ChangeDisplayNameCmd{
		BasicCommand: mustCommand(CmdChangeDisplayName, aggID),
		displayName:  displayName,
	}
}

func (c *ChangeDisplayNameCmd) DisplayName() string { return c.displayName }

type DeleteUserCmd struct {
	reasonedCommand
}

func NewDeleteUserCmd(aggID id.StreamID, reason string) *DeleteUserCmd {
	return &DeleteUserCmd{reasonedCommand: newReasonedCommand(CmdDeleteUser, aggID, reason)}
}

type AddCredentialCmd struct {
	*command.BasicCommand
	credential WebAuthnCredential
}

func NewAddCredentialCmd(aggID id.StreamID, cred WebAuthnCredential) *AddCredentialCmd {
	return &AddCredentialCmd{
		BasicCommand: mustCommand(CmdAddCredential, aggID),
		credential:   cred,
	}
}

func (c *AddCredentialCmd) Credential() WebAuthnCredential { return c.credential }

type RemoveCredentialCmd struct {
	*command.BasicCommand
	credentialID []byte
}

func NewRemoveCredentialCmd(aggID id.StreamID, credID []byte) *RemoveCredentialCmd {
	return &RemoveCredentialCmd{
		BasicCommand: mustCommand(CmdRemoveCredential, aggID),
		credentialID: credID,
	}
}

// CredentialID returns a copy (S38f): callers must not be able to mutate the
// command's credential reference through the accessor.
func (c *RemoveCredentialCmd) CredentialID() []byte { return slices.Clone(c.credentialID) }

// RecordCredentialUseCmd records the credential state observed during a
// successful WebAuthn assertion (signature counter + flags).
type RecordCredentialUseCmd struct {
	*command.BasicCommand
	use CredentialUse
}

func NewRecordCredentialUseCmd(aggID id.StreamID, use CredentialUse) *RecordCredentialUseCmd {
	return &RecordCredentialUseCmd{
		BasicCommand: mustCommand(CmdRecordCredentialUse, aggID),
		use:          use,
	}
}

func (c *RecordCredentialUseCmd) Use() CredentialUse { return c.use }

type VerifyEmailCmd struct {
	*command.BasicCommand
}

func NewVerifyEmailCmd(aggID id.StreamID) *VerifyEmailCmd {
	return &VerifyEmailCmd{
		BasicCommand: mustCommand(CmdVerifyEmail, aggID),
	}
}

type EnableTOTPCmd struct {
	*command.BasicCommand
	secret []byte
}

func NewEnableTOTPCmd(aggID id.StreamID, secret []byte) *EnableTOTPCmd {
	return &EnableTOTPCmd{
		BasicCommand: mustCommand(CmdEnableTOTP, aggID),
		secret:       secret,
	}
}

// Secret returns a copy (S38f): the TOTP secret is credential material; an
// accessor must never hand out a mutable alias to it.
func (c *EnableTOTPCmd) Secret() []byte { return slices.Clone(c.secret) }

type DisableTOTPCmd struct {
	*command.BasicCommand
}

func NewDisableTOTPCmd(aggID id.StreamID) *DisableTOTPCmd {
	return &DisableTOTPCmd{
		BasicCommand: mustCommand(CmdDisableTOTP, aggID),
	}
}

type LinkExternalAccountCmd struct {
	*command.BasicCommand
	provider    string
	subject     string
	email       string
	displayName string
}

func NewLinkExternalAccountCmd(
	aggID id.StreamID, provider, subject, email, displayName string,
) *LinkExternalAccountCmd {
	return &LinkExternalAccountCmd{
		BasicCommand: mustCommand(CmdLinkExternalAccount, aggID),
		provider:     provider,
		subject:      subject,
		email:        email,
		displayName:  displayName,
	}
}

func (c *LinkExternalAccountCmd) Provider() string    { return c.provider }
func (c *LinkExternalAccountCmd) Subject() string     { return c.subject }
func (c *LinkExternalAccountCmd) Email() string       { return c.email }
func (c *LinkExternalAccountCmd) DisplayName() string { return c.displayName }

type UnlinkExternalAccountCmd struct {
	*command.BasicCommand
	provider string
	subject  string
}

func NewUnlinkExternalAccountCmd(
	aggID id.StreamID, provider, subject string,
) *UnlinkExternalAccountCmd {
	return &UnlinkExternalAccountCmd{
		BasicCommand: mustCommand(CmdUnlinkExternalAccount, aggID),
		provider:     provider,
		subject:      subject,
	}
}

func (c *UnlinkExternalAccountCmd) Provider() string { return c.provider }
func (c *UnlinkExternalAccountCmd) Subject() string  { return c.subject }

// MergeContribution is what one merged-away user hands to the survivor of a
// merge: the authenticators and proof of email ownership it accumulated on
// its own stream. The survivor's decider decides what is adopted.
type MergeContribution struct {
	UserID           UserID
	Email            string
	EmailVerified    bool
	TOTPSecret       []byte
	Credentials      []WebAuthnCredential
	ExternalAccounts []ExternalAccount
}

// Clone returns a deep copy whose slice fields share no memory with c.
func (c MergeContribution) Clone() MergeContribution {
	cp := c
	cp.TOTPSecret = append([]byte(nil), c.TOTPSecret...)
	cp.Credentials = make([]WebAuthnCredential, len(c.Credentials))
	for i, cred := range c.Credentials {
		cp.Credentials[i] = cred.Clone()
	}
	cp.ExternalAccounts = append([]ExternalAccount(nil), c.ExternalAccounts...)
	return cp
}

func cloneMergeContributions(in []MergeContribution) []MergeContribution {
	out := make([]MergeContribution, len(in))
	for i, contribution := range in {
		out[i] = contribution.Clone()
	}
	return out
}

// MergeUsersCmd folds duplicate users into the survivor stream it targets.
// One decision adopts every contribution the survivor does not already hold
// and then records a single UserMerged event, so the survivor side of a
// merge is atomic.
type MergeUsersCmd struct {
	reasonedCommand
	contributions []MergeContribution
}

func NewMergeUsersCmd(
	aggID id.StreamID, contributions []MergeContribution, reason string,
) *MergeUsersCmd {
	return &MergeUsersCmd{
		reasonedCommand: newReasonedCommand(CmdMergeUsers, aggID, reason),
		contributions:   cloneMergeContributions(contributions),
	}
}

// Contributions returns a deep copy (S38f): the TOTP secret and public keys
// are credential material that must not be mutable through the accessor.
func (c *MergeUsersCmd) Contributions() []MergeContribution {
	return cloneMergeContributions(c.contributions)
}

// MergeUserIntoCmd tombstones the user stream it targets as merged into
// SurvivorID. It is the last step of a merge: until it lands the user is
// still live, so a failed merge is retried rather than left half-dead.
type MergeUserIntoCmd struct {
	reasonedCommand
	survivorID UserID
}

func NewMergeUserIntoCmd(aggID id.StreamID, survivorID UserID, reason string) *MergeUserIntoCmd {
	return &MergeUserIntoCmd{
		reasonedCommand: newReasonedCommand(CmdMergeUserInto, aggID, reason),
		survivorID:      survivorID,
	}
}

func (c *MergeUserIntoCmd) SurvivorID() UserID { return c.survivorID }

// --- Membership commands ---

// DeriveMembershipID constructs a deterministic AggregateID for a membership.
func DeriveMembershipID(actorID ActorID, tenantID TenantID) id.StreamID {
	return id.DeriveStreamID("membership", actorID.PrefixedString(), tenantID.Get())
}

type AddMemberCmd struct {
	*command.BasicCommand
	actorID  ActorID
	tenantID TenantID
	roles    []Role
}

func NewAddMemberCmd(actorID ActorID, tenantID TenantID, roles []Role) *AddMemberCmd {
	return &AddMemberCmd{
		BasicCommand: mustCommand(CmdAddMember, DeriveMembershipID(actorID, tenantID)),
		actorID:      actorID,
		tenantID:     tenantID,
		roles:        roles,
	}
}

func (c *AddMemberCmd) ActorID() ActorID   { return c.actorID }
func (c *AddMemberCmd) TenantID() TenantID { return c.tenantID }

// Roles returns a copy (S38f): mutating the returned slice must not write
// through to the command.
func (c *AddMemberCmd) Roles() []Role { return slices.Clone(c.roles) }

type UpdateMemberRolesCmd struct {
	*command.BasicCommand
	roles []Role
}

func NewUpdateMemberRolesCmd(
	actorID ActorID, tenantID TenantID, roles []Role,
) *UpdateMemberRolesCmd {
	return &UpdateMemberRolesCmd{
		BasicCommand: mustCommand(CmdUpdateMemberRoles, DeriveMembershipID(actorID, tenantID)),
		roles:        roles,
	}
}

// Roles returns a copy (S38f): mutating the returned slice must not write
// through to the command.
func (c *UpdateMemberRolesCmd) Roles() []Role { return slices.Clone(c.roles) }

type RemoveMemberCmd struct {
	*command.BasicCommand
}

func NewRemoveMemberCmd(actorID ActorID, tenantID TenantID) *RemoveMemberCmd {
	return &RemoveMemberCmd{
		BasicCommand: mustCommand(CmdRemoveMember, DeriveMembershipID(actorID, tenantID)),
	}
}

// --- Tenant commands ---

type CreateTenantCmd struct {
	*command.BasicCommand
	name        string
	displayName string
}

func NewCreateTenantCmd(aggID id.StreamID, name, displayName string) *CreateTenantCmd {
	return &CreateTenantCmd{
		BasicCommand: mustCommand(CmdCreateTenant, aggID),
		name:         name,
		displayName:  displayName,
	}
}

func (c *CreateTenantCmd) Name() string        { return c.name }
func (c *CreateTenantCmd) DisplayName() string { return c.displayName }

type SuspendTenantCmd struct {
	reasonedCommand
}

func NewSuspendTenantCmd(aggID id.StreamID, reason string) *SuspendTenantCmd {
	return &SuspendTenantCmd{reasonedCommand: newReasonedCommand(CmdSuspendTenant, aggID, reason)}
}

type ReactivateTenantCmd struct {
	*command.BasicCommand
}

func NewReactivateTenantCmd(aggID id.StreamID) *ReactivateTenantCmd {
	return &ReactivateTenantCmd{
		BasicCommand: mustCommand(CmdReactivateTenant, aggID),
	}
}

type DeleteTenantCmd struct {
	reasonedCommand
}

func NewDeleteTenantCmd(aggID id.StreamID, reason string) *DeleteTenantCmd {
	return &DeleteTenantCmd{reasonedCommand: newReasonedCommand(CmdDeleteTenant, aggID, reason)}
}

// RenameTenantCmd changes the display name of the tenant stream it targets.
// The tenant ID is untouched.
type RenameTenantCmd struct {
	*command.BasicCommand
	displayName string
}

func NewRenameTenantCmd(aggID id.StreamID, displayName string) *RenameTenantCmd {
	return &RenameTenantCmd{
		BasicCommand: mustCommand(CmdRenameTenant, aggID),
		displayName:  displayName,
	}
}

func (c *RenameTenantCmd) DisplayName() string { return c.displayName }

// MergeTenantsCmd records on the surviving tenant's stream that the listed
// tenants were consolidated into it.
type MergeTenantsCmd struct {
	reasonedCommand
	loserIDs []TenantID
}

func NewMergeTenantsCmd(aggID id.StreamID, loserIDs []TenantID, reason string) *MergeTenantsCmd {
	return &MergeTenantsCmd{
		reasonedCommand: newReasonedCommand(CmdMergeTenants, aggID, reason),
		loserIDs:       slices.Clone(loserIDs),
	}
}

// LoserIDs returns a copy (S38f): mutating the returned slice must not write
// through to the command.
func (c *MergeTenantsCmd) LoserIDs() []TenantID { return slices.Clone(c.loserIDs) }

// MergeTenantIntoCmd tombstones the tenant stream it targets as merged into
// SurvivorID. It is the last step of a merge, after the roster has moved.
type MergeTenantIntoCmd struct {
	reasonedCommand
	survivorID TenantID
}

func NewMergeTenantIntoCmd(aggID id.StreamID, survivorID TenantID, reason string) *MergeTenantIntoCmd {
	return &MergeTenantIntoCmd{
		reasonedCommand: newReasonedCommand(CmdMergeTenantInto, aggID, reason),
		survivorID:      survivorID,
	}
}

func (c *MergeTenantIntoCmd) SurvivorID() TenantID { return c.survivorID }

// --- Bot commands ---

type RegisterBotCmd struct {
	*command.BasicCommand
	name      string
	ownerID   UserID
	tokenHash []byte
	scopes    []string
}

func NewRegisterBotCmd(
	aggID id.StreamID, name string, ownerID UserID, tokenHash []byte, scopes []string,
) *RegisterBotCmd {
	return &RegisterBotCmd{
		BasicCommand: mustCommand(CmdRegisterBot, aggID),
		name:         name,
		ownerID:      ownerID,
		tokenHash:    tokenHash,
		scopes:       scopes,
	}
}

func (c *RegisterBotCmd) Name() string    { return c.name }
func (c *RegisterBotCmd) OwnerID() UserID { return c.ownerID }

// TokenHash and Scopes return copies (S38f): the token hash is credential
// material and the scopes are the command's authorization surface; neither
// may be mutated through an accessor.
func (c *RegisterBotCmd) TokenHash() []byte { return slices.Clone(c.tokenHash) }

func (c *RegisterBotCmd) Scopes() []string { return slices.Clone(c.scopes) }

type DeleteBotCmd struct {
	reasonedCommand
}

func NewDeleteBotCmd(aggID id.StreamID, reason string) *DeleteBotCmd {
	return &DeleteBotCmd{reasonedCommand: newReasonedCommand(CmdDeleteBot, aggID, reason)}
}

// TransferBotCmd moves the bot stream it targets to a new owner. The bot ID
// and token hash stay, so API tokens survive the merge of their owner.
type TransferBotCmd struct {
	reasonedCommand
	ownerID UserID
}

func NewTransferBotCmd(aggID id.StreamID, ownerID UserID, reason string) *TransferBotCmd {
	return &TransferBotCmd{
		reasonedCommand: newReasonedCommand(CmdTransferBot, aggID, reason),
		ownerID:       ownerID,
	}
}

func (c *TransferBotCmd) OwnerID() UserID { return c.ownerID }

// reasonedCommand is the shared base for commands that carry a human-readable
// reason (deletions, suspensions). Embedding it gives the command the standard
// *command.BasicCommand embedding plus a Reason() accessor, so each concrete
// command only declares its type and constructor.
// cqrs-lint:ignore(E005) shared base struct embedded by concrete commands, never dispatched directly
type reasonedCommand struct {
	*command.BasicCommand
	reason string
}

// Reason returns the human-readable rationale supplied with the command.
func (c *reasonedCommand) Reason() string { return c.reason }

// newReasonedCommand builds the shared base for a reason-carrying command.
func newReasonedCommand(cmdType command.Type, aggID id.StreamID, reason string) reasonedCommand {
	return reasonedCommand{
		BasicCommand: mustCommand(cmdType, aggID),
		reason:       reason,
	}
}

// mustCommand creates a BasicCommand, panicking on error.
func mustCommand(cmdType command.Type, aggID id.StreamID) *command.BasicCommand {
	base, err := command.New(cmdType, aggID)
	if err != nil {
		panic(errorfamily.Wrapf(
			err, event.Infrastructure,
			"usermgmt.command.create_failed",
			"create %s command", cmdType,
		))
	}
	return base
}
