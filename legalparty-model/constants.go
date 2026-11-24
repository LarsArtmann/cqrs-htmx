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

const (
	// AggregateTypeBillTo is the stream type of a tenant's bill-to link: the
	// aggregate that answers "which legal party receives this tenant's
	// invoices" (issue #45, D9). One stream per tenant, keyed by the tenant
	// ID, so a consumer mounts it beside its tenant lifecycle without a
	// join table.
	AggregateTypeBillTo event.StreamType = "BillTo"
)

const (
	// EventBillToOpened records that a tenant's billing link was created and
	// names the first legal party bills go to.
	EventBillToOpened event.Type = "BillToOpened"
	// EventBillToRedirected records that bills now go to another legal
	// party. Redirects are append-only: the previous party stays in the
	// journal, so an invoice issued before the redirect can still name the
	// party that actually received it.
	EventBillToRedirected event.Type = "BillToRedirected"
	// EventBillToClosed records that the tenant stopped being billed
	// directly. Closure is terminal (like LegalPartyClosed) but the stream
	// and its parties stay journaled for tax retention.
	EventBillToClosed event.Type = "BillToClosed"
)

const (
	// CmdOpenBillTo opens a tenant's bill-to link.
	CmdOpenBillTo command.Type = "OpenBillTo"
	// CmdRedirectBillTo points a tenant's bills at another legal party.
	CmdRedirectBillTo command.Type = "RedirectBillTo"
	// CmdCloseBillTo closes a tenant's bill-to link.
	CmdCloseBillTo command.Type = "CloseBillTo"
)
