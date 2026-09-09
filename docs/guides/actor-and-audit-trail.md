# Actors and the Audit Trail

Every event, command, and query dispatched through cqrs-htmx can carry **who**
did it, **on whose behalf**, and **how it correlates** to the originating HTTP
request. This guide explains the actor model, what flows automatically, and
what you wire yourself.

## The ActorID model

`ActorID` (branded type from `go-cqrs-lite/id/v4`, aliased by every cqrs-htmx
module) identifies _the effective actor_ — the identity a request acts as. It
supports five kinds:

| Kind      | Constructor                                              | Typical source             |
| --------- | -------------------------------------------------------- | -------------------------- |
| `user`    | `id.NewUserActor(uid)` / `cqrshtmx.ActorIDFromUser(uid)` | Authenticated HTTP request |
| `bot`     | `id.NewBotActor(raw)`                                    | API-token traffic          |
| `system`  | `id.NewSystemActor(name)`                                | Schedulers, expiry sweeps  |
| `service` | `id.NewServiceActor(id)`                                 | Service-to-service calls   |
| `unknown` | zero value                                               | Unauthenticated context    |

Round-trip via `ActorID.String()` (`"user:01JX..."`) and
`cqrshtmx.ParseActorID(s)` (error-returning).

### Actor vs impersonator

Two context values distinguish _acting as_ from _really being_:

- `cqrshtmx.WithActorID(ctx, actor)` — the effective identity. In a direct
  login this is the authenticated user; in an impersonation it is the **target**
  user.
- `cqrshtmx.WithImpersonatorID(ctx, admin)` — the real authenticated admin.
  When both are set, every event records the full chain, which is what
  compliance queries need ("who actually changed this record?").

## What flows automatically

For HTTP requests served by a `cqrshtmx` App (and by the `setup` bundle):

1. **Auto-derivation.** `ContextEnrichmentMiddleware` extracts the
   authenticated user from the session and stores both `UserID` and
   `ActorID = ActorIDFromUser(user)` in the request context. You do nothing for
   the common case. If your own middleware already stored an `ActorID`
   (e.g. custom impersonation handling), it is left untouched.
2. **Events.** Wherever the journal appender builds event metadata
   (`EventOptionsFromContext`), the actor ID, impersonator ID, correlation ID,
   request ID, and context deadline propagate into the persisted event record.
3. **Audit log.** The usermgmt audit projection renders the actor string
   (`meta.ActorID.String()`) per entry, so admin actions read
   `user:01JX... removed tenant:01JY...` in the dashboard.

## What you wire yourself

### Commands and queries

Dispatched through a `cqrshtmx` App route (`app.Command(...)` /
`cqrshtmx.CommandTyped[...]`)? Then **nothing to do**: the dispatch pipeline
calls `CommandOptionsFromContext` and applies actor, user, correlation, and
request IDs onto the decoded command automatically (the same is true for
queries via `QueryOptionsFromContext`). Custom `command.Command`
implementations that are not `*command.BasicCommand` skip this enrichment —
set their metadata in the decoder.

For dispatches that bypass the HTTP pipeline (background jobs, direct
dispatcher calls), attach the options yourself:

```go
cmd := identitymodel.NewCreateTenantCmd(/* ... */)
if basic, ok := cmd.(*command.BasicCommand); ok {
    for _, opt := range cqrshtmx.CommandOptionsFromContext(ctx) {
        opt(basic)
    }
}
_, err := dispatcher.Dispatch(ctx, cmd)
```

### Non-HTTP actors

Background work should set an explicit actor so the audit trail does not
attribute jobs to "unknown":

```go
ctx = cqrshtmx.WithActorID(ctx, id.NewSystemActor("token-expiry-sweeper"))
```

For bot traffic, derive the actor from the authenticated token
(`id.NewBotActor(tokenID)`) in your auth middleware.

## Reading the trail

- **Audit dashboard** — the adminui dashboard and the `setup` bundle's admin
  panel render the audit log with actor strings.
- **Event metadata** — every journal event carries `metadata.ActorID`
  (prefixed string), plus correlation/request IDs; query it from the event
  store or stream it over the shared SSE feed.
- **samber-do-auditlog** — the `auditlog/v4` bridge (`WithAuditLog`) serves the
  same trail through a live SSE viewer; see that module's README.

## Checklist

- [ ] Authenticated HTTP: nothing to do (auto-derived).
- [ ] Impersonation: set `WithActorID` (target) + `WithImpersonatorID` (admin)
      in your middleware before dispatch.
- [ ] App-served commands/queries: nothing to do (auto-enriched).
- [ ] Out-of-band dispatch: apply `CommandOptionsFromContext` /
      `QueryOptionsFromContext` yourself.
- [ ] Background jobs: `WithActorID` with a system/service actor.
- [ ] Bot API traffic: `id.NewBotActor` in the token-auth middleware.
