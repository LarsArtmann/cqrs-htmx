package usermgmt

import (
	"context"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// The session lifecycle audit journal (cqrs-htmx#25): sessions stay
// ephemeral by design (ADR 0012), but the SECURITY AUDIT facts of their
// lifecycle — especially impersonation start/end with By/Reason/At — become
// durable journal events, exactly like the login audit journal above. After
// the session rows are gone, the journal alone answers "which admin
// impersonated user X between T1 and T2".
//
// Mechanism: the service wraps its configured SessionStore in
// auditingSessionStore at NewService. Every create and delete that flows
// through the service (register, WebAuthn/OAuth2 login, impersonation
// begin/end, logout, lazy expiry, user-deletion revocation) crosses the one
// chokepoint and is journaled best-effort — an audit-store outage must
// never gate authentication. The SessionStore interface itself is unchanged.
//
// Cause attribution: Delete alone cannot say WHY a session ended, so the
// aware call sites stamp the cause into the context (withSessionEndCause).
// Unstamped deletions journal as the honest generic "deleted".

type sessionEndCauseKeyType struct{}

var sessionEndCauseKey sessionEndCauseKeyType //nolint:gochecknoglobals // sentinel type for context keys, standard Go pattern

// withSessionEndCause stamps why the session(s) being deleted end, for the
// auditing session store to journal. Call sites with a known cause wrap
// their Delete/DeleteByUserID context in it.
func withSessionEndCause(
	ctx context.Context, cause identitymodel.SessionEndCause,
) context.Context {
	return context.WithValue(ctx, sessionEndCauseKey, cause)
}

// sessionEndCauseFromContext returns the stamped cause, or the generic
// deleted fallback when the call site did not attribute one.
func sessionEndCauseFromContext(ctx context.Context) identitymodel.SessionEndCause {
	if cause, ok := ctx.Value(sessionEndCauseKey).(identitymodel.SessionEndCause); ok && cause != "" {
		return cause
	}

	return identitymodel.SessionEndDeleted
}

// auditingSessionStore decorates a SessionStore with lifecycle audit
// emission. It implements SessionStore by delegation: Find is a pure
// passthrough, Create journals SessionStarted after a successful inner
// Create, Delete snapshots the session first (the payload needs the actor
// and impersonation origin the token alone does not carry) and journals
// SessionEnded after a successful inner Delete, DeleteByUserID journals one
// user-scoped revocation event.
type auditingSessionStore struct {
	inner   SessionStore
	auditor authAuditor
}

func newAuditingSessionStore(inner SessionStore, auditor authAuditor) *auditingSessionStore {
	return &auditingSessionStore{inner: inner, auditor: auditor}
}

func (s *auditingSessionStore) Create(ctx context.Context, session *Session) error {
	if err := s.inner.Create(ctx, session); err != nil {
		return err //nolint:wrapcheck // decorator: the inner store's typed error passes through untouched
	}

	s.auditor.sessionStarted(ctx, session)

	return nil
}

func (s *auditingSessionStore) Find(ctx context.Context, token string) (*Session, error) {
	return s.inner.Find(ctx, token) //nolint:wrapcheck // decorator: pure passthrough, no audit concern
}

func (s *auditingSessionStore) Delete(ctx context.Context, token string) error {
	// Snapshot BEFORE the delete: after it, the session is gone. A failed
	// snapshot (unknown/expired-swept token) skips the journal entry — an
	// unattributable "a session ended" event has no forensic value, and
	// idempotent in-memory deletes of unknown tokens would otherwise emit
	// noise on every failed logout attempt.
	session, err := s.inner.Find(ctx, token)
	if err != nil {
		return s.inner.Delete(ctx, token) //nolint:wrapcheck // decorator: passthrough
	}

	if err := s.inner.Delete(ctx, token); err != nil {
		return err //nolint:wrapcheck // decorator: the inner store's typed error passes through untouched
	}

	s.auditor.sessionEnded(ctx, session, sessionEndCauseFromContext(ctx))

	return nil
}

func (s *auditingSessionStore) DeleteByUserID(ctx context.Context, userID UserID) error {
	if err := s.inner.DeleteByUserID(ctx, userID); err != nil {
		return err //nolint:wrapcheck // decorator: the inner store's typed error passes through untouched
	}

	s.auditor.sessionsRevokedForUser(ctx, userID)

	return nil
}
