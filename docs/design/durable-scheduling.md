# Design: Durable Scheduling for usermgmt Expiry

> **Status:** DESIGN ONLY — no production code changes. This document evaluates whether/how to wire go-cqrs-lite's `scheduling` module into usermgmt to make time-based expiry durable.
>
> **Created:** 2026-07-30 · **Source:** `docs/guides/leveraging-go-cqrs-lite.md` §3

---

## Problem Statement

usermgmt handles 6 time-based expiry mechanisms via in-process sweepers or lazy checks. These work for single-instance deployments but have two failure modes:

1. **Restart gap:** If the process restarts, all in-memory timers are lost. Lazy checks (on access) still work, but background sweepers miss their ticks. Sessions that expired during downtime remain in the session store until a user tries to use them.

2. **Multi-instance gap:** In a multi-instance deploy, each instance has its own in-memory sweepers. This is mostly safe (lazy expiry catches stale sessions on any instance), but background eviction is duplicated and there's no cross-instance coordination.

## Current Expiry Inventory

| Expiry Type | Default TTL | Lazy Check | Background Sweeper | Multi-Instance Safe? |
|---|---|---|---|---|
| Session (in-memory) | 24h | Yes (on auth) | No (manual) | No (in-memory only) |
| Session (SQL) | 24h | Yes (on auth) | Yes (per-instance ticker) | Yes (SQL is shared) |
| Email verification | 24h | Yes (on consume) | Yes (5 min) | Depends on store |
| Account lockout | 15 min | Yes (on check) | No (manual) | No (in-memory only) |
| WebAuthn session | 5 min | Yes (on get) | Yes (5 min) | Depends on store |
| Pending TOTP | 5 min | Yes (on consume) | Yes (1 min) | Depends on store |
| OAuth2 state | 10 min | Yes (on consume) | Yes (5 min) | Depends on store |

**Key observation:** Every expiry mechanism already has a **lazy check** that fires on access. This means correctness is NOT compromised by restart gaps — expired entries are always rejected when accessed. The background sweepers are an **optimization** (clean up stale entries proactively) rather than a correctness requirement.

## Assessment: Is `scheduling.TimerStore` Needed?

### For SQL-backed deployments: NO

SQL session stores (`sql_session_store.go`) already support `EvictExpired(ctx)` with a shared SQL database. Multiple instances can run the same sweeper against the shared DB. The SQL sweeper is idempotent (DELETE WHERE expires_at < NOW). No durable timer needed.

### For in-memory deployments: Partially

In-memory stores cannot be made durable regardless of the scheduling mechanism — the data itself is lost on restart. `scheduling.TimerStore` would only fire the expiry callback, but the data to expire is already gone.

### For mixed deployments (in-memory auth state + SQL event store): MAYBE

Some expiry types (WebAuthn sessions, pending TOTP, OAuth2 state) are inherently short-lived (5-10 min) and stored in-memory even in production. A durable timer could fire a "cleanup" callback after restart, but since the in-memory data is already lost, the callback has nothing to clean up. The lazy check on access handles correctness.

## Proposed Design (if pursued)

### API surface

```go
type ServiceConfig struct {
    // TimerStore enables durable expiry callbacks. When nil, in-process
    // sweepers are used (current behavior). When set, expiry events are
    // persisted to the timer store and survive restarts.
    TimerStore scheduling.TimerStore[ExpiryEvent] // optional
}

type ExpiryEvent struct {
    Type     string    // "session", "verification", "lockout", etc.
    Key      string    // session token, email, etc.
    ExpireAt time.Time
}
```

### Wiring

```go
scheduler := scheduling.New(timerStore, func(ctx context.Context, t scheduling.Timer[ExpiryEvent]) error {
    switch t.Payload.Type {
    case "session":
        svc.revokeSession(ctx, t.Payload.Key)
    case "verification":
        svc.deleteVerificationToken(ctx, t.Payload.Key)
    // ...
    }
    return nil
}, scheduling.WithPollInterval(5*time.Second))
```

### Risk assessment (VERSCHLIMMBESSERUNG traps)

1. **Behavior change:** Currently, an expired session is lazily rejected at access time. A durable timer would proactively delete it — functionally equivalent but with different timing semantics.
2. **Migration path:** Existing consumers would need to opt-in by providing a `TimerStore`. Default (nil) preserves current behavior.
3. **Double-eviction risk:** If both the lazy check AND the timer fire, the second is a no-op. Safe but wasteful.
4. **Short-lived data:** For 5-minute TTLs (WebAuthn, TOTP), the overhead of persisting to a timer store may exceed the benefit.

## Recommendation

**DO NOT IMPLEMENT NOW.** The current lazy-check + background-sweeper design is correct for all deployment modes. The SQL store already provides multi-instance safety for the longest-TTL item (sessions, 24h). Short-lived tokens (5-10 min) are not worth the complexity of durable timers.

Re-evaluate if:
- A consumer needs cross-instance lockout coordination (currently in-memory only)
- Session revocation must be immediate (not lazy-on-next-access)
- A consumer runs a single-instance deploy and needs guaranteed post-restart cleanup
