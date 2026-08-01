# Data-Mesh Interchange Interface for cqrs-htmx

**Date:** 2026-07-25
**Status:** Proposal — revised after deep source investigation
**Scope:** Root module + relationship to go-cqrs-lite `catalog/v4` and `transport/http/v4`

---

## TL;DR

**We don't need to build a data-mesh interchange interface. go-cqrs-lite already has one — `catalog/v4`.** It ships `DataProduct`, `DataContract`, `Channel` (AsyncAPI), `Message` (with producers/consumers/schemas/operations), full AsyncAPI 3.0 + OpenAPI + eventcatalog.dev + D2 exporters, and an embedded doc server. cqrs-htmx's root module **doesn't import it** — instead it ships a hand-rolled `EventCatalog` with 3 fields and its own `openapi/` sub-package, both of which are impoverished subsets of what the upstream provides.

The real work is **adoption and exposure**, not invention. The only genuinely missing code in the entire ecosystem is a **Channel↔Bus binding** (connecting a `catalog.Channel` documentation entry to a runtime `event.Bus` subscription) and an optional **CloudEvents `payloadTransform`** (using `transport/http/v4.SSEBroker`'s existing hook).

---

## Table of Contents

- [1. What Happened (The Embarrassing Truth)](#1-what-happened-the-embarrassing-truth)
- [2. What go-cqrs-lite Already Provides](#2-what-go-cqrs-lite-already-provides)
- [3. What cqrs-htmx Ships Instead (the Re-invention)](#3-what-cqrs-htmx-ships-instead-the-re-invention)
- [4. The Three Genuinely Missing Pieces](#4-the-three-genuinely-missing-pieces)
- [5. Five Approaches](#5-five-approaches)
- [6. Recommendation](#6-recommendation)
- [7. Phased Delivery Plan](#7-phased-delivery-plan)
- [Appendix A: Verified Source References](#appendix-a-verified-source-references)

---

## 1. What Happened (The Embarrassing Truth)

The original version of this document proposed building `DataProduct`, `CloudEvent`, `EventStreamHandler`, `asyncapi/`, and `DataProductRegistry` from scratch in cqrs-htmx. That was wrong. A deep read of the source revealed that **go-cqrs-lite's `catalog/v4` module already provides nearly all of this** — and cqrs-htmx already has a working demo (`examples/catalog-demo/`) and integration tests (`integration_test/catalog_test.go`) that use it.

The problem is not "we need to build an interchange." The problem is **cqrs-htmx's root module re-invented a poor subset of what its own dependency already ships, and doesn't expose the rich model.**

---

## 2. What go-cqrs-lite Already Provides

All verified by reading the source at `/home/lars/go/pkg/mod/github.com/larsartmann/go-cqrs-lite/catalog/v4@v4.1.0/` and the local checkout at `/home/lars/projects/go-cqrs-lite/catalog/`.

### 2.1 Data Product + Data Contract (the mesh envelope)

`catalog/types_resources.go:37-95`:

```go
// DataProduct represents a data product in a data mesh — a curated, owned dataset.
type DataProduct struct {
    ID      DataProductID       `json:"id"`
    Name    Name                `json:"name"`
    Version Version             `json:"version"`
    Summary Summary             `json:"summary,omitempty"`
    Inputs  []Ref               `json:"inputs,omitempty"`
    Outputs []DataProductOutput `json:"outputs,omitempty"`
    Hidden  bool                `json:"hidden,omitempty"`
    Owners  []string            `json:"owners,omitempty"`
    Badges  []Badge             `json:"badges,omitempty"`
}

type DataContract struct {
    Path string `json:"path"`
    Name Name   `json:"name,omitempty"`
    Type string `json:"type,omitempty"`
}

type DataProductOutput struct {
    Ref
    Contract *DataContract `json:"contract,omitempty"`
}
```

This is the DPDS/Bitol data-product envelope. `Domain` has a `DataProducts []DataProductID` field (`types.go:190`). `Catalog` has `DataProducts []DataProduct` (`types.go:219`). `FlowStep` can reference a `DataProduct` (`types_resources.go:145`).

### 2.2 Channel (AsyncAPI channels with protocols and delivery guarantees)

`catalog/types.go:193-206`:

```go
type Channel struct {
    ID                ChannelID               `json:"id"`
    Name              Name                    `json:"name"`
    Version           Version                 `json:"version"`
    Summary           Summary                 `json:"summary,omitempty"`
    Address           Address                 `json:"address,omitempty"`
    Protocols         []Protocol              `json:"protocols,omitempty"`     // "http", "kafka", "nats"
    Messages          []MessageID             `json:"messages,omitempty"`
    DeliveryGuarantee DeliveryGuarantee       `json:"deliveryGuarantee,omitempty"` // "at-least-once"
    Parameters        map[string]ChannelParam `json:"parameters,omitempty"`
    Routes            []ChannelRoute          `json:"routes,omitempty"`
    Owners            []string                `json:"owners,omitempty"`
    Badges            []Badge                 `json:"badges,omitempty"`
}
```

Channel options (`channel_config.go`): `ChannelAddress`, `ChannelProtocols`, `ChannelMessages`, `ChannelDeliveryGuarantee`, `ChannelParameters`, `ChannelRoutes`, `ChannelOwners`.

### 2.3 Message (rich event/command/query model)

`catalog/types.go:126-149`:

```go
type Message struct {
    Kind        MessageKind       `json:"kind"` // command | event | query
    ID          MessageID         `json:"id"`
    Name        Name              `json:"name"`
    Version     Version           `json:"version"`
    Summary     Summary           `json:"summary,omitempty"`
    Schema      *Schema           `json:"schema,omitempty"`        // auto-derived from Go types
    Schemas     []SchemaPointer   `json:"schemas,omitempty"`
    Direction   Direction         `json:"direction"`               // sends | receives
    Examples    []jsontext.Value  `json:"examples,omitempty"`
    Owners      []string          `json:"owners,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
    Deprecated  bool              `json:"deprecated,omitempty"`
    Deprecation *DeprecationInfo  `json:"deprecation,omitempty"`
    Channels    []ChannelID       `json:"channels,omitempty"`
    Changelog   []Change          `json:"changelog,omitempty"`
    Producers   []ServiceID       `json:"producers,omitempty"`
    Consumers   []ServiceID       `json:"consumers,omitempty"`
    Operation   *Operation        `json:"operation,omitempty"`     // HTTP method + path + status codes
    Responses   []ResponseSpec    `json:"responses,omitempty"`
    Badges      []Badge           `json:"badges,omitempty"`
    Repository  *Repository       `json:"repository,omitempty"`
    Security    []string          `json:"security,omitempty"`
}
```

Generics-based registration (`message_config.go:206-222`): `Command[T]()`, `Event[T]()`, `Query[T]()` — auto-derive JSON Schema from Go struct types via reflection.

### 2.4 Exporters (the contract formats)

| Exporter         | Package                   | Output                                      | cqrs-htmx equivalent                              |
| ---------------- | ------------------------- | ------------------------------------------- | ------------------------------------------------- |
| **AsyncAPI 3.0** | `catalog/v4/asyncapi`     | `asyncapi.Document` (JSON/YAML)             | **None** (proposed as "Option 3" in original doc) |
| **OpenAPI 3.0**  | `catalog/v4/openapi`      | OpenAPI spec (JSON/YAML)                    | Hand-rolled `openapi/` sub-package (~400 LOC)     |
| **D2 diagrams**  | `catalog/v4/d2`           | Architecture/service diagrams               | **None**                                          |
| **EventCatalog** | `catalog/v4/eventcatalog` | eventcatalog.dev MDX file tree              | **None**                                          |
| **Doc server**   | `catalog/v4/docserver`    | Embedded HTML UIs (Scalar, AsyncAPI Studio) | **None**                                          |

### 2.5 The `simple` fluent builder

`catalog/v4/simple/builder.go` — the high-level API used in cqrs-htmx's own catalog-demo:

```go
b := simple.New("Order Service", "1.0.0",
    simple.WithServiceSummary("Example service"),
)
simple.Command[CreateOrderCommand](b, "create-order",
    simple.WithOperation("POST", "/orders"),
    catalog.WithSummary("Place a new order"),
)
simple.Event[OrderCreatedEvent](b, "order.created", catalog.Sends,
    catalog.WithSummary("An order was placed"),
)
cat := b.Build()
```

### 2.6 The `docserver` — serve everything from one handler

`catalog/v4/docserver/docserver.go` — used in cqrs-htmx's own catalog-demo and integration tests:

```go
ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
    ServiceName: "Order Service",
    Version:     "1.0.0",
})
mux.Handle("/openapi.json", ds.OpenAPISpec())
mux.Handle("/asyncapi.json", ds.AsyncAPISpec())
mux.Handle("/diagram.d2", docserver.D2Handler(cat))
mux.Handle("/health", docserver.HealthCheckHandler(cat))
```

### 2.7 Production-grade transport — `transport/http/v4.SSEBroker`

`transport/http/v4/sse.go:27-41`:

```go
type SSEBroker struct {
    mu               sync.RWMutex
    clients          map[SSEClientID]chan event.Event
    handler          event.Handler
    cancel           context.CancelFunc
    journal          event.SeekableJournal    // optional: Last-Event-ID replay
    replayLimit      int
    replayTimeout    time.Duration
    replayMetrics    *ReplayMetrics
    replayByteBudget int
    dedupRingCap     int
    retryInterval    time.Duration
    eventFilter      func(event.Type) bool    // filtering hook
    payloadTransform func(event.Event) []byte  // CloudEvents transform hook
}
```

Features cqrs-htmx's `Broadcaster`/`JournalSSEStore` lack: event filtering, payload transformation, dedup ring, byte-budget replay limits, replay metrics, replay timeout, OpenTelemetry instrumentation, configurable heartbeat/retry.

The `transport/http/v4/doc.go` explicitly states the design vision: _"Future transports (gRPC, NATS, Redis) will live as sibling modules under `transport/`."_ `transport/grpc/v4` is already in `go.work` local replaces.

### 2.8 What's verified absent

Grep across **all** of go-cqrs-lite for `CloudEvent`, `interchange`, `dataport`, `federation` returned **zero Go matches**. The only mesh-adjacent concept is `DataProduct` (documented as "a data product in a data mesh"). There is no code binding a `catalog.Channel` to a runtime `event.Bus` topic or `StreamingJournal` — the documentation model and the runtime transport are fully decoupled.

---

## 3. What cqrs-htmx Ships Instead (the Re-invention)

| Concern                     | go-cqrs-lite `catalog/v4`                                                                             | cqrs-htmx root                                                                             | Verdict                   |
| --------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------- |
| Event metadata              | `Message` with 20+ fields (Kind, Schema, Producers, Consumers, Operation, Changelog, Deprecated, ...) | `EventMetadata` with 5 fields (Type, Aggregate, SchemaVersion, Description, PayloadFields) | **Subset**                |
| Schema derivation           | Auto from Go types via reflection (`SchemaFromType[T]()`)                                             | Manual `[]PayloadField` lists                                                              | **Manual + error-prone**  |
| Data product                | `DataProduct` + `DataContract` + `DataProductOutput`                                                  | **Does not exist**                                                                         | **Missing**               |
| Channel / transport binding | `Channel` with Address, Protocols, DeliveryGuarantee, Routes                                          | **Does not exist**                                                                         | **Missing**               |
| AsyncAPI export             | `asyncapi.Exporter` full AsyncAPI 3.0                                                                 | **Does not exist**                                                                         | **Missing**               |
| OpenAPI export              | `openapi.Exporter` full OpenAPI 3.0                                                                   | Hand-rolled `openapi/` sub-package (~400 LOC)                                              | **Re-invented**           |
| D2 diagrams                 | `d2.Exporter`                                                                                         | **Does not exist**                                                                         | **Missing**               |
| eventcatalog.dev            | `eventcatalog.Exporter` full MDX file tree                                                            | **Does not exist**                                                                         | **Missing**               |
| Doc server (HTML UIs)       | `docserver.DocsServer` (Scalar, AsyncAPI Studio)                                                      | **Does not exist**                                                                         | **Missing**               |
| HTTP serving                | `docserver.NewDocsServer(...)` — all formats from one handler                                         | `EventCatalogHandler` (JSON array only) + `OpenAPISpecHandler` (separate)                  | **Re-invented**           |
| Realtime transport          | `SSEBroker` (filtering, dedup, metrics, OTel, byte budget)                                            | `Broadcaster` + `JournalSSEStore` (simple fan-out + replay)                                | **Re-invented (simpler)** |
| Streaming reads             | `StreamingJournal.ReadStreamFrom` returns `EventIterator` (non-materializing)                         | `SeekableJournal.ReadFrom` returns `[]Event` (materializes full slice)                     | **Not used**              |

**Root cause:** cqrs-htmx's root `go.mod` depends on `event/v4`, `command/v4`, `query/v4`, `id/v4` — but **not** `catalog/v4` or `transport/http/v4`. The `catalog/v4` import only appears in `integration_test/go.mod` and `examples/catalog-demo/`. The library ships its own subset instead of exposing the upstream.

---

## 4. The Three Genuinely Missing Pieces

After accounting for everything the upstream already provides, only three things are actually absent from the combined ecosystem:

### Gap 1: Channel-to-Bus binding (the documentation-runtime disconnect)

`catalog.Channel` describes _what_ a channel is (address, protocol, messages, delivery guarantee). `event.Bus` / `StreamingJournal` provides the _runtime_ transport. **No code connects them.** A consumer who defines a `Channel` in their catalog has no way to say "this channel is backed by this `event.Bus` subscription" or "this channel reads from this `StreamingJournal`."

This is the **interchange binding** — the one piece that would make the catalog model _actionable_ rather than purely documentary.

### Gap 2: CloudEvents envelope

Neither go-cqrs-lite nor cqrs-htmx produces CloudEvents-formatted messages. The `event.Event` type carries `ID()`, `Type()`, `StreamID()`, `OccurredAt()`, `Payload()` — enough to map to CloudEvents attributes, but no code does the mapping.

**However:** `transport/http/v4.SSEBroker` already has `payloadTransform func(event.Event) []byte` — a per-event transform hook. A CloudEvents wrapper is a single function piped through this existing hook. No new type needed in the transport layer.

### Gap 3: Pull-based machine transport

The interchange currently assumes SSE (browser-friendly push). A data-mesh consumer often wants **pull-based HTTP** (`GET /events?after=<id>` then `[]CloudEvent` JSON or NDJSON) — for batch consumers, cron jobs, non-streaming clients. No existing code provides this variant. (`SSEBroker` is push-only; `SeekableJournal.ReadFrom` returns a slice but isn't wrapped in an HTTP handler; `JournalSSEStore` emits SSE wire format.)

`transport/grpc/v4` is in local replaces (exists or in development). NATS/Kafka federation is via `watermill.WithBackend`. These are upstream concerns, not cqrs-htmx gaps.

---

## 5. Five Approaches

### Approach A: Full adoption — deprecate EventCatalog + openapi/, depend on catalog/v4

Add `catalog/v4` as a root dependency. Deprecate `EventCatalog`, `EventCatalogHandler`, and the `openapi/` sub-package. Expose `docserver` handlers directly. Migrate `usermgmt.DefaultEventCatalog()` to return a `*catalog.Catalog` via `simple.New()`.

**Pros:** One source of truth. Rich model. All exporters. No duplicated types.
**Cons:** Breaking change. Forces every root consumer to pull in `catalog/v4` even if they don't need docs. Violates the "never enforce defaults" principle for consumers who want zero doc overhead.

### Approach B: Opt-in catalog adapter — bridge, don't replace

Keep `EventCatalog` as the zero-dependency lightweight path. Add `EventCatalog.ToCatalog()` returning `*catalog.Catalog`. Consumers who want AsyncAPI/DataProduct/eventcatalog.dev call the adapter. The root `go.mod` does not depend on `catalog/v4` — the adapter lives in a separate optional package.

**Pros:** Non-breaking. Zero overhead for consumers who skip docs. Clean upgrade path.
**Cons:** Two type systems. The `EventMetadata` to `catalog.Message` mapping is lossy (no Producers/Consumers/Operation/Schema in EventMetadata). Adapter produces hollow entries.

### Approach C: Expose catalog/v4 as the recommended path, deprecate EventCatalog gradually

Don't add `catalog/v4` to the root `go.mod` (keep root lean). Instead:

1. Document `catalog/v4` + `simple.New()` + `docserver` as the **recommended** way to build data-mesh contracts (the catalog-demo already shows this).
2. Add `// Deprecated:` markers on `EventCatalog` / `EventCatalogHandler` / `openapi.Spec` / `OpenAPISpecHandler` pointing to `catalog/v4`.
3. Provide `usermgmt.DefaultCatalog()` returning `*catalog.Catalog` (built via `simple.New()` from the existing identity event types) alongside the deprecated `DefaultEventCatalog()`.
4. Add a `docs/guides/data-mesh-interchange.md` guide showing the full composition.

**Pros:** Lean root (no new dep). Clear direction without forcing migration. The catalog-demo and integration_test already validate the pattern. Consumer keeps full control.
**Cons:** Two APIs coexist during the deprecation window. Consumer confusion about which to use.

### Approach D: Channel binding + pull-based EventStreamHandler (the genuinely new code)

Independent of the catalog adoption question: build the three missing runtime pieces.

1. **`ChannelBinding`** — a thin connector binding a `catalog.Channel` to a runtime `event.Bus` subscription or `StreamingJournal` read. Makes the catalog model actionable: "this Channel's messages come from this Bus subscription."

2. **`EventStreamHandler`** — a pull-based HTTP handler (`GET /events?after=<eventID>&limit=N` returning JSON array or NDJSON) backed by `event.SeekableJournal` or `StreamingJournal`. Uses `StreamingJournal.ReadStreamFrom` (non-materializing iterator) for high-volume streams. Optional CloudEvents framing.

3. **`CloudEventFromEvent()`** — a pure function mapping `event.Event` to a CloudEvents 1.0 envelope (7 required fields). Pipable through `SSEBroker.payloadTransform` or usable standalone in the `EventStreamHandler`.

**Pros:** Closes the three genuine gaps. Works with or without catalog adoption.
**Cons:** The `EventStreamHandler` partially overlaps with `transport/http/v4.SSEBroker` (which does push-based SSE). Need to decide: adopt SSEBroker, build pull-based alongside, or both.

### Approach E: Documentation only — no code changes

Write `docs/guides/data-mesh-interchange.md` showing how to compose `catalog/v4` + `transport/http/v4.SSEBroker` + `watermill.WithBackend` in a cqrs-htmx app today, with zero library changes. The catalog-demo already proves this works.

**Pros:** Zero risk. Immediate. Honest about what already exists.
**Cons:** Doesn't fix the root problem (consumers still reach for `EventCatalog` first because it's what the library exports). Doesn't close the Channel-to-Bus binding gap.

---

## 6. Recommendation

### Ranked: **Approach C + D combined**

| Criterion               | A (full adopt) | B (adapter) |     C (gradual deprecate)      | D (missing pieces) | E (docs only) |
| ----------------------- | :------------: | :---------: | :----------------------------: | :----------------: | :-----------: |
| Closes real gaps        |    Partial     | No (lossy)  |      Partial (direction)       |  **Yes (all 3)**   |      No       |
| Breaking change         |    **Yes**     |     No      |               No               |         No         |      No       |
| Root stays lean         |  No (new dep)  |     Yes     |            **Yes**             |      **Yes**       |    **Yes**    |
| New code in root        |   Migration    |   Adapter   | `DefaultCatalog()` in usermgmt |      ~180 LOC      |       0       |
| Honest about upstream   |      Yes       |   Partial   |            **Yes**             |      **Yes**       |      Yes      |
| Consumer can ignore     |       No       |     Yes     |            **Yes**             |      **Yes**       |      Yes      |
| Proven by existing demo |      Yes       |     No      |     **Yes** (catalog-demo)     |      No (new)      |      Yes      |

#### Why not Approach A

Forces `catalog/v4` as a hard dependency on every root consumer. The library principle says "never enforce defaults." Many consumers want CQRS+HTMX dispatch without documentation overhead — they shouldn't pay the `catalog/v4` import cost.

#### Why not Approach B

The `EventMetadata` to `catalog.Message` mapping is lossy. `EventMetadata` has no `Producers`, `Consumers`, `Operation`, or auto-derived `Schema`. An adapter produces hollow entries — you get the AsyncAPI format but not the richness that makes it useful. Better to register directly via `simple.Event[T]()` which auto-derives schema from Go types.

#### Why C + D combined

1. **C gives the direction** without forcing migration: deprecated markers + `DefaultCatalog()` + guide. The catalog-demo already proves the pattern works. Lean root stays lean.
2. **D closes the genuine gaps** that no upstream package addresses: Channel-to-Bus binding (~50 LOC), pull-based `EventStreamHandler` (~100 LOC), CloudEvents mapping (~30 LOC). Total new code: **~180 LOC**.
3. **Everything else is adoption, documentation, and deprecation** — not invention.

---

## 7. Phased Delivery Plan

### Phase 1: Documentation + deprecation markers (zero risk, immediate value)

**Deliverables:**

- `docs/guides/data-mesh-interchange.md` — comprehensive guide showing how to use `catalog/v4` + `simple.New()` + `docserver` in a cqrs-htmx app for data-mesh interchange. Reference the existing `examples/catalog-demo/`. Cover DataProduct, Channel, AsyncAPI export, eventcatalog.dev generation, and the `watermill.WithBackend` Kafka/NATS federation seam.
- `// Deprecated:` markers on `EventCatalog`, `EventCatalogHandler`, `openapi.Spec`, `OpenAPISpecHandler` pointing to `catalog/v4` + `docserver`.
- Update `AGENTS.md` to document the catalog/v4 relationship.

**Acceptance:** A consumer reading the guide can build a full data-mesh contract (DataProduct + Channel + AsyncAPI + eventcatalog.dev) without any library code changes.

### Phase 2: `usermgmt.DefaultCatalog()` (thin, high-value)

**Deliverables:**

- `usermgmt.DefaultCatalog()` returning `*catalog.Catalog` — built via `simple.New()` from the existing 21 identity event types, 19 command types, and queries. Uses `simple.Event[UserRegisteredPayload]()` etc. to auto-derive schemas from the Go types in `identity-model/events.go`.
- Lives in a new file `usermgmt/es_catalog.go`. The `usermgmt/go.mod` adds an explicit `catalog/v4` dependency.
- Keep `DefaultEventCatalog()` as a deprecated thin wrapper that delegates to the catalog and extracts a flat event list, or delete it if no consumers depend on it.

**Acceptance:** `curl /asyncapi.json` on a usermgmt app returns a valid AsyncAPI 3.0 document with all 21 identity events.

### Phase 3: Build the genuinely missing pieces (Approach D)

**Deliverables:**

- `event_stream_handler.go` (~100 LOC) — `EventStreamHandler(journal, opts...)` serving `GET /events?after=<eventID>&limit=N`. Content negotiation: JSON array (default) or NDJSON. Uses `StreamingJournal.ReadStreamFrom` when available (non-materializing iterator), falls back to `SeekableJournal.ReadFrom`. Options: `WithStreamLimit`, `WithStreamFilter`, `WithCloudEvents(source)`.
- `cloudevent.go` (~30 LOC) — `CloudEvent` struct (7 CloudEvents 1.0 required fields: `specversion`, `id`, `source`, `type`, `time`, `subject`, `datacontenttype`, `data`) + `CloudEventFromEvent(evt event.Event, source string) CloudEvent`. Pure function, no SDK dependency. Pipable through `SSEBroker.payloadTransform` or usable standalone.
- `channel_binding.go` (~50 LOC) — `ChannelBinding` interface binding a `catalog.Channel` to a runtime transport. Implementations: `BusChannelBinding` (subscribe to `event.Bus`), `JournalChannelBinding` (read from `StreamingJournal`). Exposes a `Stream(ctx) (<-chan event.Event, error)` or similar.

**Acceptance:**

- `curl '/events?after=01H...'` returns a JSON array of CloudEvents.
- A `catalog.Channel` can be bound to a runtime `event.Bus` and the binding exposes the live stream.

### Phase 4: Transport consolidation (deferred, larger effort)

Evaluate adopting `transport/http/v4.SSEBroker` to replace `Broadcaster`/`JournalSSEStore`. This is a bigger migration (the `Broadcaster` API is used by dashboardui, adminui, and consumers). Defer until Phase 1-3 prove demand and the catalog adoption is stable.

---

## Appendix A: Verified Source References

### go-cqrs-lite `catalog/v4@v4.1.0`

Base path: `/home/lars/go/pkg/mod/github.com/larsartmann/go-cqrs-lite/catalog/v4@v4.1.0/`
Local checkout: `/home/lars/projects/go-cqrs-lite/catalog/`

| Type / Function                    | File                       | Line | Notes                                                               |
| ---------------------------------- | -------------------------- | ---- | ------------------------------------------------------------------- |
| `DataProduct`                      | `types_resources.go`       | 37   | "represents a data product in a data mesh"                          |
| `DataContract`                     | `types_resources.go`       | 83   | Path, Name, Type                                                    |
| `DataProductOutput`                | `types_resources.go`       | 91   | Ref + Contract                                                      |
| `Domain.DataProducts`              | `types.go`                 | 190  | `[]DataProductID`                                                   |
| `Catalog.DataProducts`             | `types.go`                 | 219  | `[]DataProduct`                                                     |
| `Channel`                          | `types.go`                 | 193  | Address, Protocols, Messages, DeliveryGuarantee, Routes             |
| `ChannelOption`                    | `channel_config.go`        | 5    | Address, Protocols, Messages, DeliveryGuarantee, Parameters, Routes |
| `Message`                          | `types.go`                 | 126  | Kind, Schema, Producers, Consumers, Operation, Changelog            |
| `MessageKind`                      | `types.go`                 | 97   | command, event, query                                               |
| `Direction`                        | `types.go`                 | 79   | sends, receives                                                     |
| `Service`                          | `types.go`                 | 151  | Commands, Events, Queries, WritesTo, ReadsFrom                      |
| `Domain`                           | `types.go`                 | 173  | Services, Sends, Receives, DataProducts, SubDomains                 |
| `Catalog`                          | `types.go`                 | 208  | Top-level aggregate root                                            |
| `Operation`                        | `types_helpers.go`         | 29   | Method, Path, StatusCodes                                           |
| `SchemaFromType[T]()`              | `schema.go`                | —    | Auto-derive JSON Schema from Go types via reflection                |
| `Command[T]()`                     | `message_config.go`        | 206  | Generic command registration                                        |
| `Event[T]()`                       | `message_config.go`        | 214  | Generic event registration                                          |
| `Query[T]()`                       | `message_config.go`        | 222  | Generic query registration                                          |
| `Builder.AddDataProduct`           | `build.go`                 | 112  | Programmatic accumulation                                           |
| `Registry`                         | `registry.go`              | 15   | Thread-safe map accumulators                                        |
| `Registry.AddDataProduct`          | `registry_resources.go`    | 59   | —                                                                   |
| `Exporter[T]` interface            | `exporter.go`              | 8    | `Export(cat *Catalog) T`                                            |
| `WalkDataProducts`                 | `walk.go`                  | 50   | Iterator                                                            |
| `Validate()`                       | `validate.go`              | 25   | Includes `validateDataProduct` (`:305`)                             |
| `simple.New()`                     | `simple/builder.go`        | —    | Fluent high-level builder                                           |
| `asyncapi.Document`                | `asyncapi/types.go`        | 9    | AsyncAPI 3.0 document model                                         |
| `asyncapi.Exporter`                | `asyncapi/exporter.go`     | 22   | `Export(cat) *Document`                                             |
| `asyncapi.Export()`                | `asyncapi/builder.go`      | 11   | Converts Catalog to AsyncAPI                                        |
| `openapi.Exporter`                 | `openapi/exporter.go`      | —    | Converts Catalog to OpenAPI                                         |
| `d2.Exporter`                      | `d2/exporter.go`           | —    | Converts Catalog to D2 diagram                                      |
| `eventcatalog.Exporter`            | `eventcatalog/exporter.go` | —    | Converts Catalog to eventcatalog.dev MDX                            |
| `docserver.NewDocsServer()`        | `docserver/docserver.go`   | —    | Serves OpenAPI + AsyncAPI + D2 + HTML UIs                           |
| `docserver.D2Handler()`            | `docserver/docserver.go`   | —    | D2 diagram handler                                                  |
| `docserver.HealthCheckHandler()`   | `docserver/docserver.go`   | —    | Health check handler                                                |
| `docserver.GenerateEventCatalog()` | `docserver/embed.go`       | —    | Build-time MDX generation                                           |

### go-cqrs-lite `transport/http/v4@v4.1.0`

Base path: `/home/lars/go/pkg/mod/github.com/larsartmann/go-cqrs-lite/transport/http/v4@v4.1.0/`

| Type / Function              | File     | Line  | Notes                                                                  |
| ---------------------------- | -------- | ----- | ---------------------------------------------------------------------- |
| `SSEBroker`                  | `sse.go` | 27    | Production-grade event.Bus to SSE bridge                               |
| `SSEBroker.journal`          | `sse.go` | 33    | `event.SeekableJournal` for Last-Event-ID replay                       |
| `SSEBroker.eventFilter`      | `sse.go` | 39    | `func(event.Type) bool`                                                |
| `SSEBroker.payloadTransform` | `sse.go` | 40    | `func(event.Event) []byte` — CloudEvents hook                          |
| `SSEBroker.dedupRingCap`     | `sse.go` | 38    | Dedup ring buffer                                                      |
| `SSEBroker.replayByteBudget` | `sse.go` | 37    | Replay byte budget                                                     |
| `SSEBroker.replayMetrics`    | `sse.go` | 36    | `*ReplayMetrics`                                                       |
| `NewSSEBroker()`             | `sse.go` | 57    | Constructor (subscribes via `bus.SubscribeAll`)                        |
| `SSEHandler()`               | `sse.go` | 198   | Returns `http.Handler`                                                 |
| Design vision                | `doc.go` | 6-7   | "Future transports (gRPC, NATS, Redis) will live as sibling modules"   |
| gRPC exclusion rationale     | `doc.go` | 36-37 | "consumers who need bidirectional transport should use transport/grpc" |

### go-cqrs-lite `event/v4@v4.1.0`

| Type / Function           | File                  | Line | Notes                                                  |
| ------------------------- | --------------------- | ---- | ------------------------------------------------------ |
| `Event = *ImmutableEvent` | `event.go`            | 55   | Type alias                                             |
| `ImmutableEvent`          | `event.go`            | 59   | id, eventType, streamID, payload, metadata, occurredAt |
| `Event.ID()`              | `event.go`            | 80   | returns `id.EventID` (ULID, cursor)                    |
| `Event.Type()`            | `event.go`            | 83   | returns `Type`                                         |
| `Event.StreamID()`        | `event.go`            | 86   | returns `id.StreamID`                                  |
| `Event.Payload()`         | `event.go`            | 118  | returns cloned `[]byte`                                |
| `Event.OccurredAt()`      | `event.go`            | 129  | returns `time.Time`                                    |
| `EventIterator`           | `streaming_source.go` | 28   | `Next() (Event, error); Close() error`                 |
| `StreamingJournal`        | `streaming_source.go` | 60   | `ReadStream(ctx); ReadStreamFrom(ctx, afterID, limit)` |
| `SeekableJournal`         | `store.go`            | 113  | `ReadFrom(ctx, afterID, limit) ([]Event, error)`       |
| `Bus`                     | `bus.go`              | 40   | Publisher + Subscriber + Use + UsePublish              |
| `Publisher`               | `bus.go`              | 12   | `Publish(ctx, events ...Event) error`                  |
| `Subscriber`              | `bus.go`              | 26   | `Subscribe(type, handler); SubscribeAll(handler)`      |

### cqrs-htmx (current state)

| What                         | File                                          | Line | Status                                                                 |
| ---------------------------- | --------------------------------------------- | ---- | ---------------------------------------------------------------------- |
| `EventCatalog`               | `event_catalog.go`                            | 36   | Does not import catalog/v4 — hand-rolled 3-field `EventMetadata`       |
| `EventCatalogHandler`        | `event_catalog_handler.go`                    | 22   | Serves JSON array only (no AsyncAPI, no doc server)                    |
| `openapi.Spec`               | `openapi/types.go`                            | 20   | Hand-rolled OpenAPI 3.1 builder (~400 LOC)                             |
| `OpenAPISpecHandler`         | `options_openapi.go`                          | 52   | Separate from EventCatalogHandler                                      |
| `Broadcaster`                | `sse_broadcaster.go`                          | 24   | Simple fan-out (no filter, no dedup, no metrics)                       |
| `JournalSSEStore`            | `event_store_sse.go`                          | 32   | SSE-framed replay (not JSON/NDJSON)                                    |
| Root `go.mod` cqrs-lite deps | `go.mod`                                      | —    | event, command, query, id — **not catalog, not transport/http**        |
| `catalog/v4` usage           | `integration_test/`, `examples/catalog-demo/` | —    | Proven but not exposed as first-class                                  |
| `transport/http/v4` usage    | nowhere                                       | —    | **Zero imports anywhere in the repo**                                  |
| `StreamingJournal` usage     | nowhere                                       | —    | **Zero imports anywhere** (uses slice-materializing `SeekableJournal`) |

---

## Resolution (2026-07-31)

**Status: under consideration — no code written.** The proposal's recommendation (Approach C+D: evaluate consolidating `EventCatalog`/`openapi/` with `catalog/v4`, plus build 3 genuinely missing runtime pieces) is tracked in `ROADMAP.md` → "Data Mesh Interchange (Researched — Not Yet Adopted)". The three gaps (~180 LOC total) are: (1) Channel-to-runtime binding (~50 LOC), (2) CloudEvents envelope (~30 LOC), (3) Pull-based machine transport (~100 LOC). Not committed to a release. The strategic angle (per the landscape research at `docs/research/2026-07-25_data-mesh-landscape-and-event-sourced-advantage.md`): event sourcing structurally prevents the data-discovery problems DataHub/OpenMetadata/ODDS exist to solve, so time-travel + the catalog are a stronger positioning than a bespoke mesh product.
