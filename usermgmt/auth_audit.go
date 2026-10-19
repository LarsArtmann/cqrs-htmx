package usermgmt

import (
	"context"
	"log/slog"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The auth audit journal (cqrs-htmx#23): successful and failed logins and
// registration-closed probes become durable journal events so the identity
// journal answers "who logged in, when, and how often was registration
// probed after closing" long after the ephemeral session rows are gone.
//
// Stream placement: every auth event gets its OWN Auth-typed stream keyed by
// a fresh ULID. This keeps the audit append-only log OUT of the user
// aggregate — a user stream must stay small (every command folds it), and
// failed-login journaling must never give an attacker a way to grow a
// victim's stream. The acting user is carried by the payload and the event's
// actor metadata instead.
//
// Known-user exclusions: login failures for UNKNOWN identifiers are NOT
// journaled (no stream exists and no user can be attributed; journalling
// them would hand unauthenticated traffic an unbounded journal-write
// vector). They remain visible through request logs and rate-limit counters.

// authAuditor journals auth audit events as a focused collaborator, not as
// *Service methods: the ADR-0038 method ceiling forces new concerns onto
// their own types, and both the core Service and the OAuth2Service (a
// separate struct sharing dependencies) emit auth events.
type authAuditor struct {
	logger *slog.Logger
	store  event.Store
	bus    event.Bus
}

// append persists one auth audit event on a fresh Auth stream and publishes
// it to the bus. The append is BEST-EFFORT by design: auth audit complements
// authentication, it must never gate a successful login (an audit-store
// outage must not turn into an auth outage) — failures are logged loudly
// instead of propagated.
func (a authAuditor) append(ctx context.Context, evtType event.Type, payload any, actor id.ActorID) {
	streamID := id.NewStreamID()

	opts := []event.Option{event.WithCodec(codec.JSONCodec{})}

	if !actor.IsZero() {
		opts = append(opts, event.WithActor(actor))
	}

	evt, err := event.New(
		evtType,
		streamID,
		aggregateTypeAuth,
		event.Version(0).Increment(),
		payload,
		opts...,
	)
	if err != nil {
		a.logger.Warn("usermgmt: auth audit event not persisted (build)",
			"event_type", string(evtType), "error", err)

		return
	}

	ref := id.StreamRef{ID: streamID, Type: aggregateTypeAuth}
	if err := a.store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		a.logger.Warn("usermgmt: auth audit event not persisted (save)",
			"event_type", string(evtType), "error", err)

		return
	}

	if a.bus != nil {
		if err := a.bus.Publish(ctx, evt); err != nil {
			// Persisted but not published — projections miss one audit
			// entry; the journal itself (the durable record) has it.
			a.logger.Warn("usermgmt: auth audit event not published",
				"event_type", string(evtType), "error", err)
		}
	}
}

// loginSuccess records a successful authentication. Failures are logged,
// never propagated (see append).
func (a authAuditor) loginSuccess(ctx context.Context, userID UserID, method identitymodel.AuthMethod) {
	a.append(ctx, eventLoginSucceeded,
		identitymodel.LoginSucceededPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        method,
		},
		id.NewUserActor(userID))
}

// loginFailure records a failed authentication for a KNOWN user. The reason
// MUST be a bounded [identitymodel.AuthFailureReason] class — never raw
// provider or ceremony error text.
func (a authAuditor) loginFailure(
	ctx context.Context,
	userID UserID,
	method identitymodel.AuthMethod,
	reason identitymodel.AuthFailureReason,
) {
	a.append(ctx, eventLoginFailed,
		identitymodel.LoginFailedPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        method,
			Reason:        reason,
		},
		id.NewUserActor(userID))
}

// registrationClosed records a registration attempt that was rejected
// because registration is closed. The probe carries no user identity (the
// would-be user does not exist); the event's actor stays zero.
func (a authAuditor) registrationClosed(ctx context.Context) {
	a.append(ctx, eventRegistrationClosed,
		identitymodel.RegistrationClosedPayload{SchemaVersion: currentSchemaVersion},
		id.ActorID{})
}

// oauth2LoginSuccess is loginSuccess with the OAuth2 provider name. OAuth2
// login FAILURES have no attributable user today (state/token/email checks
// run before any user is matched), so there is no failure variant: adding
// one is a single append call when that changes.
func (a authAuditor) oauth2LoginSuccess(ctx context.Context, userID UserID, provider string) {
	a.append(ctx, eventLoginSucceeded,
		identitymodel.LoginSucceededPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        identitymodel.AuthMethodOAuth2,
			Provider:      provider,
		},
		id.NewUserActor(userID))
}
