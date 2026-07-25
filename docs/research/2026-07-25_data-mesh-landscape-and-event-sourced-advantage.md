# Data-Mesh Landscape & The Event-Sourced Advantage

**Date:** 2026-07-25
**Status:** Research synthesis
**Scope:** Industry landscape analysis + strategic positioning for cqrs-htmx / event-sourced systems

---

## TL;DR

The open-source data-mesh ecosystem (DataHub, OpenMetadata, ODCS, EventCatalog, Dagster, OpenLineage, etc.) is solving problems — centralization, stale metadata, undefined "data products," forgotten history, hand-waved governance — that **event sourcing structurally prevents from existing in the first place.** The opportunity isn't to build a better DataHub. It's to recognize that an event-sourced system already _is_ a data mesh, and the only work is **exposure**: transport + documentation + the binding between them.

---

## Table of Contents

- [1. The Open-Source Data-Mesh Landscape](#1-the-open-source-data-mesh-landscape)
- [2. What They Collectively Get Right](#2-what-they-collectively-get-right)
- [3. What They Collectively Get Wrong](#3-what-they-collectively-get-wrong)
- [4. The Event-Sourced Advantage](#4-the-event-sourced-advantage)
- [5. What's Genuinely Missing](#5-whats-genuinely-missing)
- [6. Honest Critique of the Event-Sourced Approach](#6-honest-critique-of-the-event-sourced-approach)
- [7. Strategic Positioning](#7-strategic-positioning)
- [Appendix A: Project Details](#appendix-a-project-details)
- [Appendix B: Data-Mesh Theory Recap](#appendix-b-data-mesh-theory-recap)
- [Appendix C: Sources](#appendix-c-sources)

---

## 1. The Open-Source Data-Mesh Landscape

Grouped by layer in the mesh. Star counts as of mid-2026.

| Category                | Project           | Stars | What it is                                                         |
| ----------------------- | ----------------- | ----- | ------------------------------------------------------------------ |
| **Catalog / Discovery** | OpenMetadata      | 14.6k | Schema-first unified metadata platform (born from Uber's Databook) |
|                         | DataHub           | 12.3k | LinkedIn-born, real-time Kafka push, federated GMS                 |
|                         | Amundsen          | 4.8k  | Lyft-born, usage-ranked "Google for data" (development slowing)    |
|                         | Marquez           | 2.2k  | OpenLineage reference impl, job/dataset/run provenance             |
|                         | Egeria            | 918   | Federated metadata exchange framework (complex, low adoption)      |
| **Data Contracts**      | ODCS (Bitol)      | 1.1k  | The de-facto contract standard (v3.1.0, LF-governed)               |
|                         | data-contract-cli | 958   | Lint/test/enforce contracts in CI/CD (25+ export formats)          |
| **Data Product specs**  | DPDS              | 84    | Open Data Mesh Initiative descriptor spec                          |
|                         | ODPS              | 112   | Bitol product spec                                                 |
|                         | DPROD             | 36    | EKGF product workgroup                                             |
| **Lineage**             | OpenLineage       | 2.6k  | Vendor-neutral lineage standard (Spark/Airflow/dbt)                |
| **Orchestration**       | Dagster           | 15.9k | Asset-oriented (closest to mesh-native)                            |
|                         | Airflow           | 46.2k | Task-centric DAGs (not asset-oriented)                             |
|                         | Mage              | 8.8k  | Visual notebook-style pipeline builder                             |
| **Architecture docs**   | EventCatalog      | 2.8k  | Docs-as-code for event-driven architectures (Astro + MDX)          |
| **Streaming**           | Conduit           | 604   | Go-based Kafka Connect replacement (Meroxa)                        |

**Key truth:** there is no single "data mesh platform." A mesh is assembled from a catalog + contracts + orchestrator + lineage standard. OpenMetadata comes closest to mesh-native; DataHub is the most battle-tested.

---

## 2. What They Collectively Get Right

1. **Real-time metadata propagation** — DataHub's Kafka-based push architecture (metadata changes propagate in seconds, not hours) is genuinely better than legacy pull-based catalogs (Collibra, Alation). The event-driven metadata approach is the right architectural call.

2. **Schema-first design** — OpenMetadata's 700+ JSON Schemas as single source of truth, code-generated into Java/Python/TypeScript. ODCS's dual logical/physical typing (one contract maps to BigQuery/Snowflake/Redshift). Strong typing prevents the "stringly-typed" data swamp.

3. **Asset-oriented thinking** — Dagster's "the asset graph IS your data product catalog" is the most mesh-aligned orchestration model. Declaring data assets as first-class citizens (vs Airflow's task-centric DAGs) maps naturally to "data as a product."

4. **Docs-as-code** — EventCatalog's MDX-in-git with generators that auto-produce entries from AsyncAPI/OpenAPI. PR review applies to architecture docs naturally. No separate "docs database" to keep in sync.

5. **Vendor-neutral standards** — OpenLineage, ODCS, OpenLineage decouple collection from storage. You can switch backends without re-instrumenting.

6. **Battle-tested scale** — DataHub proven at LinkedIn (10M+ assets, O(1B) relationships). OpenMetadata benchmark: 2M assets / 15M relations on Postgres + OpenSearch, reads under ~400ms at 1,200 RPS.

7. **Ecosystem convergence** — datacontract.com deprecated in favor of ODCS (the field is consolidating on one contract standard). Data Contract CLI defaults to ODCS. The fragmentation is decreasing.

---

## 3. What They Collectively Get Wrong

### 3.1 Operational complexity contradicts mesh autonomy

DataHub needs Kafka + Elasticsearch + Neo4j + MySQL + microservices (GMS, frontend, MCE consumer, MAE consumer, upgrade CLI). OpenMetadata needs MySQL/Postgres + OpenSearch + Airflow + the server. Running these **forces a centralized platform team** — recreating the centralization mesh was supposed to eliminate. The "self-serve platform" pillar assumes infrastructure that largely doesn't exist. As one practitioner put it: "You can't fix centralization with a new org chart."

### 3.2 No runtime truth

EventCatalog is docs-only (no live introspection of Kafka clusters or schema registries). Catalogs ingest metadata on schedules (pull-based via Airflow connectors) — they never see **actual message flow**. The catalog describes _intended_ architecture; reality drifts. There's no continuous "what events actually flowed where, when, to whom."

### 3.3 Schema-first but not event-first

They model **tables, datasets, pipelines** — the analytical/warehouse world. None model domain events as first-class. DataHub has a Kafka connector but doesn't model commands, queries, domains, or business flows. The contract formats (ODCS) describe static schemas, not event streams with lifecycle. The mental model is still "I have a table, let me describe it" rather than "I have a stream of domain facts."

### 3.4 "Data product" is still undefined after 6 years

No consensus definition, versioning rules, SLA, or interface. ODCS even _deprecated_ its `dataProduct` field in v3.1.0 (a step _away_ from product thinking). DPDS, ODPS, and DPROD compete on what a product even _is_. A core primitive of the theory has no consensus implementation. We're still arguing over what a "data product" actually is.

### 3.5 Governance is hand-waved

"Computational governance" — policies enforced automatically by the platform — presumes tech that doesn't exist. In practice it degrades to PDFs nobody enforces. The classic failure: "8 domain-specific definitions of revenue." DataHub's domain model has structural limitations: single-domain-per-asset, performance issues with domain-based access control. The docs explicitly warn against using domains as the access boundary.

### 3.6 Databases that forget

Traditional data products overwrite state. No time-travel, no reproducibility, no audit trail without bolting on CDC (Change Data Capture). Lineage is reverse-engineered from logs (OpenLineage), not structural. Snapshots overwrite history. The fundamental data model is lossy.

### 3.7 Fragmented and heavy

Five competing product specs (DPDS, ODPS, DPROD, ODCS, datacontract.com). Three catalog architectures. No interoperability between them. Single-domain-per-asset limits (DataHub). Security CVEs (OpenMetadata 2024 had critical RCE vulnerabilities exploited in the wild for cryptomining). Upgrade pain (OpenMetadata migrations "directly attack your database"). DataCater archived (Aug 2023).

### 3.8 The analytical/operational divide persists

"Orders" exists in the REST API _and_ in the warehouse. They drift. The mesh was supposed to bridge this divide (Dehghani's "great divide of data"); it mostly added a third copy (the "data product"). The ETL labyrinth remains; it just has more stops.

---

## 4. The Event-Sourced Advantage

Event sourcing doesn't just improve the mesh — it **dissolves several of the hard problems** the existing tools are desperately trying to solve.

### The structural advantages you get for free

| Problem traditional meshes struggle with                  | What event sourcing gives you                                                                                                                         |
| --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| "Databases that forget" (no history)                      | The event store **is** the immutable history. Time-travel is a read operation, not a feature to bolt on.                                              |
| Lineage is reverse-engineered from logs                   | Lineage is **structural** — events carry causal metadata, correlation IDs, and reference their causes. No OpenLineage instrumentation needed.         |
| "What's the actual state?" requires querying stale copies | Replay the events to any point. The source of truth is the log, not a projection.                                                                     |
| Data contracts are static documents                       | Events **carry their schema version inline**. The contract is executable — the payload struct IS the schema. Upcasters handle evolution at read time. |
| Real-time requires bolting on CDC/Kafka                   | Real-time is the default. Events propagate as they happen. No CDC, no Kafka Connect pipeline.                                                         |
| Producer shapes don't fit consumer needs                  | Consumers **build their own projections** from raw events. Decoupled by construction.                                                                 |
| "Data product" is undefined                               | The event stream **is** the product. Not a table, not a contract — a versioned, replayable, owned stream of domain facts.                             |
| Analytical/operational divide                             | A single event stream serves both operational (live projections) and analytical (replay/rebuild) consumers.                                           |

### The opportunity: an event-sourced-native mesh

The existing tools are built for the table/warehouse world. An event-sourced environment lets you build something fundamentally simpler and more honest:

**1. The event store is the data product — not a separate artifact.**
Traditional meshes require you to _define_ a data product (a contract), _build_ it (a pipeline), _publish_ it (to a catalog), and _maintain_ it (versioning, SLAs). In ES, the event stream already exists — it's the write model. The "product" is just making it discoverable and consumable. This collapses the entire "data product lifecycle" into "expose what you already have."

**2. The catalog is auto-generated from the code.**
No manual authoring, no ingestion jobs. The event types, their payloads, their schemas — all derivable from the Go types via reflection (which `catalog/v4`'s `SchemaFromType[T]()` already does). The catalog is a build artifact, not a database to keep in sync. When you change an event struct, the catalog updates on the next build. Zero drift.

**3. Contracts are executable, not documentary.**
ODCS is a YAML file describing a schema. An event-sourced contract is the **compiled binary** — the struct definition, the upcaster chain, the fold function. You can't ship an event that violates its contract because the contract _is the code_. The `UpcasterRegistry` handles backward compat automatically; no migration scripts.

**4. Lineage is free and exact.**
Every event carries: its causation ID (which command produced it), its correlation ID (which request), its aggregate ID, its schema version, its timestamp. You don't need OpenLineage — you need a query over the event store. Cross-domain lineage: when domain A's projection consumes domain B's events, the subscription itself is the lineage edge, recorded in the checkpoint store.

**5. Projections are the read models consumers subscribe to.**
Instead of "each consumer negotiates a table shape," each consumer builds a projection from the shared event stream. The `projectionhost` already manages this: per-projection checkpoints, crash-restart, DLQ. This is the mesh's "self-serve" pillar, already implemented — consumers pick their own storage, their own shape, their own pace.

**6. Federation is natural.**
Each domain owns its event store + its projections + its catalog slice. Federation = "each domain serves its events at a URL." No central catalog database. The `docserver` from `catalog/v4` already serves OpenAPI + AsyncAPI + D2 from one handler per domain. Cross-domain discovery is just an index of URLs.

---

## 5. What's Genuinely Missing

The research confirms that `catalog/v4` + `transport/http/v4` + event sourcing already cover 80% of what the mesh tools are desperately trying to build. The gaps:

1. **Channel-to-runtime binding** — connect `catalog.Channel` (documentation) to `event.Bus`/`StreamingJournal` (runtime). No existing tool does this because they don't own both sides. ~50 LOC.

2. **CloudEvents framing** — `transport/http/v4.SSEBroker` already has `payloadTransform`. A CloudEvents wrapper is one function. Makes events interoperable with EventBridge/Knative/any router. ~30 LOC.

3. **Pull-based machine transport** — `GET /events?after=<id>` returning JSON/NDJSON. No existing tool provides this (SSEBroker is push-only; JournalSSEStore is SSE-framed). ~100 LOC.

4. **Time-travel as a first-class mesh operation** — "rebuild this data product as of 2024-01-15." The event store supports it; no mesh tool exposes it. This is the killer feature traditional meshes _can't_ offer.

5. **The narrative** — the insight that in ES, you don't need DataHub/OpenMetadata/ODCS. You need: expose your events (transport), document them (catalog), and let consumers build projections. The "data product" is the event stream itself. This needs to be written down clearly, because the industry is still stuck in the table/warehouse mental model.

---

## 6. Honest Critique of the Event-Sourced Approach

It's not all upside. Intellectual honesty demands acknowledging the trade-offs:

- **Not every consumer wants events.** Some want pre-shaped tables, REST APIs, or files. EventCatalog (Adam Bellemare's "Building an Event-Driven Data Mesh") acknowledges geolocation/search and batch aggregations may be better served synchronously. You still need sink connectors to feed BI tools that expect tables.

- **Eventual consistency is harder than it looks.** Consumers must handle out-of-order events, idempotency, schema evolution. The mesh promises "just subscribe"; the reality is projection logic is non-trivial. The projection must be deterministic and replayable — a real engineering constraint.

- **The ecosystem is smaller.** DataHub has 80+ connectors; the ES world has fewer integration points. BI tools (Tableau, Looker, PowerBI) expect tables, not event streams. You'll need to bridge.

- **Scale of the event store.** Replaying years of events is expensive without snapshots. The `SnapshotConfig` helps but adds complexity. The event store grows forever; retention/cleanup policies are a real operational concern.

- **Schema evolution is still hard.** Upcasters handle backward compat, but they're code you must write and maintain. A badly designed upcaster can corrupt the read model. The contract-is-code approach is powerful but unforgiving.

---

## 7. Strategic Positioning

### The core insight

The existing data-mesh tools are solving problems (centralization, stale metadata, undefined products, forgotten history) that event sourcing **structurally prevents from existing in the first place.**

The opportunity isn't to build a better DataHub — it's to recognize that an event-sourced system already _is_ a data mesh, and the only work is **exposure** (transport + documentation + the binding between them).

### What cqrs-htmx / go-cqrs-lite should do

1. **Expose `catalog/v4` as the recommended path** for data-mesh contracts. Deprecate the hand-rolled `EventCatalog`. The catalog is auto-generated from Go types — zero-drift by construction.

2. **Build the 3 missing pieces** (~180 LOC total):
   - Channel-to-Bus binding
   - CloudEvents `payloadTransform` function
   - Pull-based `EventStreamHandler`

3. **Adopt `transport/http/v4.SSEBroker`** as the push transport (it already has filtering, dedup, metrics, OTel — everything cqrs-htmx's `Broadcaster` lacks).

4. **Document the event-sourced mesh pattern** — a guide showing that in an ES world, the "data product" is the event stream, the "catalog" is auto-generated, the "contract" is executable, and the "lineage" is structural. This is the narrative the industry is missing.

5. **Position time-travel as the killer feature.** No traditional mesh tool can offer "rebuild this data product as of any point in time." Event sourcing can. This is the differentiator that makes the ES-native mesh fundamentally better, not just simpler.

### What NOT to do

- Don't build a central catalog database (that's DataHub/OpenMetadata's mistake).
- Don't build a contract YAML format (that's ODCS's territory — and the code IS the contract).
- Don't build an ingestion framework (events are already the ingestion).
- Don't try to be a "data mesh platform" (that's a socio-technical org change, not a library).

---

## Appendix A: Project Details

### Catalogs / Discovery Platforms

#### OpenMetadata (14.6k stars)

- **Origin:** Built by the team that created Uber's metadata platform _Databook_.
- **Architecture:** Schema-first (JSON Schemas as source of truth, code-generated into Java/Python/TypeScript). REST API server (Dropwizard/Jetty). Backend: MySQL 8.x or PostgreSQL. Search: Elasticsearch/OpenSearch. No dedicated graph DB (uses `entity_relationship` table). Ingestion: Python framework with 130+ connectors, typically orchestrated by Airflow (pull-based).
- **Gets right:** Genuine breadth in one platform (catalog + DQ + profiling + lineage + governance + contracts + search). Built-in data quality (30-40+ test types — DataHub has none). Open standards alignment (ODCS, DCAT/DPROD, PROV-O, JSON-LD/SHACL). Proven scale (2M assets / 15M relations benchmark). MCP server + AI features.
- **Gets wrong:** Centralized, not federated (single store, no native federated metadata services — a structural mesh mismatch). Pull-based/Airflow-centric ingestion (weaker real-time than DataHub). No native graph DB/GraphQL. Self-hosting operational burden. Upgrades are a known pain point. Critical CVEs in 2024 (CVE-2024-28255 etc., RCE, actively exploited for cryptomining). OSS vs commercial (Collate) feature gap.

#### DataHub (12.3k stars)

- **Origin:** Built at LinkedIn (evolved from WhereHows), commercialized by Acryl Data (now DataHub Inc.).
- **Architecture:** Schema-first metadata modeling (Pegasus PDL). Stream-based real-time metadata (Kafka — changes in seconds). Federated metadata serving (multiple GMS owned by different teams). Stack: MySQL/Postgres + Elasticsearch + Neo4j + Kafka + microservices (GMS, frontend, MCE consumer, MAE consumer).
- **Gets right:** Battle-tested at hyperscale (LinkedIn, Netflix, Pinterest, 10M+ assets). Push-based real-time architecture (innovative, better than pull-based). Extensible metadata model (PDL aspects). 80+ production-grade connectors. Active commercial stewardship. Federated metadata serving (architecturally supports mesh). Explicit Domains + data products + data contracts.
- **Gets wrong:** Operational complexity (4 infrastructure deps + microservices — the #1 criticism). No native data quality profiling (requires Great Expectations/dbt). Lineage gaps (column-level not supported across all platforms). Governance not fully mature. Data contracts limited to datasets only. Single-domain-per-asset limitation. Domain-based access control can hurt performance. Federated serving not first-class in open source.

#### Amundsen (4.8k stars)

- **Origin:** Built at Lyft. Hosted by LF AI & Data Foundation.
- **Architecture:** Neo4j or Apache Atlas backend + Elasticsearch search.
- **Gets right:** Usage-based ranking (page-rank-style search where frequently queried tables rank higher). Simpler and more focused for pure discovery.
- **Gets wrong:** Development has slowed. Lacks data quality, observability, governance, data contracts. Architecturally dated (more moving parts than monolithic alternatives). Not suited as a comprehensive mesh platform.

#### Marquez (2.2k stars)

- **Origin:** Open-sourced by WeWork. LF AI & Data Graduate project. Reference implementation of OpenLineage API.
- **Gets right:** Tracks data lineage (job/dataset/run provenance). Dataset lifecycle management. Integrates with OpenLineage.
- **Gets wrong:** Narrower scope (lineage-focused, not a full catalog). Smaller community.

#### Egeria (918 stars)

- **Origin:** odpi / Linux Foundation / ODPi.
- **Gets right:** Deep governance focus. Federated metadata architecture (designed for exchanging metadata between heterogeneous platforms).
- **Gets wrong:** Steep complexity (Java-heavy, enterprise-oriented). Low community adoption. Framework/standard rather than ready-to-use product.

### Data Contract Tools

#### ODCS — Open Data Contract Standard (1.1k stars)

- **Origin:** PayPal internal template, open-sourced, now governed by Bitol under LF AI & Data Foundation. v3.1.0 (Dec 2025). Media type: `application/odcs+yaml;version=3.1.0`.
- **Sections:** Fundamentals (`apiVersion`, `kind`, `id`, `name`, `version`, `status`, `domain`, `tenant`, `description`), `schema` (dual logical/physical typing), `references` (foreign keys, v3.1.0), `servers` (30+ types), `slaProperties` (flat array), `quality` (library/sql/text/custom), `team`, `roles`, `support`, `price`, `customProperties`, `authoritativeDefinitions`.
- **Gets right:** Dual logical/physical typing (one contract, multiple warehouses). Broad lifecycle-spanning coverage. Platform/vendor-neutral. Library-based quality metrics (v3.1.0). Extensibility (`customProperties` everywhere). Linux Foundation governance. Ecosystem convergence (datacontract.com deprecated in its favor).
- **Gets wrong:** Scope creep and naming collisions (ODPS acronym clash). Schema is array-based, not map-based (fragile referencing until v3.1.0 retrofitted `id`s). No native DRY type-reuse/`$ref`. `terms` is fragmented across `description`/`price`/`authoritativeDefinitions`. Enterprise baggage/verbosity. Deprecation churn (v3.0.0 was a large breaking rewrite). SLA model relies on external Medium article for taxonomy. Contract-first, not product-first (no input/output port modeling, no discoverability).

#### data-contract-cli (958 stars)

- **Origin:** datacontract organization, created by Stefan Negele, Jochen Christ, Simon Harrer (Entropy Data / INNOQ).
- **Gets right:** End-to-end contract enforcement (lint, test against live data sources — Postgres, Snowflake, BigQuery, Kafka, Databricks). Excellent CI/CD integration. Rich import/export (25+ formats). dbt integration. Now defaults to ODCS.
- **Gets wrong:** Python-based with complex optional dependencies. Niche tool requiring organizational buy-in. No built-in UI.

### Data Product Specifications

- **DPDS** (84 stars) — Open Data Mesh Initiative descriptor spec. Full envelope: `info`, `interface`, `components` (inputPorts, outputPorts, discoveryPorts, observabilityPorts). Output port `promises` point at OpenAPI/AsyncAPI documents.
- **ODPS** (112 stars) — Bitol product spec.
- **DPROD** (36 stars) — EKGF Data Product Workgroup.

All three compete on what a "data product" even is. No consensus after 6 years.

### Lineage

#### OpenLineage (2.6k stars)

- **Origin:** Launched by Datakin/WeWork. LF AI & Data Graduate project.
- **Gets right:** Vendor-neutral standard for lineage collection. Supported by Spark, Airflow, dbt, Flink. Table-level and column-level lineage. Decouples collection from storage.
- **Gets wrong:** Standard + client libraries, not a full lineage platform. Integration coverage uneven. Requires modifying pipeline tools to emit events.

### Orchestration

#### Dagster (15.9k stars)

- Asset-oriented (closest to mesh-native). The asset graph IS your data product catalog. Integrated lineage and observability. Best-in-class testability.
- **Limitation:** Newer paradigm; enterprise features behind Dagster+.

#### Airflow (46.2k stars)

- Industry standard. Massive ecosystem. Task-centric (not asset-oriented — harder to reason about data products). Steep operational overhead. No native lineage/quality/observability.

#### Mage (8.8k stars)

- Visual notebook-style pipeline builder. Best developer experience for local development. Enterprise features gated.

### Architecture Documentation

#### EventCatalog (2.8k stars)

- **Origin:** Created by David Boyne (v1 launched January 2022). 40,000+ catalogs created. Adopted by Nike, AWS, GOV.UK, Eurostar, NHS, Worldpay, Ticketmaster, M&S.
- **Architecture:** Astro 5.x + React 18. Content model: MDX files in git (docs-as-code). Visualization: `@xyflow/react` node graphs. Static site generator (no database, no broker). AI chat + MCP server (v4, premium).
- **Gets right:** Documentation-as-code done right (PR review, version history apply to architecture docs). Architecture-native (domains, services, events, commands, queries, flows, ADRs). Strong visualization. Automation via generators (AsyncAPI, OpenAPI, Kafka, Schema Registry). Schema field-level search. AI/MCP integration. Low operational overhead (static site).
- **Gets wrong:** Enterprise features not open source (AI chat, MCP server, SSO, RBAC, schema governance — commercial license). No runtime metadata ingestion (docs tool, not discovery tool). No actual data lineage (intended relationships only). Manual relationship modeling required. JavaScript ecosystem lock-in. Scalability of flat MDX directory at very large scale.

### Streaming

#### Conduit (604 stars)

- Go-based Kafka Connect replacement (Meroxa). No JVM required. gRPC plugin protocol. Single binary.
- Not explicitly designed as a data mesh tool.

---

## Appendix B: Data-Mesh Theory Recap

### The problem (Dehghani's diagnosis, 2019)

The central data team as a bottleneck. The "great divide of data" between operational data (transactional, behind microservices) and analytical data (warehouses/lakes), bridged by "a labyrinth of data pipelines" and "continuously failing ETL jobs." Domain teams surrender data upward and "get crumbs of value in return."

### The four pillars

1. **Domain-oriented decentralized data ownership** — Decompose analytical data along business domain boundaries. The team that owns the operational system also owns and serves its analytical data.
2. **Data as a product** — Domain data must be treated as a first-class product with discoverability, understandability, trustworthiness, security.
3. **Self-serve data infrastructure as a platform** — A platform team provides abstractions so domain teams can build/deploy/run data products autonomously.
4. **Federated computational governance** — Global interoperability standards defined by a federation; policies enforced automatically by the platform.

### Why implementations fail or stall

- **It's an organizational change, not a technology you can buy.** Transformation cost and culture are the real battle.
- **Scale mismatch.** Below ~1,000 employees, the central data team is faster than coordinating across domains.
- **The org-chart shuffle.** Reorgs happen, but bottlenecks just move around.
- **Ownership ambiguity.** Transformed/derived data ownership is genuinely unclear.
- **Programs drag and get killed.** Realistic timelines: 18-30 months. Most are killed before realizing benefits.
- **Gartner flagged it as "obsolete before the plateau of productivity."**
- **Governance is the hardest part and is mostly hand-waved.** Without the federated governance plane, "mesh becomes a polite name for siloed data."

---

## Appendix C: Sources

### Project repositories

- OpenMetadata: https://github.com/open-metadata/OpenMetadata
- DataHub: https://github.com/datahub-project/datahub
- Amundsen: https://github.com/amundsen-io/amundsen
- Marquez: https://github.com/MarquezProject/marquez
- Egeria: https://github.com/odpi/egeria
- ODCS: https://github.com/bitol-io/open-data-contract-standard
- data-contract-cli: https://github.com/datacontract/datacontract-cli
- DPDS: https://github.com/opendatamesh-initiative/odm-specification-dpdescriptor
- ODPS: https://github.com/bitol-io/open-data-product-standard
- DPROD: https://github.com/EKGF/dprod
- OpenLineage: https://github.com/OpenLineage/OpenLineage
- Dagster: https://github.com/dagster-io/dagster
- Airflow: https://github.com/apache/airflow
- Mage: https://github.com/mage-ai/mage-ai
- EventCatalog: https://github.com/event-catalog/eventcatalog
- Conduit: https://github.com/ConduitIO/conduit

### Analysis and criticism

- Zhamak Dehghani, "Data Mesh Principles" (Martin Fowler): https://martinfowler.com/articles/data-mesh-principles.html
- Data Mesh Architecture: https://www.datamesh-architecture.com/
- Thinklytics, "Data Mesh in Practice" (failure modes, scale thresholds): https://thinklytics.com/insights/data-mesh-in-practice
- Jenny Kwan, "Data Mesh Foundation" (tooling fights decentralization, undefined data product): https://jennykwan.org/posts/data-mesh-foundation-part-1/
- Andriy Zabavskyy, "Data Mesh Pain Points" (Inmon/Kimball déjà vu, shared dimensions): https://towardsdatascience.com/data-mesh-pain-points-b4bebca37357/
- Andrew Jones, "Challenges of Implementing Data Mesh" (Gartner, transformation failures): https://andrew-jones.com/newsletter/2025-05-09-the-challenges-of-implementing-a-data-mesh/
- Xomnia, "Risks and Disadvantages of Data Mesh" (transformation costs, ownership blame ping-pong): https://xomnia.com/post/what-are-the-risks-and-disadvantages-of-data-mesh/

### Event-sourced / event-driven mesh

- Adam Bellemare, "Building an Event-Driven Data Mesh" (O'Reilly, 2023): https://www.oreilly.com/library/view/building-an-event-driven/9781098127596/
- Adam Bellemare, "The Definitive Guide to Building a Data Mesh with Event Streams": https://dev.to/bellemare/the-definitive-guide-to-building-a-data-mesh-with-event-streams-207b
- eventsourcing.ai, Data Mesh concept: https://www.eventsourcing.ai/concepts/data-mesh/
- Confluent, "Data Mesh Architectures with Event Streams" (ebook): https://www.confluent.io/resources/ebook/data-mesh-architectures-with-event-streams/
- Confluent Data Mesh Demo: https://github.com/confluentinc/data-mesh-demo
- AWS, "Event-driven architecture to build a data mesh on AWS": https://aws.amazon.com/blogs/big-data/use-an-event-driven-architecture-to-build-a-data-mesh-on-aws/

### Architecture comparisons

- TextQL DataHub Wiki: https://www.textql.com/wiki/datahub
- Atlan, "OpenMetadata vs DataHub": https://atlan.com/openmetadata-vs-datahub/
- DeepWiki, DataHub architecture: https://deepwiki.com/acryldata/datahub
- DeepWiki, OpenMetadata helm charts ops: https://deepwiki.com/open-metadata/openmetadata-helm-charts/5-operations-and-maintenance
- Microsoft, OpenMetadata CVE exploitation: https://www.microsoft.com/en-us/security/blog/2024/04/17/attackers-exploiting-new-critical-openmetadata-vulnerabilities-on-kubernetes-clusters/
- Saxo Bank data mesh journey (DataHub): https://medium.com/datahub-project/enabling-data-discovery-in-a-data-mesh-the-saxo-journey-451b06969c8f
- Data Mesh Governance Challenges: https://sgkg.org/blog/2026-03-17-data-mesh-governance-challenges/
