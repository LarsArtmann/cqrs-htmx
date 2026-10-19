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

// appendAuthAuditEvent persists one auth audit event on a fresh Auth stream
// and publishes it to the bus. The append is BEST-EFFORT by design: auth
// audit complements authentication, it must never gate a successful login
// (an audit-store outage must not turn into an auth outage) — failures are
// logged loudly instead of propagated. Package-level because both the core
// Service and the OAuth2Service (a separate struct sharing dependencies)
// emit auth events.
func appendAuthAuditEvent(
	ctx context.Context,
	logger *slog.Logger,
	store event.Store,
	bus event.Bus,
	evtType event.Type,
	payload any,
	actor id.ActorID,
) {
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
		logger.Warn("usermgmt: auth audit event not persisted (build)",
			"event_type", string(evtType), "error", err)

		return
	}

	ref := id.StreamRef{ID: streamID, Type: aggregateTypeAuth}
	if err := store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		logger.Warn("usermgmt: auth audit event not persisted (save)",
			"event_type", string(evtType), "error", err)

		return
	}

	if bus != nil {
		if err := bus.Publish(ctx, evt); err != nil {
			// Persisted but not published — projections miss one audit
			// entry; the journal itself (the durable record) has it.
			logger.Warn("usermgmt: auth audit event not published",
				"event_type", string(evtType), "error", err)
		}
	}
}

// journalLoginSuccess records a successful authentication. Failures are
// logged, never propagated (see appendAuthAuditEvent).
func (s *Service) journalLoginSuccess(
	ctx context.Context,
	userID UserID,
	method identitymodel.AuthMethod,
	provider string,
) {
	s.journalAuthEvent(ctx, eventLoginSucceeded,
		identitymodel.LoginSucceededPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        method,
			Provider:      provider,
		},
		id.NewUserActor(userID))
}

// journalLoginFailure records a failed authentication for a KNOWN user.
// The reason MUST be a bounded [identitymodel.AuthFailureReason] class —
// never raw provider or ceremony error text.
func (s *Service) journalLoginFailure(
	ctx context.Context,
	userID UserID,
	method identitymodel.AuthMethod,
	reason identitymodel.AuthFailureReason,
	provider string,
) {
	s.journalAuthEvent(ctx, eventLoginFailed,
		identitymodel.LoginFailedPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        method,
			Reason:        reason,
			Provider:      provider,
		},
		id.NewUserActor(userID))
}

// journalRegistrationClosed records a registration attempt that was rejected
// because registration is closed. The probe carries no user identity (the
// would-be user does not exist); the event's actor stays zero.
func (s *Service) journalRegistrationClosed(ctx context.Context) {
	s.journalAuthEvent(ctx, eventRegistrationClosed,
		identitymodel.RegistrationClosedPayload{SchemaVersion: currentSchemaVersion},
		id.ActorID{})
}

func (s *Service) journalAuthEvent(ctx context.Context, evtType event.Type, payload any, actor id.ActorID) {
	appendAuthAuditEvent(ctx, s.logger, s.store, s.bus, evtType, payload, actor)
}

// The OAuth2 sub-service mirrors (a separate struct sharing the core
// Service's dependencies — its wrappers carry provider-qualified methods
// and skip failures with no attributable user).
func (o *OAuth2Service) journalLoginSuccess(ctx context.Context, userID UserID, provider string) {
	appendAuthAuditEvent(ctx, o.logger, o.store, o.bus, eventLoginSucceeded,
		identitymodel.LoginSucceededPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        identitymodel.AuthMethodOAuth2,
			Provider:      provider,
		},
		id.NewUserActor(userID))
}

func (o *OAuth2Service) journalLoginFailure(
	ctx context.Context,
	userID UserID,
	reason identitymodel.AuthFailureReason,
	provider string,
) {
	if userID.IsZero() {
		return // no attributable user — deliberately not journalled
	}

	appendAuthAuditEvent(ctx, o.logger, o.store, o.bus, eventLoginFailed,
		identitymodel.LoginFailedPayload{
			SchemaVersion: currentSchemaVersion,
			Method:        identitymodel.AuthMethodOAuth2,
			Reason:        reason,
			Provider:      provider,
		},
		id.NewUserActor(userID))
}

func (o *OAuth2Service) journalRegistrationClosed(ctx context.Context) {
	appendAuthAuditEvent(ctx, o.logger, o.store, o.bus, eventRegistrationClosed,
		identitymodel.RegistrationClosedPayload{SchemaVersion: currentSchemaVersion},
		id.ActorID{})
}
