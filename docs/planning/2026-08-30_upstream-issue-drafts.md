# Drafts: upstream asks (go-cqrs-lite)

**Status:** PREPARED, awaiting approval to file (external comms — filing any of these is a user gate).
**Prepared:** 2026-08-30. **Verified + completed to four drafts:** 2026-09-10 (projectionadapter max re-verified via live `git ls-remote`; postgres v4.2.0 breakage re-verified via `scripts/check-templates.sh`).

## Draft 1 — tag `metaengine/projectionadapter/v4 v4.5.0` (THE blocker)

**Ask:** cut + push the next projectionadapter tag. cqrs-htmx's
systemadapter (and examples/system-demo) still carry a TEMPORARY local
replace to sibling master because master added the `OccurredAt` field on
`EventWithID` after `v4.4.1`, and our fold handlers consume it. Until
v4.5.0 exists, two modules cannot build hermetically GOWORK=off, and the
family's release-train check reports 1 replace-exemption + 5 sibling
train-lag axes that can never clear.

**Body:**

- Verified 2026-09-10: max published tag is still `v4.4.1` (live
  `git ls-remote`); the `OccurredAt` addition lives on master only
  (re-verify at filing via `git log -S OccurredAt -- metaengine/projectionadapter/`).
- Requested: `metaengine/projectionadapter/v4 v4.5.0` (semver-minor —
  additive field, backwards compatible for readers, and fold handlers
  compiled against master).
- Downstream impact if untagged: systemadapter's first family tag stays
  blocked; the replace-exemption can never reach 0.

## Draft 2 — `stack`/`metaengine` integration friction (P3, informational)

**Ask (conversation-starter, no urgency):** the per-module release trains
mean downstream repos pin cross-module axes that drift independently
(cqrs-htmx carries explicit version pins + a V006 lockstep suppression for
exactly this). Would upstream consider publishing a compatibility matrix in
the repo README (module → latest published), or a CI step annotating stale
modules on master? This would remove the two `ls-remote`-based downstream
gates we maintain (`check-version-drift --strict`, `check-release-train`
tag cache).

**Body:** pointer to the two gate scripts + the runbook §7 stage table as
the concrete downstream artifacts of the drift.

## Draft 3 — multi-module consumer upgrade story (cqrs-upgrade)

**Ask:** upstream go-cqrs-lite publishes ~20 modules on independent
version trains. Consumers that use more than a handful (cqrs-htmx uses 16+)
have no first-class upgrade path: every train bump is a hand-built sweep of
per-module `go get @latest` + `go mod tidy` + hermetic build/vet across every
consumer module, plus an `ls-remote`-based "is this tag real" check because
partial sweeps leave the graph in a state plain `go build` cannot see.

**Proposal (any subset):**

1. A machine-readable "latest published per module" manifest (even just a
   committed JSON updated by CI on tag push) — this alone makes consumer
   sweeps one `jq` away instead of N `ls-remote` calls.
2. Document the supported upgrade pattern for multi-module consumers
   (workspace replaces vs per-module pins), including the phantom-require
   failure mode where one member's unpublished pin breaks module loading for
   the whole workspace graph.
3. Or adopt/bless a `cqrs-upgrade`-style tool (we run one internally; happy
   to contribute it) that bumps a whole train across a multi-module repo and
   verifies hermetically per module.

**Evidence:** cqrs-htmx's sweep recipe + gates:
`scripts/check-release-train.sh`, `scripts/check-version-drift.sh`, and the
2026-09-09 sweep record in our CHANGELOG (16 modules, 14 consumers, zero
failures — the recipe works, it is just N×M manual labor).

## Draft 4 — retract `stack/postgres v4.2.0` (broken in isolation)

**Ask:** retract `stack/postgres/v4 v4.2.0` (Go module retraction via the
go.mod `retract` directive + tag). v4.2.0 references unreleased sibling
modules and cannot resolve in an isolated (GOWORK=off) module build — any
consumer pinning it gets an unbuildable dependency graph, and the natural
workaround (pin v4.3.0+) only works if the consumer already knows v4.2.0 is
poison.

**Evidence:** our `scripts/check-templates.sh` documents the class and pins
`stack/postgres v4.3.0` with the comment "NOT v4.2.0: broken in isolation"
(added after the storage/v4 v4.7.0 incident where a retract already proved
the mechanism — that one worked exactly as intended). Also verified
2026-09-10 against the published tag.

**Note:** the same audit found `stack/mysql/v4` never got a v4.2.0 — no
action needed there (v4.1.0 is simply the latest), mentioned only so the
maintainer knows we checked the sibling case.
