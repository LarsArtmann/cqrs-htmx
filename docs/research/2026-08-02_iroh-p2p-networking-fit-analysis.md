# Iroh (n0-computer) — P2P/QUIC Networking & cqrs-htmx Fit Analysis

**Date:** 2026-08-02
**Status:** Research synthesis (not adopted)
**Scope:** Capability mapping of [Iroh](https://github.com/n0-computer/iroh) against cqrs-htmx's architecture; recommendation for adoption posture
**Source:** [iroh.computer](https://iroh.computer), [docs.iroh.computer](https://docs.iroh.computer), [iroh-ffi](https://github.com/n0-computer/iroh-ffi)

---

## TL;DR

Iroh is a **Rust** peer-to-peer QUIC networking stack: dial-by-key identity, NAT traversal with relay fallback, and composable protocols (`iroh-blobs` for BLAKE3 content-addressed transfer, `iroh-gossip` for decentralized broadcast, `iroh-docs` for CRDT multi-writer KV sync). cqrs-htmx is a **Go, server-authoritative, HTTP/HTMX** CQRS/event-sourcing library. The two are philosophically opposite on the **write path** (event sourcing + Casbin authz = centralized trust; P2P CRDT merge = distributed trust), so Iroh is **not a core-dependency fit**. It is a fit for a few **opt-in extensions**, and the project's "never enforce defaults" library principle makes a clean additive adoption path possible.

---

## Table of Contents

- [1. What Iroh Is](#1-what-iroh-is)
- [2. cqrs-htmx's Current Networking/Sync Model](#2-cqrs-htmxs-current-networkingsync-model)
- [3. Capability → Fit Matrix](#3-capability--fit-matrix)
- [4. The Genuinely Interesting Opportunities](#4-the-genuinely-interesting-opportunities)
- [5. Why Adoption Is Taxed (Friction)](#5-why-adoption-is-taxed-friction)
- [6. Recommendation](#6-recommendation)
- [Appendix A: Iroh Protocol Detail](#appendix-a-iroh-protocol-detail)
- [Appendix B: Sources](#appendix-b-sources)

---

## 1. What Iroh Is

A modular P2P networking stack written in **Rust** by n0-computer. Core properties:

| Property | Detail |
| --- | --- |
| **Identity** | Dial by ed25519 public key (`NodeId` / `EndpointId`), not IP address. A `NodeAddr` bundles the key + relay URLs + direct addresses. |
| **Transport** | QUIC + TLS 1.3 — end-to-end encryption, authentication, stream multiplexing without head-of-line blocking. ALPN strings dispatch streams to protocol handlers via a `Router`. |
| **NAT traversal** | QAD (QUIC Address Discovery) probes → home relay selection → DNS-based or Mainline DHT peer lookup → direct QUIC hole punching (via `n0_nat_traversal` extension) → LAN/mDNS local connections → **relay fallback** if no direct path. Connections migrate across network changes without dropping. |
| **Relays** | Stateless, free community relays by default; self-hostable for production. Used for discovery + last-resort data relay. |
| **Swappable transports** | UDP default; Tor, Nym, or Bluetooth as alternatives. |
| **Platforms** | Windows, macOS, Linux, Android, iOS, embedded (ESP32), and WASM/browser. |
| **Protocols** | Composable, dispatched by ALPN. See [Appendix A](#appendix-a-iroh-protocol-detail). |

**Language bindings (via iroh-ffi / UniFFI):** Python, Swift, Kotlin/JVM, JavaScript/Node — these cover the **core 1.0 networking surface only**. A **community Go binding** ([decentral1se/iroh-go](https://git.coopcloud.tech/decentral1se/iroh-go)) exists but covers only the core surface, **not** the higher-level protocols (blobs/docs/gossip). The interesting protocols are **Rust-first** and not yet in scope for official FFI.

---

## 2. cqrs-htmx's Current Networking/Sync Model

Grounded in the actual codebase:

- **Server-authoritative event sourcing.** 22 events, 19 commands, fold functions, Casbin authz. All writes pass through command dispatch (validation + invariant enforcement + authz). Clients **cannot** append events directly.
- **SSE for server→client push.** `go-sse` v0.3.0 broadcaster/stream/replay. Per-tab `EventSource` in the browser. Single-node (the broadcaster is in-process).
- **Offline command queue (ADR-0029 + ADR-0040).** `sync-worker.js` (SharedWorker) + `sync-client.js` in the root module's `sync/` directory. The worker is a **coordinator, not a proxy**: it does NOT send HTTP requests (tabs do, via HTMX), does NOT own the SSE connection. It **persists queued commands to IndexedDB** so writes survive closed tabs/restarts, and tells tabs to retry on reconnect. Served via `cqrshtmx.SyncWorkerHandler()` / `cqrshtmx.SyncClientHandler()`. Current `syncVersion`: `1.3.0`.
  - **Key distinction:** the offline layer queues *writes* (command envelopes). It does **not** serve readable state offline — the UI cannot render stale projections when offline.
- **Opt-in `SnapshotConfig` (ADR-0041).** Zero-value = full-replay mode. `MemorySnapshotStore` is dev/test only. To enable: set `Store` + `Codec` + `Strategy` on config. Write path consults snapshot and loads only the tail via `LoadFromVersion`.
- **Multi-instance fanout:** Not currently solved in-process. The event bus + SSE broadcaster is single-node. Horizontally-scaled deployments would conventionally add Redis Pub/Sub or Kafka.
- **"Never enforce defaults" library principle.** No mandatory CSP/HSTS/CSRF — all opt-in. Auth strategies (TOTP/WebAuthn/OAuth2) are optional sub-modules via interfaces.

---

## 3. Capability → Fit Matrix

| Iroh capability | cqrs-htmx need | Fit | Notes |
| --- | --- | --- | --- |
| `iroh-gossip` (decentralized pub/sub by topic) | Multi-node SSE/event-bus replication across horizontally-scaled instances | **Strong** | Replaces a Redis/Kafka broker dependency for fanout; matches the existing `go-sse` broadcaster role but server-to-server. Epidemic broadcast trees (HyParView + PlumTree). |
| `iroh-blobs` (BLAKE3 content-addressed, resumable, verified transfer) | `SnapshotConfig` snapshot distribution; event-log segment shipping | **Medium** | Real use but speculative — current SQL snapshots work. BLAKE3 chunking maps well to snapshot segments. |
| `iroh-docs` (CRDT multi-writer KV sync via range-based set reconciliation) | Read-only projection replication to browser for true local-first UI | **Medium (read-only only)** | **Conflict on writes** — clients cannot append events; that bypasses command validation, authz, and invariants. |
| Dial-by-key / NAT traversal | Self-hosted/on-prem/edge deployments behind NAT (POS, IoT) | **Low/medium** | Niche; most consumers want a normal HTTP server. |
| `iroh-roq` / `iroh-live` (RTP-over-QUIC, Media over QUIC streaming) | — | **None** | No media use case. |

---

## 4. The Genuinely Interesting Opportunities

Ranked by signal-to-noise.

### 4.1 Broker-less server-to-server fanout (gossip) — **Highest value**

Today the event bus + SSE broadcaster is single-node. For multi-instance deployments you'd normally add Redis Pub/Sub or Kafka. `iroh-gossip` gives topic-based epidemic broadcast with **no broker** — a clean opt-in `EventBus` implementation, mirroring how `usermgmt/totp` etc. are optional sub-modules. No write-path changes; the worst case is it stays unused and respects the library principle.

### 4.2 Distributed snapshot store (blobs) — **Speculative**

`SnapshotConfig` is opt-in and `MemorySnapshotStore` is dev-only. A content-addressed, resumable, verified snapshot store shared across nodes is a legitimate design — BLAKE3 chunking maps well to snapshot segments, and verified streaming means a node can resume a partial snapshot download. Real but not urgently needed; current SQL snapshots work.

### 4.3 Local-first read projections (docs, read-only) — **Architecturally interesting**

An `iroh-docs`-replicated read-model would let the UI render stale-but-readable projections offline — closing the gap in the current offline layer (which queues writes but can't serve readable state offline). CRDT range-based set reconciliation converges across flaky links.

**Caveat:** this duplicates part of the IndexedDB/SSE sync stack, conflicts with "read model lives on server," and — critically — **writes must still route through the server's command pipeline** (validation, authz, invariants). Letting clients merge via CRDTs would defeat event sourcing's purpose.

---

## 5. Why Adoption Is Taxed (Friction)

Honest costs, not hand-waving:

1. **Language gap (the big one).** Iroh is Rust-first. The **official FFI covers Python/Swift/Kotlin/JS — not Go**. The Go binding (`decentral1se/iroh-go`) is community-maintained and covers only the **core 1.0 networking surface**, *not* blobs/docs/gossip — the actually-interesting protocols. Using the interesting parts means hand-rolling cgo FFI against `iroh-ffi` (C binding) or reimplementing.
2. **Build complexity.** The hermetic Nix build (`GOWORK=off`, vendored Go deps, the `govalid-generate` GOCACHE race saga) would need to compile/link a Rust native library. Significant friction against a currently clean 18-module Go workspace. cgo also breaks cross-compilation and complicates the devShell.
3. **Write-path conflict.** Event sourcing + Casbin = centralized trust. P2P CRDT merge = distributed trust. These are fundamentally different consistency models. Keep Iroh strictly on the *read/replication/transport* side; never on the write side.
4. **Already solved.** Offline command queueing already exists (IndexedDB-persisted, SharedWorker-coordinated). Iroh would be additive, not simplifying, for that use case.

---

## 6. Recommendation

**Do not adopt as a core dependency.** The architectural tension on the write path and the Go/FFI cost make it a poor core fit.

If pursued, the **one high-value, low-risk** experiment is an opt-in `iroh-gossip`-backed `event.Bus`/broadcaster as a new optional sub-module (like the auth modules) for broker-less multi-instance fanout — it respects the library principle, needs no write-path changes, and the worst case is it stays unused. Keep the CRDT/snapshot ideas in `ROADMAP.md` "raw ideas" rather than acting on them, pending either (a) official Go bindings covering the protocol layer, or (b) a concrete consumer request for broker-less multi-instance deployment.

**Decision triggers to revisit:**
- n0-computer ships official Go bindings for `iroh-gossip` (unblocks §4.1 with acceptable FFI cost).
- A consumer requests multi-instance deployment without an external broker.
- A consumer requests local-first offline-readable projections (re-evaluate §4.3 read-only path).

---

## Appendix A: Iroh Protocol Detail

### iroh-blobs — Content-addressed blob transfer & storage

- Blobs referenced by **BLAKE3 hash** (32-byte root hash from the BLAKE3 tree hash) — immutable identifier.
- Leverages BLAKE3 tree structure for **incremental verification / verified streaming**: each chunk's integrity is checked during transfer → resumable, seekable downloads.
- **Range requests** — fetch a verifiable contiguous subsequence by streaming only the needed BLAKE3 tree nodes.
- Chunk size defaults to 1 KiB (~6% metadata overhead), tunable without affecting the root hash.
- **Collections** — ordered sequences of hashes referencing other blobs (the only way to relate blobs).
- Pluggable stores: `MemStore` (in-memory) or `FsStore` (filesystem-backed, persistent).

### iroh-gossip — Broadcast / pub-sub

- Broadcast messages to subscribers of a **topic** without a central broker.
- Based on **epidemic broadcast trees**: [HyParView](https://asc.di.fct.unl.pt/~jleitao/pdf/dsn07-leitao.pdf) (partial view membership) + [PlumTree](https://asc.di.fct.unl.pt/~jleitao/pdf/srds07.pdf) (broadcast tree).
- Each node tracks a small set of neighbors; forwards messages probabilistically/redundantly to reach all interested peers despite churn.
- Centered around a `TopicId` (32-byte identifier, recommended derived from SHA-256 of a meaningful string).
- Scales to a few thousand peers per topic. Tradeoff vs centralized brokers (Kafka/MQTT): trades strict delivery guarantees for decentralization, simplicity, and resilience in dynamic environments.

### iroh-docs — Collaborative key-value sync (CRDTs)

- Eventually-consistent, multi-writer KV store built on CRDTs with an efficient sync protocol.
- Meta-protocol stacking on **iroh-blobs** (stores content bytes) + **iroh-gossip** (live sync notifications).
- **Document** (a.k.a. Replica): named, shared KV store; identity is a `NamespaceId` (public key gating write access).
- **Entry**: row identified by `(namespace, author, key)`; value is BLAKE3 hash of content + size + timestamp.
- **Author**: keypair signing entries; apps can create any number.
- **Ticket** (`DocTicket`): shareable string to import a document and start syncing.
- **Sync mechanism**: **range-based set reconciliation** — recursively partitioning entry sets and comparing fingerprints to detect disagreement ([Meyer 2022](https://arxiv.org/abs/2212.13567)). Fully-in-sync peers exchange a single fingerprint to confirm.
- Storage: generic interface with in-memory and persistent (file-based, [redb](https://github.com/cberner/redb)) implementations.
- Live events via `doc.subscribe()` — `InsertRemote`, `ContentReady`, sync progress.

### Other protocols

- **iroh-roq** — RTP over QUIC as an Iroh protocol.
- **iroh-live** — live streaming video/audio over Iroh using Media over QUIC (MoQ). Example: Callme (cross-platform audio streaming, Opus-encoded).
- **RPC** — remote procedure calls over Iroh connections.
- **iroh-content-discovery** — communicate with a content tracker for iroh-blobs.
- **iroh-ping** — minimal ping protocol.
- **Automerge** support — collaborative editing; "any kind of CRDT or OT sync protocol can be integrated."
- **iroh-willow** — implementation of the [Willow protocol](https://willowprotocol.org/).

---

## Appendix B: Sources

- [iroh.computer](https://iroh.computer) — project homepage, use cases, language bindings matrix.
- [docs.iroh.computer](https://docs.iroh.computer) — official documentation.
- [docs.iroh.computer/protocols/blobs](https://docs.iroh.computer/protocols/blobs/) — iroh-blobs: BLAKE3, collections, verified streaming.
- [docs.iroh.computer/connecting/gossip](https://docs.iroh.computer/connecting/gossip/) — iroh-gossip: epidemic broadcast trees.
- [docs.iroh.computer/protocols/kv-crdts](https://docs.iroh.computer/protocols/kv-crdts/) — iroh-docs: CRDT KV sync, set reconciliation.
- [docs.iroh.computer/protocols/streaming](https://docs.iroh.computer/protocols/streaming) — iroh-roq, iroh-live, MoQ streaming.
- [iroh.computer/proto](https://iroh.computer/proto) — protocol registry.
- [github.com/n0-computer/iroh](https://github.com/n0-computer/iroh) — source (Rust).
- [github.com/n0-computer/iroh-ffi](https://github.com/n0-computer/iroh-ffi) — official FFI bindings (Python, Swift, Kotlin, JS).
- [decentral1se/iroh-go](https://git.coopcloud.tech/decentral1se/iroh-go) — community Go binding (core surface only).
