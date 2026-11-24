package legalparty

import (
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BillToOpenedPayload opens a tenant's billing link.
type BillToOpenedPayload struct {
	SchemaVersion int          `json:"schema_version"`
	TenantID      string       `json:"tenant_id"`
	LegalPartyID  LegalPartyID `json:"legal_party_id"`
	OpenedAt      time.Time    `json:"opened_at"`
}

// BillToRedirectedPayload points bills at another legal party. The previous
// party stays in the journal — retention requires every redirect's before
// and after to be answerable.
type BillToRedirectedPayload struct {
	SchemaVersion int          `json:"schema_version"`
	LegalPartyID  LegalPartyID `json:"legal_party_id"`
	Reason        string       `json:"reason"`
	RedirectedAt  time.Time    `json:"redirected_at"`
}

// BillToClosedPayload closes a tenant's billing link, terminally.
type BillToClosedPayload struct {
	SchemaVersion int       `json:"schema_version"`
	Reason        string    `json:"reason"`
	ClosedAt      time.Time `json:"closed_at"`
}

// OpenBillToCmd opens a tenant's bill-to link. The stream is keyed by the
// tenant (DeriveBillToStreamID), so the command carries no tenant field —
// the ID is the aggregate.
type OpenBillToCmd struct {
	*command.BasicCommand
	partyID  LegalPartyID
	openedBy identitymodel.ActorID
}

// NewOpenBillToCmd builds the open command on a tenant's bill-to stream.
func NewOpenBillToCmd(
	tenantID string, partyID LegalPartyID, openedBy identitymodel.ActorID,
) *OpenBillToCmd {
	return &OpenBillToCmd{
		BasicCommand: mustCommand(CmdOpenBillTo, DeriveBillToStreamID(tenantID)),
		partyID:      partyID,
		openedBy:     openedBy,
	}
}

// LegalPartyID names the party bills go to.
func (c *OpenBillToCmd) LegalPartyID() LegalPartyID { return c.partyID }

// OpenedBy is the maintainer attribution (never a bot).
func (c *OpenBillToCmd) OpenedBy() identitymodel.ActorID { return c.openedBy }

// RedirectBillToCmd points an open bill-to link at another legal party.
type RedirectBillToCmd struct {
	*command.BasicCommand
	partyID      LegalPartyID
	reason       string
	redirectedBy identitymodel.ActorID
}

// NewRedirectBillToCmd builds the redirect command on a tenant's stream.
func NewRedirectBillToCmd(
	tenantID string, partyID LegalPartyID, reason string, redirectedBy identitymodel.ActorID,
) *RedirectBillToCmd {
	return &RedirectBillToCmd{
		BasicCommand: mustCommand(CmdRedirectBillTo, DeriveBillToStreamID(tenantID)),
		partyID:      partyID,
		reason:       reason,
		redirectedBy: redirectedBy,
	}
}

func (c *RedirectBillToCmd) LegalPartyID() LegalPartyID          { return c.partyID }
func (c *RedirectBillToCmd) Reason() string                      { return c.reason }
func (c *RedirectBillToCmd) RedirectedBy() identitymodel.ActorID { return c.redirectedBy }

// CloseBillToCmd closes a tenant's billing link, terminally.
type CloseBillToCmd struct {
	*command.BasicCommand
	reason   string
	closedBy identitymodel.ActorID
}

// NewCloseBillToCmd builds the close command on a tenant's stream.
func NewCloseBillToCmd(
	tenantID, reason string, closedBy identitymodel.ActorID,
) *CloseBillToCmd {
	return &CloseBillToCmd{
		BasicCommand: mustCommand(CmdCloseBillTo, DeriveBillToStreamID(tenantID)),
		reason:       reason,
		closedBy:     closedBy,
	}
}

func (c *CloseBillToCmd) Reason() string                  { return c.reason }
func (c *CloseBillToCmd) ClosedBy() identitymodel.ActorID { return c.closedBy }

// DeriveBillToStreamID builds the bill-to stream ID for a tenant: the
// tenant key IS the aggregate ID, so there is exactly one link per tenant
// by construction.
func DeriveBillToStreamID(tenantID string) id.StreamID {
	return id.DeriveStreamID(string(AggregateTypeBillTo), tenantID, "billing")
}

// BillToDecider returns the Decider for the BillTo aggregate.
func BillToDecider() decider.Decider[BillToState] {
	return decider.Decider[BillToState]{
		Initial: BillToState{},
		Apply:   FoldBillTo,
	}
}

// DecideOpenBillTo opens a tenant's billing link: the link must not exist,
// the actor must be human/system/service (never a bot), and the party must
// be named.
func DecideOpenBillTo(
	aggID id.StreamID, partyID LegalPartyID, openedBy identitymodel.ActorID, openedAt time.Time,
) func(BillToState, event.Version) ([]event.Event, error) {
	return func(state BillToState, version event.Version) ([]event.Event, error) {
		if state.Opened {
			return nil, errorfamily.NewConflict(
				"legalparty.bill_to.already_exists",
				"this tenant already has a bill-to link (redirect it instead)",
			)
		}
		if err := rejectIllegalMaintainer(openedBy); err != nil {
			return nil, err
		}
		if partyID.IsZero() {
			return nil, errorfamily.NewRejection(
				"legalparty.bill_to.party_required",
				"a bill-to link names the legal party that receives the invoices",
			)
		}
		evt, err := event.New(
			EventBillToOpened, aggID, AggregateTypeBillTo, version.Increment(),
			BillToOpenedPayload{
				SchemaVersion: CurrentSchemaVersion,
				TenantID:      tenantKeyOfBillTo(aggID),
				LegalPartyID:  partyID,
				OpenedAt:      openedAt.UTC(),
			},
			//cqrs-lint:ignore(A027) one codec per repository is intentional; global DefaultCodec would couple unrelated aggregates
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "legalparty.bill_to.event_failed", "create BillToOpened event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// DecideRedirectBillTo points an open link at another party. Redirecting to
// the party already named is a no-op (idempotent re-delivery), everything
// else requires a reason — a billing address change is an auditable act.
func DecideRedirectBillTo(
	aggID id.StreamID, partyID LegalPartyID, reason string,
	redirectedBy identitymodel.ActorID, redirectedAt time.Time,
) func(BillToState, event.Version) ([]event.Event, error) {
	return func(state BillToState, version event.Version) ([]event.Event, error) {
		if !state.Opened {
			return nil, ErrBillToNotFound
		}
		if state.Closed {
			return nil, ErrBillToClosed
		}
		if err := rejectIllegalMaintainer(redirectedBy); err != nil {
			return nil, err
		}
		if partyID.IsZero() {
			return nil, errorfamily.NewRejection(
				"legalparty.bill_to.party_required",
				"a redirect names the legal party that receives the invoices",
			)
		}
		if state.LegalPartyID == partyID {
			return nil, nil
		}
		if strings.TrimSpace(reason) == "" {
			return nil, errorfamily.NewRejection(
				"legalparty.bill_to.redirect_reason_required",
				"redirecting a tenant's bills names a reason (audit trail)",
			)
		}
		evt, err := event.New(
			EventBillToRedirected, aggID, AggregateTypeBillTo, version.Increment(),
			BillToRedirectedPayload{
				SchemaVersion: CurrentSchemaVersion,
				LegalPartyID:  partyID,
				Reason:        reason,
				RedirectedAt:  redirectedAt.UTC(),
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "legalparty.bill_to.event_failed", "create BillToRedirected event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// DecideCloseBillTo closes a tenant's billing link: it must be open, the
// reason is required, and closure is terminal.
func DecideCloseBillTo(
	aggID id.StreamID, reason string, closedBy identitymodel.ActorID, closedAt time.Time,
) func(BillToState, event.Version) ([]event.Event, error) {
	return func(state BillToState, version event.Version) ([]event.Event, error) {
		if !state.Opened {
			return nil, ErrBillToNotFound
		}
		if state.Closed {
			return nil, ErrBillToClosed
		}
		if err := rejectIllegalMaintainer(closedBy); err != nil {
			return nil, err
		}
		if strings.TrimSpace(reason) == "" {
			return nil, errorfamily.NewRejection(
				"legalparty.bill_to.close_reason_required",
				"closing a bill-to link names a reason (e.g. tenant churned, billing moved in-house)",
			)
		}
		evt, err := event.New(
			EventBillToClosed, aggID, AggregateTypeBillTo, version.Increment(),
			BillToClosedPayload{
				SchemaVersion: CurrentSchemaVersion,
				Reason:        reason,
				ClosedAt:      closedAt.UTC(),
			},
			event.WithCodec(codec.JSONCodec{}), event.WithSchemaVersion(CurrentSchemaVersion),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err, "legalparty.bill_to.event_failed", "create BillToClosed event",
			)
		}
		return []event.Event{evt}, nil
	}
}

// tenantKeyOfBillTo extracts the tenant key from a bill-to stream ID (the
// payload's TenantID field is informational; the stream is the truth).
func tenantKeyOfBillTo(aggID id.StreamID) string {
	return aggID.Get()
}
