package legalparty

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// OpenLegalPartyCmd opens a legal party aggregate with its initial profile.
type OpenLegalPartyCmd struct {
	*command.BasicCommand
	profile  LegalPartyProfile
	openedBy identitymodel.ActorID
	openedAt time.Time
}

// NewOpenLegalPartyCmd constructs the command. openedAt is a command-side
// timestamp (e.g. the importer's watermark); events normalize to UTC.
func NewOpenLegalPartyCmd(
	aggID id.StreamID, profile LegalPartyProfile, openedBy identitymodel.ActorID, openedAt time.Time,
) *OpenLegalPartyCmd {
	return &OpenLegalPartyCmd{
		BasicCommand: mustCommand(CmdOpenLegalParty, aggID),
		profile:      profile,
		openedBy:     openedBy,
		openedAt:     openedAt,
	}
}

// Profile returns the initial profile.
func (c *OpenLegalPartyCmd) Profile() LegalPartyProfile { return c.profile }

// OpenedBy returns the actor opening the party (never a bot; enforced by
// the decider).
func (c *OpenLegalPartyCmd) OpenedBy() identitymodel.ActorID { return c.openedBy }

// OpenedAt returns the command-side open timestamp.
func (c *OpenLegalPartyCmd) OpenedAt() time.Time { return c.openedAt }

// UpdateLegalPartyProfileCmd replaces the profile wholesale, optionally
// carrying provenance-stamped registry facts.
type UpdateLegalPartyProfileCmd struct {
	*command.BasicCommand
	profile       LegalPartyProfile
	registryFacts *RegistryFacts
	updatedBy     identitymodel.ActorID
	updatedAt     time.Time
}

// NewUpdateLegalPartyProfileCmd constructs the command.
func NewUpdateLegalPartyProfileCmd(
	aggID id.StreamID, profile LegalPartyProfile, facts *RegistryFacts,
	updatedBy identitymodel.ActorID, updatedAt time.Time,
) *UpdateLegalPartyProfileCmd {
	return &UpdateLegalPartyProfileCmd{
		BasicCommand:  mustCommand(CmdUpdateLegalPartyProfile, aggID),
		profile:       profile,
		registryFacts: facts,
		updatedBy:     updatedBy,
		updatedAt:     updatedAt,
	}
}

// Profile returns the replacement profile.
func (c *UpdateLegalPartyProfileCmd) Profile() LegalPartyProfile { return c.profile }

// RegistryFacts returns a copy of the provenance-stamped registry facts, or
// nil when the update carried none. Commands are immutable once constructed,
// so the copy cannot write through.
func (c *UpdateLegalPartyProfileCmd) RegistryFacts() *RegistryFacts {
	if c.registryFacts == nil {
		return nil
	}
	cp := *c.registryFacts

	return &cp
}

// UpdatedBy returns the actor maintaining the party (never a bot; enforced
// by the decider).
func (c *UpdateLegalPartyProfileCmd) UpdatedBy() identitymodel.ActorID { return c.updatedBy }

// UpdatedAt returns the command-side update timestamp.
func (c *UpdateLegalPartyProfileCmd) UpdatedAt() time.Time { return c.updatedAt }

// CloseLegalPartyCmd closes a legal party. Closure is terminal and
// non-destructive: the profile stays retained.
type CloseLegalPartyCmd struct {
	*command.BasicCommand
	reason   string
	closedBy identitymodel.ActorID
	closedAt time.Time
}

// NewCloseLegalPartyCmd constructs the command; the reason is required and
// enforced by the decider.
func NewCloseLegalPartyCmd(
	aggID id.StreamID, reason string, closedBy identitymodel.ActorID, closedAt time.Time,
) *CloseLegalPartyCmd {
	return &CloseLegalPartyCmd{
		BasicCommand: mustCommand(CmdCloseLegalParty, aggID),
		reason:       reason,
		closedBy:     closedBy,
		closedAt:     closedAt,
	}
}

// Reason returns the human-readable closure reason.
func (c *CloseLegalPartyCmd) Reason() string { return c.reason }

// ClosedBy returns the actor closing the party (never a bot; enforced by
// the decider).
func (c *CloseLegalPartyCmd) ClosedBy() identitymodel.ActorID { return c.closedBy }

// ClosedAt returns the command-side close timestamp.
func (c *CloseLegalPartyCmd) ClosedAt() time.Time { return c.closedAt }

// mustCommand creates a BasicCommand, panicking on error.
func mustCommand(cmdType command.Type, aggID id.StreamID) *command.BasicCommand {
	base, err := command.New(cmdType, aggID)
	if err != nil {
		panic(errorfamily.Wrapf(
			err, event.Infrastructure,
			"legalparty.command.create_failed",
			"create %s command", cmdType,
		))
	}

	return base
}
