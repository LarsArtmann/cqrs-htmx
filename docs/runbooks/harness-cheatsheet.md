# Harness cheat sheet — shared-tree session conventions

The one page a session needs for the recurring invocation shapes: e2e battery,
headless probes, Go caches, templ regen, and the commit-daemon dance. Each
block carries the exact command that worked last; when one rots, fix it here
in the same change.

## e2e browser battery

```bash
# 0. clear battery ports (stale servers from any session)
bash scripts/tools/e2e-kill-strays.sh

# 1. full chromium battery + JSON report for the ran-count gate
cd e2e && E2E_BROWSER_PATH=/nix/store/33pxss8h71cl7vmfpy21bidsw0lj1g8q-chromium-152.0.7977.82/bin/chromium \
  PLAYWRIGHT_JSON_OUTPUT_NAME=/tmp/e2e-report.json \
  ./node_modules/.bin/playwright test --project=chromium --reporter=json

# 2. the ran-count gate (fail loud on silently-skipped specs)
bash scripts/checks/check-e2e-ran-count.sh /tmp/e2e-report.json
```

Single spec: append the spec path (`tests/admin-behavior.spec.ts`) to the
playwright invocation. The chromium binary path above is the nix store path of
the battery browser; re-resolve with `ls /nix/store/*chromium*/bin/chromium`
when the store GC moved it.

## Computed-style probes (CSS engine claims)

The reusable position-area probe (the templ-components#101 investigation's
form) verifies any `position-area` spelling through the same `var()` route a
declarative CSS layer uses:

```bash
E2E_BROWSER_PATH=<chromium> node e2e/probes/position-area-probe/probe.mjs \
  'bottom span-inline-start' 'bottom span-x-start'
```

Rejected spellings compute to `none` (initial value), NOT a cascade fallback,
because the var() substitution fails at computed-value time. See the probe
header for adding candidate rows. Rule: engine-verify the VALUES, not just an
`@supports` gate.

## Go caches and toolchain

```bash
source scripts/lib/go-cache-env.sh   # BEFORE any go/nix heavy run
df -h /mnt/buildcache                # gotcha 12: the fill-drain cycle recurs; gates fail en masse when it is full
```

The script picks a writable GOCACHE (falls back to /tmp when buildcache is
low) and aligns the GOTOOLCHAIN floor. Never trust a battery that ran with a
full disk and no cache-env source.

## templ regeneration (canonical generator only)

```bash
nix build .#templGen && ./result/bin/templ generate -path <module> && rm -f result
```

Never the ambient templ binary (gotcha 10). CSS bundles rebuild in the SAME
change as any templ-components bump (`nix run .#build-adminui-css` and the
dashboardui twin).

## Committing on the shared tree

- The auto-commit daemon sweeps changed files roughly every 10 minutes; a
  phase boundary commits within ~2 minutes or the daemon carries it with a
  heuristic message.
- Plain `git commit -m` (staged index) passes the BuildFlow pre-commit hook;
  PATHSPEC commits (`git commit -- <path>`) trip the hook's roundtrip-clone
  step under concurrency (the nix-fmt:detect flake, seen 2026-11-16).
- The hook passing STANDALONE (`bash .githooks/pre-commit`) while the same
  commit fails via `git commit` = the roundtrip race, not a content problem;
  one plain-commit retry settles it.
- Never mutate the tree while a composite gate or battery runs; run them in a
  read-only window (research, reading, planning).
- One writer per worktree: check `git log -1 --format='%ci'` and file mtimes
  before touching a sibling session's module area.
