# docs/analysis — Static-Analysis Ratchet

Home of the `branching-flow` (go-design-smells) ratchet: a **committed SARIF
baseline** that freezes every adjudicated finding, plus the gate that fails
only on **NEW** findings.

## Files

| File | Purpose |
|---|---|
| [`branching-flow-baseline.sarif`](./branching-flow-baseline.sarif) | Minified SARIF, one entry per frozen finding (659 at 2026-10-05, re-pinned from 663 after the first ratchet-down: 497 phantom, 76 strong-id, 51 mixins, 16 duplicate-types, 7 large-struct, 5 do, 2 flag-param, 3 iface-complete, 1 bool-blind, 1 base-naming). |
| [`triage-decisions.md`](./triage-decisions.md) | The verdict ledger: every analyzer class, its count, its reject/fix verdict, and the reason. Read this before "fixing" a baseline finding. |

## The gate

```sh
nix run .#check-branching-flow     # local: full gate, fails on NEW findings only
bash scripts/selftests/test-check-branching-flow.sh   # offline fixture self-test
```

Semantics (verified empirically 2026-10-05 vs branching-flow 0.2.0):

- `rc 0` — no new findings. **Modified** (line-shifted) and **removed**
  findings do NOT fail; removals are the ratchet improving.
- `rc 1` — new findings since the baseline. Fix them, or — if adjudicated as
  deliberate — refresh the baseline in the same commit.
- anything else — tool failure (most commonly the experimental `panic` linter
  failing to load packages because `go` on PATH is below the go.work floor;
  run via the flake app, which pins the 1.27 toolchain).

## Refreshing the baseline (ratchet down)

After landing changes that **remove** findings — or adjudicating new ones as
deliberate — re-pin:

```sh
source scripts/lib/go-cache-env.sh
branching-flow all . --format sarif --exclude-generated --no-emoji \
  --no-experimental-warn 2>/dev/null | jq -c . > docs/analysis/branching-flow-baseline.sarif
git diff --stat docs/analysis/branching-flow-baseline.sarif   # sanity: should SHRINK on fixes
git add docs/analysis/branching-flow-baseline.sarif           # commit together with the change
```

Rules:

- The baseline **must be minified** (`jq -c .`): the pretty form exceeds the
  1 MB `check-large-files` limit, and the compact `finding` format does not
  round-trip through the baseline differ (a byte-identical tree diffs as
  +610 false adds).
- The gate refuses an **uncommitted** baseline (false-green guard).
- A baseline that only GROWS must come with justification in the commit
  message — growth means new findings were accepted, which belongs in
  [`triage-decisions.md`](./triage-decisions.md).

## Why a baseline instead of fixing all 659

The 2026-10-05 full-analysis pass adjudicated the bulk as deliberate
(wire contracts, golden-pinned DTOs, config composition roots — see the
verdict ledger). A gate that failed on all of them would brick every commit;
freezing them and failing on NEW findings only converts the one-off firehose
into a durable ratchet without re-litigating a single verdict.
