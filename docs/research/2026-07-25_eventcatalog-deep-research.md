# EventCatalog Deep Research

**Date:** 2026-07-25
**Status:** Research analysis
**Subject:** eventcatalog.dev by David Boyne — architecture, content model, SDK, generators, and strategic comparison with go-cqrs-lite `catalog/v4`

---

## TL;DR

EventCatalog is the most mature open-source **architecture documentation tool** for event-driven systems. Its content model (domains, services, events, commands, queries, channels, flows, entities, agents, data products, ADRs, teams, users) is richer than go-cqrs-lite's `catalog/v4` in several dimensions: versioning-with-preservation, changelogs, flows with branching, the SDK (150+ functions), the generators ecosystem, MCP/AI integration, and a linter. But it's a **documentation tool, not a runtime tool** — it describes architecture from external specs (AsyncAPI/OpenAPI), not from running code. catalog/v4's strength is the opposite: the model is **derived from Go types at build time** (zero drift), and it has structural integration with the event-sourced runtime. The two are complementary, not competitive — and `catalog/v4` already has an EventCatalog exporter.

---

## Table of Contents

- [1. Architecture](#1-architecture)
- [2. Content Model (Complete Type System)](#2-content-model-complete-type-system)
- [3. The SDK (150+ Functions)](#3-the-sdk-150-functions)
- [4. The Generators System](#4-the-generators-system)
- [5. Versioning & Changelogs](#5-versioning--changelogs)
- [6. MCP / AI Integration](#6-mcp--ai-integration)
- [7. CLI, Linter, Connectors](#7-cli-linter-connectors)
- [8. What EventCatalog Gets Right (That We Should Learn From)](#8-what-eventcatalog-gets-right-that-we-should-learn-from)
- [9. What catalog/v4 Gets Right (That EventCatalog Can't)](#9-what-catalogv4-gets-right-that-eventcatalog-cant)
- [10. Side-by-Side Comparison](#10-side-by-side-comparison)
- [11. Recommendations](#11-recommendations)

---

## 1. Architecture

EventCatalog is a **static-site generator** built on **Astro 5.x + React 18** with an **island architecture**. Your catalog is a folder of MDX files in git. Astro compiles these into a navigable documentation site.

```
┌─────────────────────────────────────────────────────┐
│  External Data Sources                               │
│  (AsyncAPI, OpenAPI, Schema Registry, EventBridge,   │
│   Kafka, your code)                                   │
└──────────────────────┬──────────────────────────────┘
                       │  parsed by generators
                       ▼
┌─────────────────────────────────────────────────────┐
│  GENERATORS (separate pnpm packages)                  │
│  • @eventcatalog/generator-asyncapi                  │
│  • @eventcatalog/generator-openapi                   │
│  • @eventcatalog/generator-confluent-schema-registry │
│  • custom plugin.js                                  │
└──────────────────────┬──────────────────────────────┘
                       │  calls SDK functions
                       ▼
┌─────────────────────────────────────────────────────┐
│  @eventcatalog/sdk (the data layer)                  │
│  writeEvent(), writeService(), writeDomain(), ...    │
│  150+ functions over the MDX file system             │
└──────────────────────┬──────────────────────────────┘
                       │  writes MDX + schema files
                       ▼
┌─────────────────────────────────────────────────────┐
│  EventCatalog MDX Files (in git)                     │
│  domains/*/index.mdx, events/*/index.mdx, etc.       │
└──────────────────────┬──────────────────────────────┘
                       │  Astro builds
                       ▼
┌─────────────────────────────────────────────────────┐
│  Static Documentation Site                           │
│  (or SSR for AI chat / MCP / auth / editing)         │
└─────────────────────────────────────────────────────┘
```

**Key architectural decisions:**

- **No database.** The file system IS the database. MDX frontmatter is the schema.
- **No message broker.** Static by default; SSR only for premium AI features.
- **Docs-as-code.** PR review, version history, branching all apply to architecture docs.
- **Generators are build-time.** They run before Astro builds, writing MDX files to disk.
- **Monorepo.** `packages/core` (Astro app), `packages/sdk` (data layer), `packages/cli`, `packages/linter`, `packages/connectors`, `packages/visualiser`, `packages/breaking-changes`, `packages/create-eventcatalog`. Generators live in a separate repo (`event-catalog/generators`).

**Monorepo package inventory:**

| Package                             | Purpose                                    |
| ----------------------------------- | ------------------------------------------ |
| `@eventcatalog/core`                | The Astro app + build pipeline             |
| `@eventcatalog/sdk`                 | 150+ function data layer over MDX files    |
| `@eventcatalog/cli`                 | Terminal access to SDK functions           |
| `@eventcatalog/linter`              | Zod-based frontmatter validation           |
| `@eventcatalog/connectors`          | GitHub / Microsoft Entra ID directory sync |
| `@eventcatalog/visualiser`          | Standalone React graph component           |
| `@eventcatalog/breaking-changes`    | Breaking change detection between versions |
| `@eventcatalog/create-eventcatalog` | Scaffolding CLI                            |

---

## 2. Content Model (Complete Type System)

Every resource is an `index.mdx` file (or flat `.mdx` for users/teams) with YAML frontmatter. All resources share a `BaseSchema` with `id`, `name`, `version`, `summary`, `badges`, `owners`, `repository`, `deprecated`, `styles`, `attachments`, `diagrams`, `draft`, `schemaPath`, `schemas`, `resourceGroups`.

### Resource types

| Type             | Directory        | Key fields beyond BaseSchema                                                                                                                                                     |
| ---------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Domain**       | `domains/`       | `systems[]`, `services[]`, `agents[]`, `domains[]` (subdomains), `entities[]`, `flows[]`, `dataProducts[]`, `sends[]`, `receives[]`, `ubiquitousLanguage[]`                      |
| **Service**      | `services/`      | `sends[]`, `receives[]`, `entities[]`, `writesTo[]`, `readsFrom[]`, `flows[]`, `externalSystem`, `specifications[]`                                                              |
| **Event**        | `events/`        | `channels[]`, `operation`                                                                                                                                                        |
| **Command**      | `commands/`      | `channels[]`, `operation`                                                                                                                                                        |
| **Query**        | `queries/`       | `channels[]`, `operation`                                                                                                                                                        |
| **Channel**      | `channels/`      | `address`, `protocols[]`, `deliveryGuarantee`, `routes[]`, `parameters{}`                                                                                                        |
| **Entity**       | `entities/`      | `aggregateRoot`, `identifier`, `properties[]` (nested, with references + relationType)                                                                                           |
| **Flow**         | `flows/`         | `steps[]` (each: service/message/agent/actor/custom/externalSystem, with `next_step`/`next_steps` for branching)                                                                 |
| **System**       | `systems/`       | `scope` (internal/external), `services[]`, `flows[]`, `entities[]`, `containers[]`, `relationships[]`, `actors[]`                                                                |
| **Agent**        | `agents/`        | `sends[]`, `receives[]`, `writesTo[]`, `readsFrom[]`, `flows[]`, `model` (provider/name/version), `tools[]`                                                                      |
| **Container**    | `containers/`    | `container_type` (database/cache/objectStore/searchIndex/dataWarehouse/dataLake/externalSaaS/other), `technology`, `authoritative`, `classification`, `retention`                |
| **Data Product** | `data-products/` | `inputs[]`, `outputs[]` (each with optional `contract: {path, name, type}`)                                                                                                      |
| **ADR**          | `adrs/`          | `status` (proposed/accepted/rejected/deprecated/superseded), `date`, `decisionMakers[]`, `appliesTo[]`, `supersedes[]`, `supersededBy[]`, `amends[]`, `amendedBy[]`, `related[]` |
| **User**         | `users/`         | `avatarUrl`, `role`, `email`, `slackDirectMessageUrl`, `source` (provider/id/url)                                                                                                |
| **Team**         | `teams/`         | `members[]`, `email`, `slackDirectMessageUrl`                                                                                                                                    |
| **Diagram**      | `diagrams/`      | (file-based, C4/Mermaid/LikeC4)                                                                                                                                                  |
| **Custom Doc**   | `docs/`          | (freeform pages)                                                                                                                                                                 |

### Relationship model — `sends` / `receives` pointers

Relationships between services/agents and messages are **bidirectional pointers** declared on the service/agent side:

```typescript
type SendsPointer = {
	id: string; // message ID
	version?: string; // semver, supports ranges (^1.0.0, ~1.2.0, 0.0.x, latest)
	fields?: string[]; // specific fields this service touches
	to?: ChannelPointer[]; // which channels this message goes through
	group?: string; // visual grouping
};

type ReceivesPointer = {
	id: string;
	version?: string;
	fields?: string[];
	from?: ChannelPointer[];
	group?: string;
	triggers?: TriggerPointer[]; // conditional consumption: { id, version?, condition? }
};
```

The reverse lookup (finding which services produce/consume a given message) is computed at build time by the graph engine.

### Entity property model

```typescript
interface EntityProperty {
	name: string;
	type: string;
	required?: boolean;
	description?: string;
	references?: string; // FK to another entity
	referenceTarget?: "entity";
	relationType?: string; // belongsTo, hasMany, etc.
	enum?: string[];
	properties?: EntityProperty[]; // nested objects
	items?: { type: string; properties?: EntityProperty[] }; // arrays
}
```

This is a richer entity model than catalog/v4's `Entity` which has `Properties []EntityProperty` but no nested types or relation types.

### Data Product model

```typescript
interface DataProduct extends BaseSchema {
	inputs?: ResourcePointer[];
	outputs?: {
		id: string;
		version?: string;
		contract?: { path: string; name?: string; type?: string };
	}[];
}
```

This matches DPDS/ODPS structure — input ports, output ports with contracts. catalog/v4's `DataProduct` is simpler: `Inputs []Ref`, `Outputs []DataProductOutput` where `DataProductOutput` embeds `Ref` + `Contract *DataContract`.

---

## 3. The SDK (150+ Functions)

The SDK (`@eventcatalog/sdk`) is the data layer — a single default function that takes a catalog path and returns an object with ~150+ methods.

```typescript
import utils from "@eventcatalog/sdk";
const catalog = utils("/path/to/eventcatalog");
```

### CRUD pattern (consistent across all resource types)

| Operation                | Naming convention                   | Example                                    |
| ------------------------ | ----------------------------------- | ------------------------------------------ |
| Read one                 | `get<Resource>(id, version?)`       | `getEvent('OrderCreated', '1.0.0')`        |
| Read all                 | `get<Resources>(opts?)`             | `getEvents()`                              |
| Write (create or update) | `write<Resource>(data, opts?)`      | `writeEvent({...}, { override: true })`    |
| Delete by path           | `rm<Resource>(path)`                | `rmEvent('events/OrderCreated')`           |
| Delete by ID             | `rm<Resource>ById(id)`              | `rmEventById('OrderCreated')`              |
| Version                  | `version<Resource>(id)`             | `versionEvent('OrderCreated')`             |
| Check version            | `<resource>HasVersion(id, version)` | `eventHasVersion('OrderCreated', '1.0.0')` |

### Resource types with full CRUD

Events, Commands, Queries, Services, Domains, Systems, Agents, Channels, Flows, Entities, Containers, Data Products, Diagrams, ADRs, Teams, Users, Custom Docs.

### Relationship management

```typescript
// Add a message to a service (sends or receives)
addEventToService(serviceId, "sends" | "receives", { event: "OrderCreated", version: "2.0.0" });
addCommandToService(serviceId, "sends" | "receives", { command: "PlaceOrder", version: "1.0.0" });
addQueryToService(serviceId, "sends" | "receives", { query: "GetOrder", version: "1.0.0" });

// Add a service to a domain
addServiceToDomain(domainId, { id: "OrderService", version: "1.0.0" });

// Add a subdomain
addSubDomainToDomain(domainId, { id: "Checkout", version: "1.0.0" });

// Add an entity to a service
addEntityToService(serviceId, { id: "Order", version: "1.0.0" });

// Add a data store to a service
addDataStoreToService(serviceId, "writesTo" | "readsFrom", { id: "orders-db", version: "2.0.0" });

// Add a data product to a domain
addDataProductToDomain(domainId, { id: "orders-stream", version: "1.0.0" });

// Add ubiquitous language to a domain
addUbiquitousLanguageToDomain(domainId, dictionary);
```

### Schema and file management

```typescript
// Attach a JSON schema to an event
addSchemaToEvent("OrderCreated", { fileName: "schema.json", content: "{...}" });

// Attach an example payload
addExampleToEvent("OrderCreated", { fileName: "example.json", content: "{...}" });

// Attach arbitrary file
addFileToEvent("OrderCreated", { fileName: "migration.md", content: "..." });
```

### Graph and analysis

```typescript
// Get the relationship graph
getGraph(root, options);

// Who produces/consumes this message?
getProducersAndConsumersForMessage("OrderCreated", "1.0.0");
// → { producers: [...], consumers: [...] }

// Who produces/consumes this schema?
getProducersOfSchema("/path/to/schema.json");
getConsumersOfSchema("/path/to/schema.json");

// Find a message by schema
getSchemaForMessage("OrderCreated", "1.0.0");
getMessageBySchemaPath("/path/to/schema.json");

// Who owns this resource?
getOwnersForResource("OrderCreated", "1.0.0");
```

### Snapshots and diffing

```typescript
createSnapshot();
diffSnapshots(snapshot1, snapshot2);
listSnapshots();
```

### Changelogs

```typescript
writeChangelog(resourceId, changelog, { version?, format? })
appendChangelog(resourceId, changelog, opts)  // creates if missing
getChangelog(resourceId, { version? })
rmChangelog(resourceId, opts)
```

### FlowBuilder (fluent API)

```typescript
import { FlowBuilder } from "@eventcatalog/sdk";

const flow = new FlowBuilder({ id: "OrderFlow", name: "Order Process", version: "0.0.1" });
flow
	.addServiceStep({ id: "OrdersService", title: "Order Service" })
	.addMessageStep({ id: "OrderPlaced", title: "Order Placed" })
	.addActorStep({ title: "Customer" })
	.build();
```

### dumpCatalog

```typescript
// Dump entire catalog to JSON
dumpCatalog(directory);
```

**Assessment:** The SDK is the most complete CRUD layer for a documentation system I've seen. It makes programmatic catalog generation trivial — which is exactly what a code-to-catalog pipeline needs. catalog/v4 has no equivalent; it has Go builders (`simple.New()`, `catalog.Event[T]()`) but no read/delete/version/relationship management API.

---

## 4. The Generators System

Generators are **build-time scripts** that consume external specs and produce catalog content via the SDK.

### How generators work

```javascript
// eventcatalog.config.js
generators: [
  ['@eventcatalog/generator-asyncapi', {
    services: [
      { path: './asyncapi-files/orders.yml', id: 'Orders Service' },
      { path: 'https://raw.githubusercontent.com/.../payments.yml', id: 'Payments Service' },
    ],
    domain: { id: 'orders', name: 'Orders', version: '0.0.1' },
    parseChannels: true,
  }],
  ['@eventcatalog/generator-openapi', {
    services: [{ path: './openapi/petstore.yml', id: 'pet-store' }],
  }],
],
```

Run with `pnpm run generate` → each generator calls SDK functions (`writeEventToService`, `writeServiceToDomain`) to create MDX files on disk.

### AsyncAPI generator — mapping

| AsyncAPI concept                          | EventCatalog resource                      |
| ----------------------------------------- | ------------------------------------------ |
| `info.title` + `info.version`             | Service                                    |
| `operations` with `action: send`          | Service `sends[]`                          |
| `operations` with `action: receive`       | Service `receives[]`                       |
| `components/messages/*`                   | Event, Command, or Query (default: event)  |
| `channels/*` (when `parseChannels: true`) | Channel                                    |
| Config `domain: {id, name, version}`      | Domain                                     |
| `info.version`                            | Default version for service + all messages |

### The `x-eventcatalog-*` extensions (clever design)

These AsyncAPI extensions let you control catalog behavior from within the spec itself:

| Extension                        | Values                          | Effect                                                                                                                                       |
| -------------------------------- | ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `x-eventcatalog-message-type`    | `event` \| `command` \| `query` | Which EventCatalog directory the message lands in. Default: `event`                                                                          |
| `x-eventcatalog-message-version` | semver string                   | Per-message version override (default: inherits `info.version`)                                                                              |
| `x-eventcatalog-role`            | `provider` \| `client`          | `provider` = this service owns the message (creates docs). `client` = external contract (records relationship only, no message docs created) |
| `x-eventcatalog-draft`           | `true`                          | Marks message/service as draft                                                                                                               |
| `x-eventcatalog-deprecated-date` | date string                     | Deprecation banner                                                                                                                           |
| `x-eventcatalog-group`           | string                          | Visual grouping in the graph                                                                                                                 |

**The `x-eventcatalog-role: provider|client` pattern is particularly well-designed.** It solves the "who owns this message?" problem in multi-service architectures. When service A produces `OrderCreated` and service B consumes it, both might list it in their AsyncAPI specs. Without the role extension, the generator would create duplicate message docs. With `provider`/`client`, only the provider creates the message; the client gets the `receives` relationship without duplicating docs.

---

## 5. Versioning & Changelogs

This is EventCatalog's strongest feature and the area where catalog/v4 is weakest.

### Versioning

Every versioned resource supports semver (`version: 1.0.0` in frontmatter). The `versionResource(id)` function moves the current resource into a `versioned/{version}/` subdirectory, preserving history.

Version references support:

- Exact: `version: 1.0.0`
- `latest` keyword
- Semver ranges: `^1.0.0`, `~1.2.0`
- X-patterns: `0.0.x`, `1.x`

The graph engine resolves version pointers using `semver.maxSatisfying`.

### Generator versioning-on-re-run (the killer feature)

When a generator re-runs with a new version:

1. **Old version is snapshotted** — moved to `versioned/{oldVersion}/`
2. **New version is written** as current/latest
3. **Manual edits are preserved** — markdown, owners, repository, styles, badges, attachments, diagrams, flows are carried over from the old version to the new one
4. **Schemas are updated** — the payload schema from the spec overwrites the old one

This means you can regenerate from specs without losing manual documentation. The `preserveExistingMessages` flag (default `true`) controls this; setting it to `false` overwrites everything.

### Changelogs

```typescript
interface Changelog {
	createdAt: Date | string;
	badges?: Badge[];
	markdown: string; // freeform MDX
}
```

Changelogs are stored alongside resources (e.g., `events/OrderCreated/changelog.md`). SDK: `writeChangelog`, `appendChangelog`, `getChangelog`, `rmChangelog`.

**Assessment:** catalog/v4 has `Version` on every message and `Changelog []Change` (with `Version`, `Date`, `Summary`), but:

- No versioning-on-re-run logic (no snapshot/preserve mechanism)
- No changelog as a separate document (it's an inline field, not a file)
- No semver range resolution in relationships
- No `versioned/` directory concept

---

## 6. MCP / AI Integration

### The MCP server (premium, Scale license)

Available at `https://your-catalog.com/docs/mcp/` when running in SSR mode. Supports scoped connections:

```
/docs/mcp/                         # whole catalog
/docs/mcp/domains/{id}             # domain-scoped (only reachable resources)
/docs/mcp/systems/{id}             # system-scoped
/docs/mcp/systems/{id}/{version}   # version-scoped
```

### 19 MCP tools

| Tool                                                  | What it does                                            |
| ----------------------------------------------------- | ------------------------------------------------------- |
| `getResources`                                        | Get events, services, commands, queries, flows, domains |
| `getResource`                                         | Get a specific resource by id and version               |
| `getMessagesProducedOrConsumedByResource`             | Messages a resource sends/receives                      |
| `getSchemaForResource`                                | Get OpenAPI/AsyncAPI schemas                            |
| `findResourcesByOwner`                                | Resources owned by a team or user                       |
| `getProducersOfMessage`                               | Services that produce a message                         |
| `getConsumersOfMessage`                               | Services that consume a message                         |
| `getC4Diagram`                                        | Get C4 diagram source                                   |
| `analyzeChangeImpact`                                 | Impact of changing a message                            |
| `explainBusinessFlow`                                 | Detailed flow information                               |
| `getTeams` / `getTeam`                                | Query teams                                             |
| `getUsers` / `getUser`                                | Query users                                             |
| `findMessageBySchemaId`                               | Find messages by schema identifiers                     |
| `explainUbiquitousLanguageTerms`                      | DDD ubiquitous language from domains                    |
| `getCustomDocs` / `searchCustomDocs` / `getCustomDoc` | Custom documentation                                    |

### 17 MCP resources

```
eventcatalog://all
eventcatalog://events        eventcatalog://commands      eventcatalog://queries
eventcatalog://agents        eventcatalog://adrs          eventcatalog://services
eventcatalog://systems       eventcatalog://channels      eventcatalog://entities
eventcatalog://diagrams      eventcatalog://containers    eventcatalog://data-products
eventcatalog://domains       eventcatalog://flows
eventcatalog://teams         eventcatalog://users
```

### Custom MCP tools

```javascript
// eventcatalog.chat.js
export const tools = {
	myCustomTool: {
		description: "My custom tool",
		parameters: z.object({ query: z.string() }),
		execute: async ({ query }) => {
			return { result: "..." };
		},
	},
};
```

### OAuth protection

Configurable via `mcp.auth` with JWKS endpoint, inline asymmetric public key, or symmetric shared secret.

**Assessment:** This is forward-thinking. The tool list is a blueprint for what AI agents want to query about an architecture. catalog/v4 has no MCP server. However, catalog/v4's `docserver` serves JSON APIs (OpenAPI, AsyncAPI, D2) that could be wrapped in an MCP server.

---

## 7. CLI, Linter, Connectors

### CLI (`@eventcatalog/cli`)

```bash
eventcatalog --dir ./my-catalog getEvent "OrderCreated" "1.0.0"
eventcatalog --dir ./my-catalog writeEvent '{...}'
eventcatalog export --all --output catalog.ec
```

Exposes SDK functions from the terminal.

### Linter (`@eventcatalog/linter`)

Validates frontmatter schemas using **Zod**. Checks:

- All required fields present
- All resource references (sends/receives pointers) resolve to existing resources
- Semver ranges are valid
- Configurable rules via `.eventcatalogrc.js`

Validated resource types: domains, services, events, commands, queries, channels, flows, entities, agents, containers, data products, diagrams, ADRs, users, teams.

### Connectors (`@eventcatalog/connectors`)

Syncs external systems into teams/users:

- **GitHub directory**: `githubDirectory({ org, teams, users, token })`
- **Microsoft Entra ID**: `microsoftEntraDirectory({ tenantId, clientId, clientSecret, groups })`
- **Custom**: `defineDirectorySource({ type, name, loadUsers(), loadTeams() })`

**Assessment:** The linter concept is valuable. catalog/v4 has `Validate()` which checks required fields, but EventCatalog's linter validates _references_ (do all `sends`/`receives` pointers resolve?) — a stronger guarantee. The connectors concept (syncing org structure) is out of scope for catalog/v4 but relevant for mesh federation.

---

## 8. What EventCatalog Gets Right (That We Should Learn From)

### 8.1 Versioning-with-preservation

EventCatalog's generator re-run logic is the gold standard: snapshot old versions, preserve manual edits, update schemas. catalog/v4 has no equivalent — rebuilding the catalog from Go types replaces everything. We need a "preserve manual annotations across regenerations" story.

**Actionable:** Consider a `//catalog:preserve` comment directive or an overlay mechanism in catalog/v4 that lets manual annotations survive catalog rebuilds.

### 8.2 Changelogs as first-class

EventCatalog stores changelogs as separate markdown files alongside each resource. catalog/v4 has `Changelog []Change` as an inline field — structurally weaker. Changelogs should be first-class documents, not array entries.

**Actionable:** Consider promoting changelogs to first-class in catalog/v4 — either as a `ChangeLog` type or as a separate file/resource.

### 8.3 The `x-eventcatalog-role: provider|client` pattern

This elegantly solves the "who owns this message?" problem in multi-service catalogs. Only the provider creates message docs; consumers get the relationship without duplication.

**Actionable:** catalog/v4's `Message` already has `Producers []ServiceID` and `Consumers []ServiceID`, which is the inverse approach (declare on the message, not the service). Both work, but EventCatalog's approach (declare on the service side) is more natural for code generation where each service generates its own spec.

### 8.4 Flows with branching

EventCatalog's `FlowStep` supports `next_step` and `next_steps` (plural) for branching paths and conditional flows. catalog/v4's `FlowStep` has `NextStep` and `NextSteps` — same structure. But EventCatalog's flows are richer in node types: `actor`, `custom`, `externalSystem`, plus conditional triggers on receives.

**Actionable:** catalog/v4 already has this structurally. Ensure the D2/visual exporters render branching correctly.

### 8.5 The SDK as the foundation

150+ functions over the file system. Read, write, delete, version, relate, graph-traverse, snapshot, diff. This makes every generator trivial to write.

**Actionable:** catalog/v4 doesn't need a 150-function SDK (it's Go, not JS, and the model is code-derived). But the `docserver` should expose a richer API surface for querying the built catalog — `getProducersOfMessage`, `getConsumersOfMessage`, `analyzeChangeImpact` would be valuable endpoints.

### 8.6 The MCP tool list as a blueprint

The 19 MCP tools are a requirements document for "what AI agents want to know about an architecture." This is directly applicable to catalog/v4.

**Actionable:** Design an MCP server for catalog/v4's docserver that exposes these tools. The catalog already has the data; the MCP server is the AI-facing API.

### 8.7 The linter with reference validation

Validating that all `sends`/`receives` pointers resolve to existing resources catches configuration errors at build time.

**Actionable:** catalog/v4's `Validate()` should validate cross-references (do all `Message.Channels` point to existing channels? do all `Domain.Services` point to existing services?).

### 8.8 Scoped servers per domain/system

EventCatalog lets AI agents connect to a scoped MCP endpoint that only sees resources reachable through a specific domain or system. This is the right granularity for autonomous agents.

**Actionable:** Consider scoped API endpoints in catalog/v4's docserver — `/docs/{domain}/api/catalog` returning only that domain's slice.

---

## 9. What catalog/v4 Gets Right (That EventCatalog Can't)

### 9.1 The code IS the contract

EventCatalog requires external specs (AsyncAPI/OpenAPI YAML files) that must be manually kept in sync with code. catalog/v4 derives schemas from Go types via reflection (`SchemaFromType[T]()`) — **zero drift by construction**. When you change a struct, the catalog updates on the next build. No spec file to forget to update.

This is the fundamental structural advantage. EventCatalog's generators mitigate the drift problem, but they depend on you having an accurate AsyncAPI/OpenAPI spec — which is itself a manual artifact.

### 9.2 Type safety

catalog/v4's types are Go structs. EventCatalog's types are YAML frontmatter parsed into TypeScript interfaces at build time. The Go approach catches errors at compile time; the YAML approach catches them at lint time.

### 9.3 Structural integration with the runtime

EventCatalog is documentation-only. It has no connection to running code. catalog/v4 sits in the same binary as the event store, the projections, the command/query dispatch. The Channel-to-Bus binding gap (from the previous proposal) is the last mile to connect documentation to runtime.

### 9.4 Time-travel (the ES advantage)

EventCatalog versions resources by snapshotting MDX files. catalog/v4 can rebuild the catalog from the event store at any point in time — every past state is reconstructable. This is the killer feature EventCatalog structurally cannot offer.

### 9.5 The exporter ecosystem

catalog/v4 already has AsyncAPI, OpenAPI, D2, eventcatalog.dev, and docserver exporters. EventCatalog _consumes_ AsyncAPI/OpenAPI but doesn't _produce_ them. catalog/v4 is a **producer** of contract formats; EventCatalog is a **consumer**. They're complementary.

---

## 10. Side-by-Side Comparison

| Dimension              | EventCatalog                                                           | go-cqrs-lite catalog/v4                                    |
| ---------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Language**           | TypeScript (Astro + React)                                             | Go                                                         |
| **Source of truth**    | External specs (AsyncAPI/OpenAPI YAML)                                 | Go types (reflection)                                      |
| **Drift risk**         | High (spec may diverge from code)                                      | Zero (code IS the spec)                                    |
| **Content model**      | Rich: 17 resource types                                                | Rich: 12 resource types                                    |
| **Versioning**         | Semver + snapshot + preserve-on-regenerate                             | Semver (inline field only)                                 |
| **Changelogs**         | First-class (separate markdown files)                                  | Inline field (`[]Change`)                                  |
| **SDK**                | 150+ functions                                                         | None (Go builders only)                                    |
| **Generators**         | External specs → MDX (AsyncAPI, OpenAPI, Schema Registry, EventBridge) | None (code → catalog is the generator)                     |
| **Exporters**          | None (consumer, not producer)                                          | AsyncAPI, OpenAPI, D2, eventcatalog.dev, docserver         |
| **CLI**                | Yes (`@eventcatalog/cli`)                                              | No                                                         |
| **Linter**             | Yes (Zod validation + reference checking)                              | `Validate()` (basic field checking)                        |
| **MCP server**         | Yes (19 tools, 17 resources, scoped servers)                           | No                                                         |
| **AI chat**            | Yes (premium)                                                          | No                                                         |
| **Visualizations**     | Node graphs (`@xyflow/react`), schema viewer, Mermaid, LikeC4          | D2 diagrams, docserver HTML UIs (Scalar, AsyncAPI Studio)  |
| **Teams/Users**        | Yes (with GitHub/Entra ID sync)                                        | Types exist (`Team`, `User`), no sync                      |
| **ADRs**               | Yes (with supersedes/amends chains)                                    | No                                                         |
| **Flows**              | Rich (branching, conditions, triggers)                                 | Present (`FlowStep` with `NextSteps`)                      |
| **Data Products**      | Yes (`inputs[]`, `outputs[]` with contracts)                           | Yes (`DataProduct` + `DataContract` + `DataProductOutput`) |
| **License**            | MIT core + Commercial for premium                                      | Apache 2.0 (fully open)                                    |
| **Deployment**         | Static site (trivial) or SSR (for AI)                                  | Go handler (embed in any HTTP server)                      |
| **Runtime connection** | None (documentation only)                                              | Same binary as event store + projections                   |
| **Time-travel**        | No (snapshots are manual)                                              | Yes (rebuild from event store at any point)                |

---

## 11. Recommendations

### What to adopt from EventCatalog

1. **Changelogs as first-class** — promote from inline `[]Change` to a proper `Changelog` concept in catalog/v4. The event-sourced advantage: changelogs can be auto-generated from the event store (every schema version bump is a changelog entry).

2. **Reference validation in `Validate()`** — validate that all cross-references resolve (Channels → existing Channels, Domain.Services → existing Services, etc.). Catches configuration errors at build time.

3. **The MCP tool list as a blueprint** — design an MCP server for catalog/v4's docserver. The 19 tools are a requirements document for "what AI agents want to query." The catalog already has the data.

4. **Scoped API endpoints** — `/api/catalog/{domain}` returning only that domain's slice. Matches EventCatalog's scoped MCP servers.

5. **The `provider`/`client` ownership pattern** — catalog/v4's `Producers`/`Consumers` on `Message` is the inverse approach and already works. But documenting this pattern (and validating that every message has at least one producer) would improve governance.

### What NOT to adopt

1. **The 150-function SDK** — catalog/v4 is Go, not JS. The model is code-derived, not file-system-manipulated. A Go SDK over MDX files makes no sense when the source of truth is Go types.

2. **External spec generators** — catalog/v4 doesn't need AsyncAPI/OpenAPI generators because it _produces_ AsyncAPI/OpenAPI, not consumes them. The direction is reversed.

3. **The MDX file system** — catalog/v4 builds an in-memory `*Catalog` from Go types. Storing it as MDX files would be a regression.

### The complementary relationship

EventCatalog and catalog/v4 are **complementary**, not competitive:

- **catalog/v4 produces** the contract documents (AsyncAPI, OpenAPI, D2, eventcatalog.dev MDX) from Go types.
- **EventCatalog consumes** those documents and renders a beautiful, navigable, AI-queryable documentation site.

The pipeline is: **Go types → catalog/v4 → eventcatalog.dev exporter → EventCatalog generators → MDX files → Astro build → documentation site.**

This pipeline already exists: catalog/v4 has an `eventcatalog` exporter package (`catalog/v4/eventcatalog/exporter.go`) that produces eventcatalog.dev MDX files. The EventCatalog AsyncAPI generator can also consume the AsyncAPI specs that catalog/v4's `asyncapi` exporter produces.

**The recommendation:** strengthen this pipeline. Make catalog/v4's eventcatalog exporter a first-class, well-documented output. Add a guide showing the full Go-types-to-EventCatalog-site flow. This gives consumers the best of both worlds: zero-drift code-derived contracts AND the richest event-driven documentation platform.
