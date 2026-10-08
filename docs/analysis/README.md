# docs/analysis — Static-Analysis Ratchet

Home of the `branching-flow` (go-design-smells) ratchet: a **committed SARIF
baseline** that freezes every adjudicated finding, plus the gate that fails
only on **NEW** findings.

## Files

| File                                                               | Purpose                                                                                                                                                                                                                                                                                                              |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`branching-flow-baseline.sarif`](./branching-flow-baseline.sarif) | Minified SARIF, one entry per frozen finding (229 at 2026-10-08 — re-pinned from 666 after the M17–M26 panel rewrites + dedup rounds dissolved 449 findings, +59 new branding-advice findings accepted under existing verdicts (see the amendment log)). |
| [`triage-decisions.md`](./triage-decisions.md)                     | The verdict ledger: every analyzer class, its count, its reject/fix verdict, and the reason. Read this before "fixing" a baseline finding.                                                                                                                                                                           |

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

Every verdict line prints the tool's Baseline counts (`+added, -removed,
~modified, =unchanged`) so a suspicious run is visible in the output. The
`rc 0` path carries a **zero-candidate false-green guard**: a committed
non-empty baseline with detected=0 (added+modified+unchanged) and removed>0
means the analyzer matched NOTHING (toolchain-floor misfire, SARIF parse
failure, or a full ratchet-down that never re-pinned) — the gate fails
loudly instead of passing.

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

## Why a baseline instead of fixing all 229

The 2026-10-05 full-analysis pass adjudicated the bulk as deliberate
(wire contracts, golden-pinned DTOs, config composition roots — see the
verdict ledger). A gate that failed on all of them would brick every commit;
freezing them and failing on NEW findings only converts the one-off firehose
into a durable ratchet without re-litigating a single verdict.

## Analyzer-subset measurement (R15/T14, 2026-10-08): keep all 14

Measured against the 229-finding baseline × the verdict ledger's per-class
dispositions, the signal fraction of the full analyzer set is ~2.6% (~6 of
229: the 4 high-severity STRONG_ID rows under TODO T6 plus borderlines).
Per-analyzer: STRONG_ID 76 (reject ~70 — wire contracts),
COMPOSITION_mixin 51 (reject — golden-pinned wire structs), PHANTOM
64 across TRANSPOSE/COLLISION (reject — branding advice),
DUPLICATE_TYPE 18 (rejected FP/intentional groups; its one realized fix —
the T3 `commandOptionApplier` drift guard — already landed),
large-struct 8 / DO 5 / IFACE_COMPLETE 3 / FLAG_PARAM 2 / BOOL_BLIND 1 /
base-naming 1 (all reject or FP).

Narrow-or-not verdict: **no narrowing.** The ratchet shape (fail on NEW
only, tolerate modified/removed) already neutralizes the noise — the 229
never block a commit — so a subset would buy nothing operationally while
removing the new-finding tripwires for the two classes that DID produce
real fixes (T3 split-brain guard from DUPLICATE_TYPE, T5 named-options
from FLAG_PARAM). The three-day churn measurement (663 → 666 → 229,
2026-10-05→08) also shows per-line baselines swing ~68% under normal
development as line-shift re-attribution — re-measure only when upstream
adds an analyzer; the ratchet absorbs new classes the same way.
