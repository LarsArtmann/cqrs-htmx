# Migration Guide: v2 → v3

> **Prerequisite:** You are on cqrs-htmx v2.x (module paths with `/v2` suffix).
> This guide walks you through upgrading to v3.x (`/v3` suffix).

**Target version:** v3.2.0 · **go-cqrs-lite:** v3.1.0 · **go-error-family:** v0.5.1

---

## Overview

v3 is a breaking release driven by the go-cqrs-lite v3.0.0 upgrade (module
path bump, projection rewrite) and a UserID type unification. The migration
touches import paths, event-sourcing setup, and type conversions.

### What Changed at a Glance

| Area                       | v2                     | v3                                 | Breaking? |
| -------------------------- | ---------------------- | ---------------------------------- | --------- |
| Module paths               | `/v2`                  | `/v3`                              | Yes       |
| go-cqrs-lite               | v2.6.0                 | v3.1.0                             | Yes       |
| usermgmt.UserID            | `brandid.ID` (string)  | `id.UserID` (ULID)                 | Yes       |
| Projection startup         | `projection.Runner`    | Manual replay + `bus.SubscribeAll` | Yes       |
| Corruption HTTP status     | 422                    | 500                                | Yes       |
| Infrastructure HTTP status | 500                    | 503                                | Yes       |
| SSEEvent.ID                | `string`               | `SSEEventID` branded type          | Yes       |
| go-error-family            | v0.4.0                 | v0.5.1                             | Internal  |
| Catalog module             | `cqrs-htmx/catalog/v3` | `go-cqrs-lite/catalog/v3` (v3.2.0) | Yes       |

---

## Step 1: Update Module Paths

Replace all `/v2` import paths with `/v3`:

```bash
# In every .go file and go.mod:
go get github.com/larsartmann/cqrs-htmx/v3@v3.2.0
go get github.com/larsartmann/cqrs-htmx/usermgmt/v3@v3.2.0
```

**Before:**

```go
import (
    cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
    "github.com/larsartmann/cqrs-htmx/v2/usermgmt"
)
```

**After:**

```go
import (
    cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
    "github.com/larsartmann/cqrs-htmx/v3/usermgmt"
)
```

Update your `go.mod` module path if you were using `/v2`:

```
module your-app // unchanged — your own module path doesn't change
```

Then bump go-cqrs-lite dependencies:

```bash
go get github.com/larsartmann/go-cqrs-lite@v3.1.0
go get github.com/larsartmann/go-cqrs-lite/event/v3@v3.1.0
go get github.com/larsartmann/go-cqrs-lite/command/v3@v3.1.0
# ... etc for all go-cqrs-lite modules you use
go mod tidy
```

---

## Step 2: Fix UserID Type Conversions

`usermgmt.UserID` changed from string-backed (`brandid.ID`) to ULID-backed
(`id.UserID`). This eliminates most manual conversions but changes `.Get()`.

### `.Get()` returns `ulid.ULID`, not `string`

**Before (v2):**

```go
userIDStr := user.ID.Get()        // string
usermgmt.NewUserID(userIDStr)     // string → UserID
```

**After (v3):**

```go
user.ID.Get()                     // ulid.ULID
user.ID.Get().String()            // string — use at SQL/Casbin/logging boundaries
usermgmt.NewUserID(s)             // accepts any string; ULIDs pass through,
                                  // non-ULIDs are deterministically hashed (backward compat)
usermgmt.MustParseUserID(s)       // strict ULID validation — use in production code
```

### Root ↔ usermgmt conversion

The conversion is now trivial since both use `id.UserID`:

```go
// v3: same underlying type — no conversion needed in most cases
rootUserID := cqrshtmx.UserIDFromContext(ctx)  // id.UserID
usermgmtUser, _ := service.GetUser(ctx, usermgmt.UserID(rootUserID))
```

### Zero-value UserID serializes as `null`

If your API serializes UserID values to JSON, note that a zero-value `UserID`
now serializes as `null` instead of `""`. Update any client-side null checks.

---

## Step 3: Update Projection Setup

go-cqrs-lite v3 deleted the `projection/` module (the `projection.Runner`).
Replace any custom projection startup with `StartProjections()`:

**Before (v2):**

```go
// You may have used projection.Runner directly (now deleted upstream)
runner := projection.NewRunner(bus, store, projections...)
runner.Start(ctx)
```

**After (v3):**

```go
// StartProjections replays all journal events synchronously, then subscribes
// to live events via bus.SubscribeAll, with id.EventID dedup.
setup.StartProjections(ctx)
```

If you use the one-call stack presets, projections are wired automatically:

```go
setup, err := usermgmt.NewSQLiteEventSourcedSetup(usermgmt.EventSourcedConfig{
    // ...
})
// Projections start automatically inside NewSQLiteEventSourcedSetup
```

### EventBus configuration

The watermill GoChannel backend uses `BlockPublishUntilSubscriberAck: true`
for read-your-writes consistency. If you configure the bus manually:

```go
bus, err := watermill.NewEventBus(watermill.GoChannelConfig{
    BlockPublishUntilSubscriberAck: true, // REQUIRED for projection consistency
})
```

---

## Step 4: Update Error Status Codes

The HTTP status mapping was reconciled with go-error-family upstream:

| Family         | v2 Status | v3 Status | Why                                                     |
| -------------- | --------- | --------- | ------------------------------------------------------- |
| Corruption     | 422       | **500**   | Data integrity breaks are server-side, not client input |
| Infrastructure | 500       | **503**   | Correct "service unavailable, retry later" semantic     |
| Panics         | 500       | 500       | Unchanged (explicit override)                           |

If clients check for 422 on corruption or 500 on infrastructure, update to 500 and 503.

---

## Step 5: Update SSEEvent.ID Type

`SSEEvent.ID` changed from `string` to the branded `SSEEventID` type:

**Before:**

```go
event := sse.SSEEvent{ID: "abc123", Data: "hello"}
id := stream.LastEventID() // string
```

**After:**

```go
event := sse.SSEEvent{ID: sse.NewSSEEventID("abc123"), Data: "hello"}
id := stream.LastEventID() // SSEEventID
idStr := id.String()       // string — at storage/HTTP boundaries
```

The `SSEEventStore.EventsAfter(string)` interface is unchanged — pass `.String()`.

---

## Step 6: Migrate Catalog Module (v3.2.0)

The `cqrs-htmx/catalog/v3` module is deleted. Use `go-cqrs-lite/catalog/v3` v3.2.0:

```bash
go get github.com/larsartmann/go-cqrs-lite/catalog/v3@v3.2.0
```

| Old (cqrs-htmx/catalog)                 | New (go-cqrs-lite/catalog)                       |
| --------------------------------------- | ------------------------------------------------ |
| `cataloghtmx.New(...)`                  | `simple.New(...)`                                |
| `cataloghtmx.Command[T](b, id)`         | `simple.Command[T](b, id)`                       |
| `cataloghtmx.D2Handler(cat)`            | `docserver.D2Handler(cat)`                       |
| `cataloghtmx.HealthCheckHandler(cat)`   | `docserver.HealthCheckHandler(cat)`              |
| `cataloghtmx.GenerateEventCatalog(...)` | `docserver.GenerateEventCatalog(...)`            |
| `cataloghtmx.OpenAPIHandler(cat)`       | `docserver.NewDocsServer(fn, cfg).OpenAPISpec()` |

See [ADR 0020](../adr/0020-merge-catalog-into-go-cqrs-lite.md) for rationale.

---

## Step 7: Update AuditLog Calls

`AuditLog.EntriesFor` now takes `id.AggregateID` instead of `string`:

**Before:**

```go
entries := auditLog.EntriesFor(user.ID.Get()) // string
```

**After:**

```go
entries := auditLog.EntriesFor(id.ParseAggregateID(user.ID.Get().String()))
```

---

## Checklist

- [ ] All import paths `/v2` → `/v3`
- [ ] go-cqrs-lite bumped to v3.1.0
- [ ] `.Get().String()` used at string boundaries (SQL, Casbin, logging)
- [ ] `MustParseUserID` used for strict ULID validation in production paths
- [ ] Projection startup uses `StartProjections()` (not deleted `projection.Runner`)
- [ ] EventBus uses `BlockPublishUntilSubscriberAck: true`
- [ ] Client status-code checks updated (Corruption→500, Infrastructure→503)
- [ ] SSEEvent.ID uses `NewSSEEventID` / `.String()` at boundaries
- [ ] Catalog imports migrated to `go-cqrs-lite/catalog/v3`
- [ ] AuditLog.EntriesFor takes `id.AggregateID`
- [ ] `go build ./...` and `go test ./... -race` pass

---

## Further Reading

- [ADR 0016: go-cqrs-lite v3.0.0 Migration](../adr/0016-go-cqrs-lite-v3-migration.md) — projection rewrite rationale
- [ADR 0017: Reconcile HTTP Status Mapping](../adr/0017-reconcile-http-status-mapping.md) — error family → status codes
- [ADR 0018: Unify UserID](../adr/0018-unify-userid.md) — UserID type unification
- [ADR 0020: Merge catalog into go-cqrs-lite](../adr/0020-merge-catalog-into-go-cqrs-lite.md) — catalog module deletion
- [CHANGELOG.md](../../CHANGELOG.md) — full release history
