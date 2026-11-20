package legalparty

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const (
	// AggregateTypeLegalParty is the stream type of a legal party aggregate.
	AggregateTypeLegalParty event.StreamType = "LegalParty"
)

const (
	// EventLegalPartyOpened records that a legal party entered the billing
	// system's view with its initial profile.
	EventLegalPartyOpened event.Type = "LegalPartyOpened"
	// EventLegalPartyProfileUpdated records a full profile replacement,
	// optionally carrying provenance-stamped registry facts.
	EventLegalPartyProfileUpdated event.Type = "LegalPartyProfileUpdated"
	// EventLegalPartyClosed records that the party stopped being billable.
	// Closure is terminal and non-destructive: tax retention (GoBD, 10
	// years in DE) requires the profile behind an issued invoice to
	// survive, so no event ever deletes a legal party.
	EventLegalPartyClosed event.Type = "LegalPartyClosed"
)

const (
	// CmdOpenLegalParty opens a legal party aggregate.
	CmdOpenLegalParty command.Type = "OpenLegalParty"
	// CmdUpdateLegalPartyProfile replaces the party's profile.
	CmdUpdateLegalPartyProfile command.Type = "UpdateLegalPartyProfile"
	// CmdCloseLegalParty closes a legal party (terminal, retained).
	CmdCloseLegalParty command.Type = "CloseLegalParty"
)

// CurrentSchemaVersion is the payload schema version stamped on every event
// this module emits. Bump it together with a payload shape change; folds
// keep decoding older versions (see the profile payloads' omitempty fields).
const CurrentSchemaVersion = 1
