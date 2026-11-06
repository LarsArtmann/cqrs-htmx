# ADR 0060: Session Lifecycle Audit Events (XS Scope)

**Status:** Accepted
**Date:** 2026-11-06
**Related:** [ADR 0012](0012-sql-session-store.md) (sessions stay ephemeral), [ADR 0043](0043-identity-model-extraction.md) (identity-model as domain source of truth), issue [#25](https://github.com/LarsArtmann/cqrs-htmx/issues/25), [actor-and-audit-trail guide](../guides/actor-and-audit-trail.md)

## Context

Sessions are deliberately ephemeral: rows live only until logout, lazy expiry, or
the SQL sweeper deletes them ([ADR 0012](0012-sql-session-store.md)). The audit
journal answers "who did X" for commands, but the impersonation origin
(`Impersonation{By, Reason, At}`) lived only inside the ephemeral session row —
after deletion, "which admin impersonated user X between T1 and T2" was
unanswerable from durable data. Issue #25 measured this as the one gap that
justifies session events at all, and prescribed the demand-gated shape: start
XS, keep `SessionStore` as-is, build a full `sessionlifecycle` module only when
a concrete consumer asks for session projections or concurrency limits.

## Decision

**Emit session lifecycle audit events; keep sessions ephemeral.**

The XS scope from issue #25, exactly:

1. **Two new events** in the published language:
   `SessionStarted` and `SessionEnded` (identity-model constants + payloads).
   `SessionStartedPayload` carries `ActorID`, `Impersonator`, `Reason`,
   `StartedAt`, `ExpiresAt`, and `TokenHash`. `SessionEndedPayload` carries
   `ActorID`, `Impersonator`, `TokenHash`, and a bounded-vocabulary `Cause`
   (`logout`, `expired`, `impersonation_ended`, `revoked`, `deleted`).
2. **One chokepoint decorator.** `NewService` wraps the configured
   `SessionStore` in `auditingSessionStore` (usermgmt/session_audit.go). Every
   create/delete that flows through the service — register, WebAuthn/OAuth2
   login, impersonation begin/end, logout, lazy expiry, user-deletion
   revocation — crosses it. The `SessionStore` interface is unchanged; custom
   stores get the audit journal for free.
3. **Auth-typed streams**, one fresh stream per event, following the login
   audit precedent (issue #23): the events are audit facts about auth, not
   state changes to a session aggregate.
4. **Best-effort emission.** An audit-store outage logs and continues; audit
   never gates authentication.
5. **Token safety.** The session token is never journaled. `SessionTokenHash`
   (SHA-256, base64url) is the correlation key linking a `SessionStarted` to
   its `SessionEnded`; a 256-bit random token is not recoverable from it.
6. **Cause attribution via context.** `Delete(ctx, token)` cannot say why a
   session ends, so aware call sites stamp the cause
   (`withSessionEndCause`); unstamped deletions journal the honest generic
   `deleted`.

## Consequences

- The issue's verification criterion holds: after sessions expire and rows are
  gone, the journal alone answers who impersonated user X, when, why, and when
  it ended (correlated by token hash). Pinned by
  `TestImpersonationForensicsSurviveSessionDeletion`.
- The event count grows 25 → 27 across identity-model, the usermgmt catalog,
  and the systemadapter decoder/schema/declarations.
- adminui's audit log renders the two new actions
  (`session_started`, `session_ended`), with the end cause as the entry detail.
- Load: two small events per session lifetime. A session-heavy deployment
  grows its journal by roughly one write per login and one per logout.

## Limitations

- **The sweeper is invisible.** `SQLSessionStore.EvictExpired` bulk-deletes
  without crossing the `SessionStore` interface the decorator wraps, so a
  session that expires unused and is swept before any lazy-expiry check gets no
  `SessionEnded`. The `SessionStarted` payload's `ExpiresAt` still bounds the
  window forensically. Closing this means widening the store interface or
  moving the sweeper behind the service — deferred until someone needs exact
  end events for swept sessions.
- **User-scoped revocation is one event.** `DeleteByUserID` journals a single
  `SessionEnded` (cause `revoked`, no token hash) rather than N per-session
  events; the affected user is the event's actor metadata.

## Demand gate (unchanged from issue #25)

The full `sessionlifecycle` module — projections over session events,
concurrency limits — is **not** built. Reopen when a concrete audit consumer
asks for a session projection ("list all live sessions", "how many concurrent
sessions did user X hold"), not before.

## Rejected alternatives

- **Event-source the session aggregate itself.** Reopens the settled T18
  decision (sessions external to the event store) and adds journal volume for
  state nobody queries.
- **Keep sessions in SQL forever + query rows.** Converts an ephemeral
  credential store into an unbounded one; retention becomes a new problem and
  the SQL row still is not the published audit language.
- **Emit only on impersonation.** Half the vocabulary; ordinary session
  start/end is exactly what makes the impersonation events interpretable.
