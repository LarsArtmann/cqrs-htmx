package legalparty

import (
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// LegalPartyDecider returns the Decider for the LegalParty aggregate.
func LegalPartyDecider() decider.Decider[LegalPartyState] {
	return decider.Decider[LegalPartyState]{
		Initial: LegalPartyState{},
		Apply:   FoldLegalParty,
	}
}

// DecideOpenLegalParty returns the decide function for opening a legal
// party: the aggregate must not exist, the actor must be human/system/
// service (never a bot), and the profile must satisfy every invariant
// (identity facts, class-consistent identifiers, registry anchors).
func DecideOpenLegalParty(
	aggID id.StreamID, profile LegalPartyProfile, openedBy identitymodel.ActorID, openedAt time.Time,
) func(LegalPartyState, event.Version) ([]event.Event, error) {
	return func(state LegalPartyState, version event.Version) ([]event.Event, error) {
		if state.Opened {
			return nil, errorfamily.NewConflict(
				"legalparty.legal_party.already_exists",
				"legal party with this ID already exists",
			)
		}
		if err := rejectIllegalMaintainer(openedBy); err != nil {
			return nil, err
		}
		if err := profile.Validate(); err != nil {
			return nil, err
		}

		evt, err := event.New(
			EventLegalPartyOpened, aggID, AggregateTypeLegalParty, version.Increment(),
			LegalPartyOpenedPayload{
				SchemaVersion: CurrentSchemaVersion,
				Profile:       profile,
				OpenedAt:      openedAt.UTC(),
			},
			//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"legalparty.legal_party.event_failed",
				"create LegalPartyOpened event",
			)
		}

		return []event.Event{evt}, nil
	}
}

// DecideUpdateLegalPartyProfile returns the decide function for replacing a
// party's profile: the party must exist and be open, and optionally carries
// provenance-stamped registry facts.
func DecideUpdateLegalPartyProfile(
	aggID id.StreamID, profile LegalPartyProfile, facts *RegistryFacts,
	updatedBy identitymodel.ActorID, updatedAt time.Time,
) func(LegalPartyState, event.Version) ([]event.Event, error) {
	return func(state LegalPartyState, version event.Version) ([]event.Event, error) {
		if !state.Opened {
			return nil, ErrLegalPartyNotFound
		}
		if state.Closed {
			return nil, ErrLegalPartyClosed
		}
		if err := rejectIllegalMaintainer(updatedBy); err != nil {
			return nil, err
		}
		if err := profile.Validate(); err != nil {
			return nil, err
		}
		if facts != nil {
			if err := facts.Validate(); err != nil {
				return nil, err
			}
		}

		evt, err := event.New(
			EventLegalPartyProfileUpdated, aggID, AggregateTypeLegalParty, version.Increment(),
			LegalPartyProfileUpdatedPayload{
				SchemaVersion: CurrentSchemaVersion,
				Profile:       profile,
				RegistryFacts: facts,
				UpdatedAt:     updatedAt.UTC(),
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"legalparty.legal_party.event_failed",
				"create LegalPartyProfileUpdated event",
			)
		}

		return []event.Event{evt}, nil
	}
}

// DecideCloseLegalParty returns the decide function for closing a party:
// the party must exist and be open, the reason is required, and closure is
// terminal (a second close conflicts).
func DecideCloseLegalParty(
	aggID id.StreamID, reason string, closedBy identitymodel.ActorID, closedAt time.Time,
) func(LegalPartyState, event.Version) ([]event.Event, error) {
	return func(state LegalPartyState, version event.Version) ([]event.Event, error) {
		if err := requireCloseable(
			state.Opened, state.Closed, ErrLegalPartyNotFound, ErrLegalPartyClosed, closedBy, reason,
			"legalparty.legal_party.close_reason_required",
			"closing a legal party names a reason (e.g. ceased trading, merged, registry dissolved)",
		); err != nil {
			return nil, err
		}

		evt, err := event.New(
			EventLegalPartyClosed, aggID, AggregateTypeLegalParty, version.Increment(),
			LegalPartyClosedPayload{
				SchemaVersion: CurrentSchemaVersion,
				Reason:        reason,
				ClosedAt:      closedAt.UTC(),
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"legalparty.legal_party.event_failed",
				"create LegalPartyClosed event",
			)
		}

		return []event.Event{evt}, nil
	}
}

// requireCloseable runs the refusal ladder the two close deciders share
// (DecideCloseLegalParty and DecideCloseBillTo), in this order: the aggregate
// must exist (notFound), must not be closed already — closure is terminal
// (alreadyClosed) — the actor must be a legal maintainer (see
// rejectIllegalMaintainer), and the close must name a reason (a fresh
// rejection built from reasonCode and reasonMsg). It returns nil once every
// rung passes; which event the close emits stays each aggregate's decision.
func requireCloseable(
	opened, closed bool, notFound, alreadyClosed error, actor identitymodel.ActorID,
	reason, reasonCode, reasonMsg string,
) error {
	if !opened {
		return notFound
	}
	if closed {
		return alreadyClosed
	}
	if err := rejectIllegalMaintainer(actor); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return errorfamily.NewRejection(reasonCode, reasonMsg)
	}

	return nil
}

// rejectIllegalMaintainer enforces the actor axis: a legal party is opened,
// maintained, and closed by a human (ActorUser) or trusted infrastructure
// (ActorSystem, ActorService) actor — never by a bot, whose credentials are
// machine-scoped, and never by an unattributed actor.
func rejectIllegalMaintainer(actor identitymodel.ActorID) error {
	switch actor.Kind() {
	case identitymodel.ActorUser, identitymodel.ActorSystem, identitymodel.ActorService:
		return nil
	case identitymodel.ActorBot, id.ActorUnknown:
		return ErrBotActorForbidden
	default:
		return ErrBotActorForbidden
	}
}
