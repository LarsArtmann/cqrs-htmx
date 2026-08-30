# Drafts: upstream asks (go-cqrs-lite)

**Status:** PREPARED, awaiting approval to file (external comms).
**Prepared:** 2026-08-30.

## Draft 1 — tag `metaengine/projectionadapter/v4 v4.5.0` (THE blocker)

**Ask:** cut + push the next projectionadapter tag. cqrs-htmx's
systemadapter (and examples/system-demo) still carry a TEMPORARY local
replace to sibling master because master added the `OccurredAt` field on
`EventWithID` after `v4.4.1`, and our fold handlers consume it. Until
v4.5.0 exists, two modules cannot build hermetically GOWORK=off, and the
family's release-train check reports 1 replace-exemption + 5 sibling
train-lag axes that can never clear.

**Body:**

- master commit range containing the `OccurredAt` addition (verify at
  filing time via `git log -S OccurredAt -- metaengine/projectionadapter/`).
- Requested: `metaengine/projectionadapter/v4 v4.5.0` (semver-minor —
  additive field, backwards compatible for readers, and fold handlers
  compiled against master).
- Downstream impact if untagged: systemadapter's first family tag stays
  blocked (see cqrs-htmx Pareto plan M21); replace-exemption can never
  reach 0.

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
