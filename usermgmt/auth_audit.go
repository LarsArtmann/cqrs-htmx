package usermgmt

import (
	"context"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
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
// logged loudly instead of propagated.
func (s *Service) appendAuthAuditEvent(ctx context.Context, evtType event.Type, payload any, actor id.ActorID) {
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
		s.logger.Warn("usermgmt: auth audit event not persisted (build)",
			"event_type", string(evtType), "error", err)

		return
	}

	ref := id.StreamRef{ID: streamID, Type: aggregateTypeAuth}
	if err := s.store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		s.logger.Warn("usermgmt: auth audit event not persisted (save)",
			"event_type", string(evtType), "error", err)

		return
	}

	if s.bus != nil {
		if err := s.bus.Publish(ctx, evt); err != nil {
			// Persisted but not published — projections miss one audit
			// entry; the journal itself (the durable record) has it.
			s.logger.Warn("usermgmt: auth audit event not published",
				"event_type", string(evtType), "error", err)
		}
	}
}

// journalLoginSuccess records a successful authentication. Failures are
// logged, never propagated (see appendAuthAuditEvent).
func (s *Service) journalLoginSuccess(ctx context.Context, userID UserID, method identitymodel.AuthMethod, provider string) {
	s.appendAuthAuditEvent(ctx, eventLoginSucceeded, identitymodel.LoginSucceededPayload{
		SchemaVersion: currentSchemaVersion,
		Method:        method,
		Provider:      provider,
	}, id.NewUserActor(userID))
}

// journalLoginFailure records a failed authentication for a KNOWN user.
// The reason MUST be a bounded [identitymodel.AuthFailureReason] class —
// never raw provider or ceremony error text.
func (s *Service) journalLoginFailure(ctx context.Context, userID UserID, method identitymodel.AuthMethod, reason identitymodel.AuthFailureReason, provider string) {
	s.appendAuthAuditEvent(ctx, eventLoginFailed, identitymodel.LoginFailedPayload{
		SchemaVersion: currentSchemaVersion,
		Method:        method,
		Reason:        reason,
		Provider:      provider,
	}, id.NewUserActor(userID))
}

// journalRegistrationClosed records a registration attempt that was rejected
// because registration is closed. The probe carries no user identity (the
// would-be user does not exist); the event's actor stays zero.
func (s *Service) journalRegistrationClosed(ctx context.Context) {
	s.appendAuthAuditEvent(ctx, eventRegistrationClosed, identitymodel.RegistrationClosedPayload{
		SchemaVersion: currentSchemaVersion,
	}, id.ActorID{})
}
