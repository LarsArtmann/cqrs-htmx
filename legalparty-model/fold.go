package legalparty

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// LegalPartyState is the aggregate state of a legal party, reconstructed by
// folding events. There is no Deleted flag by design: closure is terminal
// and non-destructive, so the profile behind an issued invoice survives
// (tax retention).
type LegalPartyState struct {
	Profile           LegalPartyProfile
	Opened            bool
	Closed            bool
	CloseReason       string
	OpenedAt          time.Time
	LastUpdatedAt     time.Time
	ClosedAt          time.Time
	LastRegistryFacts *RegistryFacts
}

// Exists reports whether the legal party has been opened (at least one
// event on the stream).
func (s LegalPartyState) Exists() bool { return s.Opened }

// IsOpen reports whether the party exists and is not closed.
func (s LegalPartyState) IsOpen() bool { return s.Opened && !s.Closed }

// IsValid enforces struct-level invariants: closure flags, reasons and
// timestamps agree with each other.
func (s LegalPartyState) IsValid() bool {
	if s.Closed && s.CloseReason == "" {
		return false
	}
	if !s.Closed && s.CloseReason != "" {
		return false
	}
	if s.Closed && s.ClosedAt.IsZero() {
		return false
	}
	if !s.Closed && !s.ClosedAt.IsZero() {
		return false
	}
	if !s.Opened && !s.OpenedAt.IsZero() {
		return false
	}

	return true
}

// FoldLegalParty applies an event to the current LegalPartyState.
// cqrs-lint:ignore(B005) canonical fold pattern: switch is the standard ES fold
func FoldLegalParty(
	state LegalPartyState,
	evt event.Event,
) (LegalPartyState, error) {
	next := state

	switch evt.Type() {
	case EventLegalPartyOpened:
		p, err := UnmarshalPayload[LegalPartyOpenedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Profile = p.Profile
		next.Opened = true
		next.Closed = false
		next.CloseReason = ""
		next.OpenedAt = p.OpenedAt
		next.LastUpdatedAt = p.OpenedAt
		next.ClosedAt = time.Time{}
		next.LastRegistryFacts = nil

	case EventLegalPartyProfileUpdated:
		p, err := UnmarshalPayload[LegalPartyProfileUpdatedPayload](evt)
		if err != nil {
			return state, err
		}
		next.Profile = p.Profile
		next.LastRegistryFacts = p.RegistryFacts
		next.LastUpdatedAt = p.UpdatedAt

	case EventLegalPartyClosed:
		p, err := UnmarshalPayload[LegalPartyClosedPayload](evt)
		if err != nil {
			return state, err
		}
		// Retention: the profile and the last registry facts stay intact —
		// closure marks the party non-billable, it never erases the
		// identity behind an issued invoice.
		next.Closed = true
		next.CloseReason = p.Reason
		next.ClosedAt = p.ClosedAt

	default:
		return state, errorfamily.NewRejection(
			"legalparty.legal_party.unknown_event",
			"FoldLegalParty received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
