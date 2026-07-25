# Data-Mesh Interchange Interface for cqrs-htmx

**Date:** 2026-07-25
**Status:** Proposal — awaiting decision
**Scope:** Root module (`github.com/larsartmann/cqrs-htmx/v4`)

---

## TL;DR

cqrs-htmx already holds the three contract primitives a data-mesh interchange needs (Published Language via `EventCatalog`, HTTP input-port via `openapi/`, and a cursor-based ordered event log via `event.SeekableJournal`), but it lacks three things: **(1) a machine-readable transport**, **(2) a vendor-neutral event envelope**, and **(3) a data-product descriptor**. This proposal evaluates five implementation options and recommends a phased delivery: CloudEvents-framed `EventStreamHandler` first, `DataProduct` envelope second, AsyncAPI export third.

---

## Table of Contents

- [1. Problem Statement](#1-problem-statement)
- [2. Research: Existing Primitives in cqrs-htmx](#2-research-existing-primitives-in-cqrs-htmx)
- [3. Research: Industry Standards](#3-research-industry-standards)
- [4. The Three Real Gaps](#4-the-three-real-gaps)
- [5. Five Implementation Options](#5-five-implementation-options)
- [6. Recommendation](#6-recommendation)
- [7. Phased Delivery Plan](#7-phased-delivery-plan)
- [8. Open Questions](#8-open-questions)
- [Appendix A: Verified Code References](#appendix-a-verified-code-references)
- [Appendix B: Standards Quick Reference](#appendix-b-standards-quick-reference)

---

## 1. Problem Statement

> **How can we provide an interchange interface for data-mesh scenarios?**

A data mesh treats domains as independent **data products** that expose their data through well-defined contracts. For cqrs-htmx — a CQRS/event-sourcing library — this means: how does a domain built on this library **publish its events** to other domains/services in a discoverable, versioned, transport-agnostic, and standards-aligned way?

The question is explicitly about **interchange** (machine-to-machine, cross-domain), not the existing SSE/WS realtime (browser-to-server, within one app).

### Design constraints (from the library principle)

From `AGENTS.md`:

- **Library, not platform.** "Never enforce defaults consumers might disagree with." The library provides building blocks; consumers compose them with their infrastructure (Kafka, NATS, Backstage, Confluent).
- **Framework-agnostic.** Works with `net/http`, Chi, Gin. Never picks the router.
- **Opt-in everything.** No mandatory CSP/HSTS/CSRF. Interchange features must follow the same pattern.
- **Dependency-free where possible.** The `openapi/` sub-package has zero deps on the root package; the envelope type should avoid dragging in `cloudevents/sdk-go` (a heavy SDK) when the spec is only 7 required fields.

---

## 2. Research: Existing Primitives in cqrs-htmx

The library already contains most of the building blocks. This section is grounded in a full read of the source (see [Appendix A](#appendix-a-verified-code-references) for exact file:line).

### 2.1 Published Language — `EventCatalog`

**Files:** `event_catalog.go`, `event_catalog_handler.go`

`EventCatalog` (`event_catalog.go:36`) is explicitly designed for the Published Language pattern (DDD). It is a mutable registry of `EventMetadata` (`event_catalog.go:21`):

```go
type EventMetadata struct {
    Type          string         `json:"type"`          // "UserRegistered"
    Aggregate     string         `json:"aggregate"`     // "User"
    SchemaVersion int            `json:"schemaVersion"` // 2
    Description   string         `json:"description,omitempty"`
    PayloadFields []PayloadField `json:"payloadFields,omitempty"`
}
```

`EventCatalogHandler` (`event_catalog_handler.go:22`) serves it as immutable JSON with a 1-year cache and FNV-1a ETag via the shared `serveImmutableJSON` helper (`event_catalog_handler.go:47`). `usermgmt.DefaultEventCatalog()` pre-registers all 21 identity events.

**Role in a mesh:** This is the **output-port schema** — the list of event types a domain emits.

### 2.2 HTTP Input-Port Contract — `openapi/`

**Files:** `openapi/types.go`, `openapi/builder.go`, `openapi/schema.go`, `options_openapi.go`

A dependency-free fluent builder for OpenAPI 3.1 specs. `Spec` (`openapi/types.go:20`) carries `Info`, `Paths`, `Components`. `OpenAPISpecHandler` (`options_openapi.go:52`) serves it as immutable JSON via the same `serveImmutableJSON` path.

`WithOpenAPI(op)` (`options_openapi.go:32`) attaches operation metadata to a handler — pure documentation, no runtime effect, because the library does not own the consumer's router.

**Role in a mesh:** This is the **input-port contract** — the commands/queries a domain accepts.

### 2.3 Cursor-Based Event Log — `event.SeekableJournal`

**Source:** go-cqrs-lite `event/v4` (consumed by cqrs-htmx)

```go
type Journal interface {
    ReadAll(ctx) ([]Event, error)                           // global, ordered by OccurredAt
}
type SeekableJournal interface {
    Journal
    ReadFrom(ctx, afterEventID id.EventID, limit int) ([]Event, error) // position-based cursor
}
```

The cursor is the event's ULID `EventID` — monotonically ordered, so `ReadFrom(afterID, N)` returns the next N events after that position. This is **the natural interchange primitive**: an ordered, resumable global event log.

### 2.4 Replay Machinery (browser-framed)

**Files:** `event_store_sse.go`, `sse_store.go`, `sse_event.go`

- `JournalSSEStore` (`event_store_sse.go:32`) adapts `event.SeekableJournal` into `SSEEventStore`.
- `JournalSSEStore.EventsAfter(lastID)` (`event_store_sse.go:101`) calls `SeekableJournal.ReadFrom` under the hood — cursor-based replay already works.
- `ReplayEvents(stream, store, lastEventID)` (`sse_store.go:25`) drives the replay.
- `LastEventIDFromRequest(r)` (`sse_event.go:64`) reads the `Last-Event-ID` header.

**The problem:** This machinery emits **SSE wire format** (`event: foo\ndata: {...}\n\n`). That is browser-shaped. A service consumer wants `GET /events/stream?after=<id>` → JSON array or NDJSON of envelopes.

### 2.5 Federation Seam (undocumented)

**Source:** go-cqrs-lite `watermill/v4`

`watermill.EventBus` exposes `WithBackend(publisher, subscriber, closer)` — inject Kafka/NATS Watermill plugins to make the in-process bus multi-process. Currently single-topic (`DefaultEventBusTopic = "cqrs.events"`), undocumented, with no example.

### 2.6 Schema Evolution

**File:** `identity-model/upcaster.go`

`UpcasterRegistry` chains upcasters v0 → … → current (`CurrentSchemaVersion = 2`). Handles backward-compatible reads across versions.

### 2.7 What does NOT exist

Grep for `interchange|exchange|data product|mesh|federat` across `*.go` returned **zero matches** for the mesh concepts. The terms "contract", "registry", "schema" appear only incidentally (OAuth2 token exchange, interface contracts, the EventCatalog calling itself "a registry"). There is **no** data-product envelope, **no** federated discovery, **no** machine transport, **no** CloudEvents framing.

---

## 3. Research: Industry Standards

Grounded in the actual specs (AsyncAPI, CloudEvents verified via web fetch; DPDS/Bitol from spec knowledge).

### 3.1 Data Product Descriptor Specification (DPDS)

Maintained by the Open Data Mesh Initiative (Agile Lab). Top-level: `dataProductDescriptorSpecVersion`, `info` (owner, version SemVer, name), `interface` (the contract), `components` (`inputPorts`, `outputPorts`, `discoveryPorts`, `observabilityPorts`).

Each output port has `promises` containing a `dataContracts` / `api` object whose `specification` is usually an **OpenAPI or AsyncAPI document** (inline or `$ref`). DPDS wraps existing spec formats — it does not invent a new message schema.

**Relevance:** DPDS is the canonical "data product envelope." A Go library can model a slim subset.

### 3.2 Data Contracts (Bitol / Data Contract CLI)

**Bitol** (`bitol-io/data-contracts`): YAML/JSON with `dataContractSpecification: 1.1.1`, `info` (title, version, status, owner, contact), `servers` (per protocol: `type: kafka|postgres|bigquery|s3|local`, `host`, `topic`), `terms` (usage, limitations, billing, noticePeriod), `schema` (type: dbt|sql|jsonschema|avro|protobuf), `servicelevels` (freshness, frequency, latency, availability, retention, support, backup), `quality`.

`info.status` ∈ `proposed | draft | active | deprecated | retired`. The CLI validates and exports to OpenAPI/AsyncAPI/Soda/dbt/Great Expectations.

**Relevance:** Bitol is the most pragmatic contract format — servers + schema + SLAs + terms. A Go `DataProduct` could mirror this subset.

### 3.3 AsyncAPI

Spec `asyncapi: 3.0.x`. Differs from OpenAPI by modeling **channels/topics + messages + operations** instead of request/response paths. `channels` keys are addresses (Kafka topic / AMQP queue / SSE stream); each channel declares `messages` and `address`.

Each message has `name`, `title`, `contentType`, `headers` (JSON Schema), `payload` (`schemaFormat` + inline/`$ref` schema), `correlationId`, `bindings`. Transport-specific detail lives in `bindings` (`bindings.kafka`, `bindings.http`, `bindings.nats`, etc.).

**Relevance:** AsyncAPI is the OpenAPI-for-events. It describes both the schema and the transport in one document. Tooling generates Go types (modelina) and renders eventcatalog.dev pages. This is the richest contract format for event-driven architectures.

### 3.4 CloudEvents (CNCF, graduated)

Problem: every producer used bespoke envelopes, so consumers rewrote parsing per source. CloudEvents defines one vendor-neutral event envelope so routers (Knative Eventing, AWS EventBridge, Azure EventGrid) and sinks are interoperable.

**Required attributes:** `specversion` ("1.0"), `id`, `source` (URI, identifies producer context, unique per `id`), `type` (e.g. `com.example.order.created.v3`).
**Optional:** `time`, `subject`, `datacontenttype`, `dataschema` (URI to a schema).

Three serializations: **structured** (whole envelope is the body, `application/cloudevents+json`), **binary** (attributes in headers, `data` is raw body — dominant Kafka mode), **batch**. Protocol bindings exist for HTTP, Kafka, AMQP, MQTT, WebSockets, NATS.

Go SDK: `github.com/cloudevents/sdk-go/v2`.

**Relevance:** This is the right **per-message wire format**. It is normalized, vendor-neutral, and every event router speaks it. The `type` + `source` pair maps naturally to cqrs-htmx's `event.Type` + service identity.

### 3.5 Schema Registry (Confluent, Apicurio)

REST API backing state into a compacted Kafka topic `_schemas`. **Subject** = the name a schema is registered under; versions are sequential integers per subject. Subject-naming strategies: `TopicNameStrategy`, `RecordNameStrategy`, `TopicRecordNameStrategy`.

Compatibility modes: `NONE`, `BACKWARD` (new schema reads old data — default), `FORWARD`, `FULL`, `*_TRANSITIVE`. Producers register/lookup schema before producing; consumers download by ID.

**Relevance:** A `Registry` interface (`Register`, `GetByID`, `CheckCompatibility`, `GetLatest`) abstracting Confluent + Apicurio is the right boundary if schema registry support is added.

### 3.6 Event Catalog (eventcatalog.dev)

Open-source documentation site generator for event-driven/AsyncAPI architectures. Not a registry, not a transport — it's the **discovery/documentation layer**. Ingests AsyncAPI and OpenAPI as the source of truth.

**Relevance:** This is where discovery lives in a real mesh. cqrs-htmx should produce the contract documents that EventCatalog consumes, not reinvent discovery.

### 3.7 The 3 Layers in Real Data-Mesh Implementations

| Layer | What | Examples |
|-------|------|----------|
| **(a) Envelope / descriptor** | A data-product or contract file the owning domain authors and versions in git | DPDS descriptor, Bitol YAML, AsyncAPI doc |
| **(b) Discovery / registry** | Where consumers find contracts | EventCatalog, Schema Registry, Backstage, DataHub |
| **(c) Transport** | The actual wire | Kafka topics (+ Schema Registry + Avro/Protobuf + CloudEvents), Pulsar, NATS JetStream, webhook, SSE, HTTP |

The descriptor's `servers[].type` enumerates the transport; the message on the wire is the CloudEvents/AsyncAPI message; the contract and registry keep producers and consumers aligned without coupling.

---

## 4. The Three Real Gaps

Cross-referencing the existing primitives (§2) against the industry 3-layer model (§3.7):

| Gap | What's missing | What exists (partially) |
|-----|----------------|------------------------|
| **Gap 1: Machine transport** | `GET /events/stream?after=<id>` → JSON/NDJSON of envelopes for service consumers | `SeekableJournal.ReadFrom` cursor + `JournalSSEStore.EventsAfter` replay logic exist, but framed for browsers (SSE wire format) |
| **Gap 2: Event envelope** | CloudEvents-style `source`/`type`/`dataschema`/`subject` on every message | Events carry `event.Type` and payload, but no vendor-neutral envelope. `dashboardui/sse.go` payload is ad-hoc |
| **Gap 3: Data-product descriptor** | One document with owner/version/SLO/transport binding that composes EventCatalog + OpenAPI + stream URL | EventCatalog and OpenAPI are served at separate URLs with no binding |

The federation seam (`watermill.WithBackend`) is present but undocumented and single-topic — **not** the first thing to fix.

---

## 5. Five Implementation Options

| # | Option | Scope | New code | Standards alignment | Solves |
|---|--------|-------|----------|---------------------|--------|
| **1** | **DataProduct envelope only** | Root module | ~1 file | DPDS-lite | Gap 3 (discovery) |
| **2** | **CloudEvents-framed `EventStreamHandler`** | Root module | ~2 files | CloudEvents 1.0 | Gap 1 + Gap 2 (transport + envelope) |
| **3** | **`asyncapi/` contract package** | New sub-package | ~`openapi/`-sized | AsyncAPI 3.0 | Gaps 2+3 (rich contract with transport bindings) |
| **4** | **`DataProductRegistry` federation** | Root module | ~2 files | DPDS registry | Mesh-wide discovery (layer b) |
| **5** | **Watermill `TopicRouter` + backend docs** | Docs + small helper | ~1 file + example | Kafka/NATS via Watermill | Production-scale push transport (layer c) |

### Option 1 — DataProduct envelope only

A `DataProduct` struct that composes the existing pieces into one discoverable descriptor:

```go
dp := cqrshtmx.NewDataProduct("identity").
    Owner("team-platform", "platform@x.com").
    Version("1.2.0").
    OutputPort(cqrshtmx.DataPort{
        Name:        "user-events",
        Description: "All events emitted by the identity domain",
        Catalog:     svc.EventCatalog(),     // *EventCatalog (existing)
        Transport:   "https",                 // or "kafka", "nats"
        StreamURL:   "https://api.x.com/events/stream",
    }).
    InputPort(cqrshtmx.DataPort{
        Name:        "identity-api",
        Description: "HTTP commands and queries",
        Spec:        openapiSpec,             // *openapi.Spec (existing)
        Transport:   "https",
        BaseURL:     "https://api.x.com",
    }).
    SLO(cqrshtmx.SLO{Availability: "99.9%", LatencyP99: "200ms"})
```

Served via `DataProductHandler(dp)` reusing `serveImmutableJSON` (`event_catalog_handler.go:47`).

**Pros:** Trivial (~1 file); composes what exists; makes the contract discoverable as one document; aligns with DPDS without adopting its full weight.
**Cons:** Doesn't move bytes — discovery only. Must be paired with a transport.
**LOC estimate:** ~120 LOC + tests.

### Option 2 — CloudEvents-framed `EventStreamHandler`

`EventStreamHandler(journal event.SeekableJournal, opts...)` returns a handler serving `GET /events/stream?after=<eventID>&limit=N` → `[]CloudEvent` JSON (or `application/x-ndjson` streaming).

Each event is framed as a CloudEvent:

```go
type CloudEvent struct {
    SpecVersion     string          `json:"specversion"`           // "1.0"
    ID              string          `json:"id"`                    // event ULID
    Source          string          `json:"source"`                // service URI (e.g. "/identity")
    Type            string          `json:"type"`                  // event.Type (e.g. "UserRegistered")
    Time            time.Time       `json:"time"`                  // event.OccurredAt
    Subject         string          `json:"subject"`               // aggregate ID
    DataContentType string          `json:"datacontenttype"`       // "application/json"
    Data            json.RawMessage `json:"data"`                  // event payload
}
```

Cursor = ULID event ID (same as `JournalSSEStore` already uses). Resumable, cache-friendly, router-compatible.

```go
mux.Handle("GET /events/stream", cqrshtmx.EventStreamHandler(journal,
    cqrshtmx.WithStreamSource("/identity"),
    cqrshtmx.WithStreamLimit(1000),
    cqrshtmx.WithStreamFilter(eventTypeFilter), // optional: filter by event.Type
))
```

**Pros:** Closes Gap 1 + Gap 2 together. The cursor, replay logic, and immutable-serve pattern all exist — only the framing is missing. CloudEvents is vendor-neutral and every router speaks it. ~150 LOC, no new deps (the envelope is 7 required fields; no need for `cloudevents/sdk-go`).
**Cons:** Pull-based only (client polls). No push notifications when new events arrive. Consumers wanting push must compose with webhooks or the SSE handler.
**LOC estimate:** ~150 LOC + tests.

### Option 3 — `asyncapi/` contract package

A parallel to `openapi/`: fluent builder for AsyncAPI 3.0.

```go
spec := asyncapi.New("Identity Events", "1.0.0").
    Server("production", "api.x.com", asyncapi.ProtocolHTTP).
    Channel("user-events", asyncapi.Send,
        asyncapi.Message("UserRegistered").
            ContentType("application/json").
            Payload(userRegisteredSchema),
    )
```

`EventCatalog` becomes a thin facade that can *export* to AsyncAPI via `EventCatalog.AsyncAPI()`.

**Pros:** AsyncAPI is the OpenAPI-for-events. One document describes both schema and transport (via `bindings.kafka`/`bindings.http`/`bindings.nats`). Tooling generates Go types (modelina), renders eventcatalog.dev pages. Standards-correct.
**Cons:** Bigger investment. AsyncAPI 3.0 has more surface area than OpenAPI 3.1 (channels, operations, messages, bindings). Best deferred until Options 1+2 prove demand.
**LOC estimate:** ~`openapi/`-sized (~400 LOC).

### Option 4 — `DataProductRegistry` federation

A federated discovery layer:

```go
reg := cqrshtmx.NewDataProductRegistry().
    Register("identity", "https://api.x.com/.well-known/data-product").
    Register("billing", "https://billing.x.com/.well-known/data-product").
    Register("inventory", "https://inv.x.com/.well-known/data-product")

mux.Handle("GET /.well-known/data-products", cqrshtmx.DataProductRegistryHandler(reg))
```

Remote descriptors fetched on a schedule, cached, staleness-tagged. Served as mutable JSON with short TTL + per-request ETag (like `ProjectionStatusHandler`).

**Pros:** This is where "mesh" actually starts — federated discovery across domains.
**Cons:** Platform territory. The library principle says make it building blocks, not a service. Consumers may prefer Backstage/DataHub/EventCatalog for this.
**LOC estimate:** ~200 LOC + tests.

### Option 5 — Watermill `TopicRouter` + backend docs

A `TopicRouter` mapping `event.Type` → topic name (replacing the single `DefaultEventBusTopic = "cqrs.events"`):

```go
router := cqrshtmx.NewTopicRouter().
    Route("UserRegistered", "identity.users").
    Route("TenantCreated", "identity.tenants").
    Default("identity.misc")

bus := watermill.NewEventBus(
    watermill.WithBackend(kafkaPublisher, kafkaSubscriber, kafkaCloser),
    watermill.WithEventBusTopic(router.TopicFor),
)
```

Combined with `WithBackend(kafkaPublisher, kafkaSubscriber, closer)`, this gives real Kafka/NATS federation.

**Pros:** Production-scale push transport. Mostly documentation + an example — the seam exists, it's just invisible.
**Cons:** No new transport code in cqrs-htmx; consumers bring Watermill plugins. Kafka/NATS are infra choices the library should not enforce.
**LOC estimate:** ~80 LOC + docs + example.

---

## 6. Recommendation

### Ranked recommendation: **Option 2 first → Option 1 second → Option 3 as the evolution path**

#### Why Option 2 first

1. **It is the only thing that actually closes a gap.** The cursor (`SeekableJournal.ReadFrom`), replay logic, and immutable-serve pattern all exist — only the framing is missing. Without a machine transport, "interchange" is just documentation.
2. **CloudEvents is the right envelope.** 7 required fields, vendor-neutral, every router speaks it. The `type` + `source` pair maps naturally to cqrs-htmx's `event.Type` + service identity.
3. **Small surface, no new deps.** ~150 LOC. The envelope is a plain struct — no need for `cloudevents/sdk-go` (a heavy SDK) when the spec is this small.
4. **Immediate utility.** A consumer can `GET /events/stream?after=<id>` and get a resumable JSON stream of CloudEvents. This works with curl, HTTP clients, and can be bridged to Kafka/NATS by the consumer.

#### Why Option 1 second

1. **Nearly free once Option 2 exists.** The descriptor references the stream URL from Option 2, the EventCatalog, and the OpenAPI spec — all already served.
2. **Turns scattered endpoints into a data product.** Instead of EventCatalog at `/events/catalog`, OpenAPI at `/openapi.json`, stream at `/events/stream` — one document at `/.well-known/data-product` with owner/version/SLO/transport binding.
3. **Aligns with DPDS without adopting its full weight.** A slim Go struct, not a YAML parser.

#### Why Option 3 is the evolution path (deferred)

1. **AsyncAPI is the right format** for event-driven contracts, but it's a bigger investment.
2. **Clean migration:** `EventCatalog.AsyncAPI()` emits an AsyncAPI doc from the registered metadata — EventCatalog stays the source of truth, AsyncAPI becomes an export view. No rewrite.
3. **Defer until demand is proven** by Options 1+2 adoption.

#### Why Options 4 and 5 are consumer composition, not library code

1. **Option 4 (federation registry):** The library should *enable* discovery (via the descriptor in Option 1) but not *be* a discovery platform. Consumers compose Backstage/DataHub/EventCatalog.
2. **Option 5 (Kafka backend):** The library should *document* the existing `WithBackend` seam and provide a `TopicRouter` helper, but not enforce Kafka/NATS. Consumers bring Watermill plugins.
3. **The "never enforce defaults" principle** (AGENTS.md) means the library provides building blocks; the consumer's infrastructure choices stay theirs.

### Decision matrix

| Criterion | Opt 1 | Opt 2 | Opt 3 | Opt 4 | Opt 5 |
|-----------|-------|-------|-------|-------|-------|
| Closes a real gap | Partial (discovery) | **Yes (transport+envelope)** | Yes (contract) | Yes (federation) | Yes (push transport) |
| New code size | ~120 LOC | ~150 LOC | ~400 LOC | ~200 LOC | ~80 LOC |
| New dependencies | None | None | None | None | None (consumer brings Watermill plugins) |
| Standards alignment | DPDS-lite | **CloudEvents 1.0** | AsyncAPI 3.0 | DPDS registry | Kafka/NATS via Watermill |
| Reuses existing code | EventCatalog + OpenAPI | SeekableJournal + JournalSSEStore replay | EventCatalog (as export source) | Option 1 descriptors | watermill.WithBackend |
| Library principle fit | Strong | **Strong** | Strong | Medium (platform-flavored) | Strong (docs + helper) |
| Immediate utility | Discovery only | **Machine transport** | Rich contract docs | Mesh discovery | Push at scale |

---

## 7. Phased Delivery Plan

### Phase 1 (now): `EventStreamHandler` + `CloudEvent` envelope

**Deliverables:**
- `cloudevent.go` — `CloudEvent` struct (7 required + optional fields, CloudEvents 1.0 compliant).
- `event_stream_handler.go` — `EventStreamHandler(journal, opts...)` serving `GET /events/stream?after=<id>&limit=N`.
- Options: `WithStreamSource("/identity")`, `WithStreamLimit(1000)`, `WithStreamFilter(fn)`.
- Content negotiation: `Accept: application/json` → JSON array; `Accept: application/x-ndjson` → streaming NDJSON; default JSON array.
- Cursor validation and error mapping (404 for unknown cursor, 400 for malformed).
- Tests: unit tests for envelope, handler tests for cursor/replay/filter, example in `examples/`.

**Acceptance criteria:**
- `curl 'https://api.x.com/events/stream'` returns `[]CloudEvent` JSON.
- `curl 'https://api.x.com/events/stream?after=<id>'` returns events after the cursor.
- Events are valid CloudEvents 1.0 (validated against the spec).
- No new dependencies in `go.mod`.

### Phase 2 (next): `DataProduct` + `DataProductHandler`

**Deliverables:**
- `data_product.go` — `DataProduct` struct with `Owner`, `Version`, `OutputPorts`, `InputPorts`, `SLO`.
- `data_product_handler.go` — `DataProductHandler(dp)` reusing `serveImmutableJSON`.
- Example showing composition of EventCatalog + OpenAPI + EventStreamHandler URL.
- Guide: `docs/guides/data-mesh-interchange.md`.

**Acceptance criteria:**
- `curl /.well-known/data-product` returns one JSON document binding owner/version/SLO/ports.
- The descriptor references the EventCatalog, OpenAPI spec, and EventStreamHandler URL.

### Phase 3 (when justified): `EventCatalog.AsyncAPI()` exporter

**Deliverables:**
- `asyncapi.go` — `EventCatalog.AsyncAPI()` method emitting an AsyncAPI 3.0 document from registered metadata.
- Optionally a full `asyncapi/` sub-package if demand warrants.
- Integration with the `DataProduct` descriptor (output port can reference an AsyncAPI doc instead of / in addition to the EventCatalog).

**Acceptance criteria:**
- `EventCatalog.AsyncAPI()` produces a valid AsyncAPI 3.0 document.
- The document can be validated by the AsyncAPI parser.
- The document can be ingested by eventcatalog.dev.

### Documentation (cross-cutting): Consumer patterns for Options 4 and 5

**Deliverables:**
- `docs/guides/data-mesh-interchange.md` — covers the 3-layer model, how cqrs-htmx maps to it, and consumer patterns for federation (Option 4) and Kafka/NATS backends (Option 5).
- Example: `examples/data-mesh/` showing a two-service interchange using EventStreamHandler + DataProduct.

---

## 8. Open Questions

1. **Push vs pull transport.** Option 2 is pull-based (client polls). Should the library also provide a webhook/SSE-for-machines variant, or leave push to consumer composition (Option 5 / `Broadcaster`)? **Recommendation:** leave push to composition; pull is the simpler, more interoperable default.

2. **Schema registry integration.** Should the library provide a `Registry` interface abstracting Confluent/Apicurio, or leave that to consumer infra? **Recommendation:** defer until consumers ask; the CloudEvents `dataschema` URI is enough for discovery.

3. **Envelope dependency.** `cloudevents/sdk-go/v2` is a heavy SDK. The CloudEvents spec is only 7 required fields. **Recommendation:** implement a dependency-free `CloudEvent` struct; do not import the SDK.

4. **Filtering.** Should `EventStreamHandler` support server-side filtering by event type, aggregate, or content? **Recommendation:** start with event-type filter only (`WithStreamFilter(fn)`); defer content-based filtering.

5. **Authz.** Should the interchange endpoints integrate with the existing Casbin authz, or be treated as a separate concern? **Recommendation:** treat as separate — the consumer wraps the handler with their own auth middleware (consistent with the library principle).

---

## Appendix A: Verified Code References

All file:line references verified against the source on 2026-07-25.

### Root module (`github.com/larsartmann/cqrs-htmx/v4`)

| Primitive | File | Line | Signature / Type |
|-----------|------|------|------------------|
| `PayloadField` | `event_catalog.go` | 12 | `struct { Name, Type string; Required bool }` |
| `EventMetadata` | `event_catalog.go` | 21 | `struct { Type, Aggregate string; SchemaVersion int; Description string; PayloadFields []PayloadField }` |
| `EventCatalog` | `event_catalog.go` | 36 | `struct { events []EventMetadata }` — Published Language registry |
| `NewEventCatalog` | `event_catalog.go` | 41 | `func NewEventCatalog() *EventCatalog` |
| `EventCatalog.Register` | `event_catalog.go` | 48 | `func (c *EventCatalog) Register(meta EventMetadata)` |
| `EventCatalog.Events` | `event_catalog.go` | 62 | `func (c *EventCatalog) Events() []EventMetadata` |
| `EventCatalog.JSON` | `event_catalog.go` | 71 | `func (c *EventCatalog) JSON() ([]byte, error)` |
| `EventCatalogHandler` | `event_catalog_handler.go` | 22 | `func EventCatalogHandler(catalog *EventCatalog) (http.HandlerFunc, error)` |
| `serveImmutableJSON` | `event_catalog_handler.go` | 47 | `func serveImmutableJSON(w, r, etag string, data []byte)` — shared helper |
| `hashTag` | `options_openapi.go` | 86 | `func hashTag(data []byte) string` — FNV-1a 64-bit → hex ETag |
| `WithOpenAPI` | `options_openapi.go` | 32 | `func WithOpenAPI(op openapi.Operation) HandlerOption` |
| `OpenAPISpecHandler` | `options_openapi.go` | 52 | `func OpenAPISpecHandler(spec *openapi.Spec) (http.HandlerFunc, error)` |
| `JournalSSEStore` | `event_store_sse.go` | 32 | `struct { journal event.Journal; seekable event.SeekableJournal; mapper EventToSSEMapper; maxReplay int }` |
| `NewJournalSSEStore` | `event_store_sse.go` | 63 | `func NewJournalSSEStore(journal event.Journal, mapper EventToSSEMapper, opts ...JournalSSEStoreOption) *JournalSSEStore` |
| `JournalSSEStore.EventsAfter` | `event_store_sse.go` | 101 | `func (s *JournalSSEStore) EventsAfter(lastID SSEEventID) ([]SSEEvent, error)` |
| `WithMaxReplay` | `event_store_sse.go` | 45 | `func WithMaxReplay(n int) JournalSSEStoreOption` |
| `SSEEventStore` interface | `sse_store.go` | 12 | `interface { EventsAfter(lastID SSEEventID) ([]SSEEvent, error) }` |
| `ReplayEvents` | `sse_store.go` | 25 | `func ReplayEvents(stream *SSEStream, store SSEEventStore, lastEventID SSEEventID) (int, error)` |
| `LastEventIDFromRequest` | `sse_event.go` | 64 | `func LastEventIDFromRequest(r *http.Request) SSEEventID` |
| `Broadcaster` | `sse_broadcaster.go` | 24 | `struct { *sse.Broadcaster[sse.Event] }` |
| `Broadcaster.ServeSSE` | `sse_broadcaster.go` | 90 | `func (b *Broadcaster) ServeSSE(w http.ResponseWriter, r *http.Request)` |
| `WSBroadcaster` | `ws_broadcaster.go` | 31 | `struct { *sse.Broadcaster[string] }` |
| `SSEStream` | `sse_event.go` | 56 | `type SSEStream = sse.Stream` (type alias) |
| `ProjectionStatusProvider` | `projection_status_handler.go` | 28 | `interface { ProjectionStatuses() []ProjectionStatusEntry }` |
| `ProjectionStatusHandler` | `projection_status_handler.go` | — | `func ProjectionStatusHandler(provider ProjectionStatusProvider) http.HandlerFunc` |

### `openapi/` sub-package

| Primitive | File | Line |
|-----------|------|------|
| `Spec` | `openapi/types.go` | 20 |
| `Info` | `openapi/types.go` | 28 |
| `Components` | `openapi/types.go` | 35 |
| `PathItem` | `openapi/types.go` | 40 |
| `Operation` | `openapi/types.go` | 54 |
| `Parameter` | `openapi/types.go` | 66 |
| `RequestBody` | `openapi/types.go` | 75 |
| `Response` | `openapi/types.go` | 82 |
| `MediaType` | `openapi/types.go` | 88 |
| `New` | `openapi/builder.go` | 5 |
| `Schema` | `openapi/schema.go` | 10 |
| `Spec.JSON` | `openapi/marshal.go` | 11 |

### go-cqrs-lite (`event/v4`)

| Primitive | File (in module cache) | Line | Signature |
|-----------|------------------------|------|-----------|
| `EventSink` | `store.go` | 20 | `interface { Save(ctx, ref, events []Event, expectedVersion Version) error; AppendBatch(ctx, ref, events []Event) error }` |
| `EventSource` | `store.go` | 60 | `interface { Load(ctx, ref) ([]Event, error); LoadFromVersion(ctx, ref, version) ([]Event, error); ... }` |
| `Store` | `store.go` | 93 | `interface { EventSink; EventSource }` |
| `Journal` | `store.go` | 102 | `interface { ReadAll(ctx) ([]Event, error) }` |
| `SeekableJournal` | `store.go` | 113 | `interface { Journal; ReadFrom(ctx, afterEventID id.EventID, limit int) ([]Event, error) }` |
| `EventIterator` | `streaming_source.go` | 28 | `interface { Next() (Event, error); Close() error }` |
| `StreamingSource` | `streaming_source.go` | 43 | `interface { LoadStream(ctx, ref) (EventIterator, error); LoadStreamFromVersion(ctx, ref, version) (EventIterator, error) }` |
| `StreamingJournal` | `streaming_source.go` | 60 | `interface { ReadStream(ctx) (EventIterator, error); ReadStreamFrom(ctx, afterEventID, limit) (EventIterator, error) }` |
| `Handler` | `bus.go` | 8 | `type Handler func(ctx context.Context, event Event) error` |
| `Publisher` | `bus.go` | 12 | `interface { Publish(ctx, events ...Event) error }` |
| `Subscriber` | `bus.go` | 26 | `interface { Subscribe(eventType Type, handler Handler) error; SubscribeAll(handler Handler) error }` |
| `Bus` | `bus.go` | 40 | `interface { Publisher; Subscriber; Use(middleware ...Middleware) error; UsePublish(middleware ...PublishMiddleware) error }` |
| `Checkpoint` | `checkpoint.go` | 11 | `struct { EventID id.EventID; ProcessedAt time.Time }` |
| `CheckpointStore` | `checkpoint.go` | 42 | `interface { CheckpointSink; CheckpointSource }` |

### go-cqrs-lite (`watermill/v4`)

| Primitive | File (in module cache) | Line | Signature |
|-----------|------------------------|------|-----------|
| `DefaultEventBusTopic` | `event_bus.go` | 54 | `const DefaultEventBusTopic = "cqrs.events"` |
| `WithEventBusTopic` | `event_bus_options.go` | 19 | `func WithEventBusTopic(topic string)` |
| `WithBackend` | `event_bus_options.go` | 26 | `func WithBackend(pub message.Publisher, sub message.Subscriber, closer io.Closer)` |
| `EventBus.MessageSubscriber` | `event_bus.go` | 52 | `func (b *EventBus) MessageSubscriber() message.Subscriber` |

### go-cqrs-lite (`projection/v4`)

| Primitive | File (in module cache) | Line | Signature |
|-----------|------------------------|------|-----------|
| `Projection` | `projection.go` | 23 | `interface { Name() string; Handle(ctx, evt Event) error; EventTypes() []event.Type }` |

### `identity-model/v4`

| Primitive | File | Line |
|-----------|------|------|
| `CurrentSchemaVersion = 2` | `constants.go` | 64 |
| 21 event type constants | `constants.go` | 14–37 |
| `Upcaster` | `upcaster.go` | 14 |
| `UpcasterRegistry` | `upcaster.go` | 19 |
| `UpcasterRegistry.Register` | `upcaster.go` | 35 |
| `UpcasterRegistry.Upcast` | `upcaster.go` | 52 |
| `MarshalPayload` | `events.go` | 137 |

### `usermgmt/v4`

| Primitive | File | Line |
|-----------|------|------|
| `DefaultEventCatalog()` | `es_event_catalog.go` | 13 |
| `Service.EventCatalog()` | `es_projection_health.go` | 76 |
| `EventSourcedSetup.EventCatalog()` | `es_projection_health.go` | 17 |
| `Service.RebuildProjection` | `es_projection_setup.go` | — |
| `UserReadModel.Name()` | `es_readmodel.go` | 54 |

---

## Appendix B: Standards Quick Reference

| Standard | Version | Role | URL |
|----------|---------|------|-----|
| **CloudEvents** | 1.0 | Per-message wire envelope (CNCF graduated) | https://cloudevents.io |
| **AsyncAPI** | 3.0.x | Event-driven API contract (channels/messages/bindings) | https://www.asyncapi.com |
| **OpenAPI** | 3.1 | HTTP request/response API contract | https://www.openapis.org |
| **DPDS** | 1.x | Data product descriptor (envelope + ports) | https://dpds.opendatamesh.org/ |
| **Bitol Data Contracts** | 1.1.1 | Pragmatic data contract (servers + schema + SLAs + terms) | https://github.com/bitol-io/data-contracts |
| **EventCatalog** | — | Discovery/docs site generator for event-driven architectures | https://www.eventcatalog.dev/ |
| **Confluent Schema Registry** | — | Subject/version schema registry for Kafka | https://docs.confluent.io/platform/current/schema-registry/index.html |
| **Apicurio Registry** | — | Open-source schema registry (Confluent-compatible) | https://www.apicur.io/registry/ |
| **JSON Schema** | 2020-12 | Schema language for JSON payloads | https://json-schema.org/ |
